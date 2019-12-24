package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

// Name for test exec
const Name = "grpc"

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

type chunk map[string]interface{}

type chunkSender struct {
	data          io.Reader
	requestData   grpcurl.RequestSupplier
	requestParser grpcurl.RequestParser
	formatter     grpcurl.Formatter
}

// Executor represents a Test Exec
type Executor struct {
	Url                  string                 `json:"url" yaml:"url"`
	Service              string                 `json:"service" yaml:"service"`
	Method               string                 `json:"method" yaml:"method"`
	Stream               string                 `json:"stream,omitempty" yaml:"stream,omitempty"`
	Plaintext            bool                   `json:"plaintext,omitempty" yaml:"plaintext,omitempty"`
	JsonDefaultFields    bool                   `json:"default_fields" yaml:"default_fields"`
	IncludeTextSeparator bool                   `json:"include_text_separator" yaml:"include_text_separator"`
	Data                 map[string]interface{} `json:"data" yaml:"data"`
	Headers              map[string]string      `json:"headers" yaml:"headers"`
	ConnectTimeout       *int64                 `json:"connect_timeout" yaml:"connect_timeout"`
}

// Result represents a step result
type Result struct {
	Executor      Executor    `json:"executor,omitempty" yaml:"executor,omitempty"`
	Systemout     string      `json:"systemout,omitempty" yaml:"systemout,omitempty"`
	SystemoutJSON interface{} `json:"systemoutjson,omitempty" yaml:"systemoutjson,omitempty"`
	Systemerr     string      `json:"systemerr,omitempty" yaml:"systemerr,omitempty"`
	SystemerrJSON interface{} `json:"systemerrjson,omitempty" yaml:"systemerrjson,omitempty"`
	Err           string      `json:"err,omitempty" yaml:"err,omitempty"`
	Code          string      `json:"code,omitempty" yaml:"code,omitempty"`
	TimeSeconds   float64     `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman     string      `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
}

type customHandler struct {
	formatter grpcurl.Formatter
	target    *Result
	err       error
}

// OnResolveMethod is called with a descriptor of the method that is being invoked.
func (*customHandler) OnResolveMethod(m *desc.MethodDescriptor) {}

// OnSendHeaders is called with the request metadata that is being sent.
func (*customHandler) OnSendHeaders(metadata.MD) {}

// OnReceiveHeaders is called when response headers have been received.
func (*customHandler) OnReceiveHeaders(m metadata.MD) {}

// OnReceiveResponse is called for each response message received.
func (c *customHandler) OnReceiveResponse(msg proto.Message) {
	res, err := c.formatter(msg)
	if err != nil || c.err != nil {
		c.err = err
		return
	}
	c.target.Systemout = res
}

// OnReceiveTrailers is called when response trailers and final RPC status have been received.
func (c *customHandler) OnReceiveTrailers(stat *status.Status, met metadata.MD) {

	if err := stat.Err(); err != nil {
		c.target.Systemerr = err.Error()
	}
	c.target.Code = strconv.Itoa(int(uint32(stat.Code())))
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.code ShouldEqual 0"}}
}

// Run execute TestStep of type exec
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep, workdir string) (venom.ExecutorResult, error) {
	// decode test
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	// prepare headers
	headers := make([]string, len(e.Headers))
	for k, v := range e.Headers {
		headers = append(headers, fmt.Sprintf("%s: %s", k, v))
	}

	result := Result{Executor: e}
	start := time.Now()

	ctx := context.Background()

	// prepare dial function
	dial := func() *grpc.ClientConn {
		dialTime := 10 * time.Second
		if e.ConnectTimeout != nil && *e.ConnectTimeout > 0 {
			dialTime = time.Duration(*e.ConnectTimeout * int64(time.Second))
		}
		ctx, cancel := context.WithTimeout(ctx, dialTime)
		defer cancel()
		var creds credentials.TransportCredentials
		cc, err := grpcurl.BlockingDial(ctx, "tcp", e.Url, creds)
		if err != nil {
			return nil
		}
		return cc
	}

	var cc *grpc.ClientConn
	var descSource grpcurl.DescriptorSource
	var refClient *grpcreflect.Client
	md := grpcurl.MetadataFromHeaders(headers)
	refCtx := metadata.NewOutgoingContext(ctx, md)
	cc = dial()
	refClient = grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(cc))
	descSource = grpcurl.DescriptorSourceFromServer(ctx, refClient)

	// arrange for the RPCs to be cleanly shutdown
	defer func() {
		if refClient != nil {
			refClient.Reset()
			refClient = nil
		}
		if cc != nil {
			_ = cc.Close()
			cc = nil
		}
	}()

	// Invoke an RPC
	if cc == nil {
		cc = dial()
	}

	var in io.Reader

	if e.Stream == "" {
		// prepare data
		data, err := json.Marshal(e.Data)
		if err != nil {
			return nil, fmt.Errorf("runGrpcurl: Cannot marshal request data: %s\n", err)
		}

		// prepare request and send
		in = bytes.NewReader(data)

		rf, formatter, err := grpcurl.RequestParserAndFormatterFor(
			grpcurl.FormatJSON,
			descSource,
			e.JsonDefaultFields,
			e.IncludeTextSeparator,
			in,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to construct request parser and formatter %s", err)
		}

		handle, err := sendData(
			ctx,
			descSource,
			formatter,
			&result,
			cc,
			e.Service+"/"+e.Method,
			headers,
			rf.Next,
			start,
		)
		if err != nil {
			return nil, fmt.Errorf("error invoking method %s", err)
		}

		if handle.err != nil {
			result.Err = handle.err.Error()
		}

		result.SystemoutJSON, result.SystemerrJSON = parseOut(result)

		return executors.Dump(result)

	} else {

		// Opening json file
		// If proto message type is:
		//     message BodyChunk {
		//         bytes Foo = 1;
		//     }
		// the JSON file must be:
		// [
		//   {
		//     "Foo": "Chunk 1"
		//   },
		//   {
		//     "Foo": "Chunk 2"
		//   },
		// ]

		dat, err := ioutil.ReadFile(e.Stream)
		if err != nil {
			return nil, fmt.Errorf("runGrpcurl: file %s could not be open %s", e.Stream, err)
		}

		chunks := []chunk{}
		err = json.Unmarshal(dat, &chunks)
		if err != nil {
			return nil, fmt.Errorf("runGrpcurl: file content could not be read %s", err)
		}

		senders := make([]chunkSender, len(chunks)+1)

		for i, chunk := range chunks {
			data, err := json.Marshal(chunk)
			if err != nil {
				return nil, fmt.Errorf("runGrpcurl: Cannot marshal chunk #%d: %s", i, err)
			}

			in := bytes.NewReader(data)

			// prepare request and send
			senders[i] = chunkSender{
				data: in,
				// requestData: ,
			}

			rf, formatter, err := grpcurl.RequestParserAndFormatterFor(
				grpcurl.FormatJSON,
				descSource,
				e.JsonDefaultFields,
				e.IncludeTextSeparator,
				in,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to construct request parser and formatter %s", err)
			}

			senders[i] = chunkSender{
				data:          in,
				requestData:   rf.Next,
				requestParser: rf,
				formatter:     formatter,
			}

		}

		senders[len(chunks)] = chunkSender{
			data:          bytes.NewReader([]byte{}),
			requestData:   func(proto.Message) error { return io.EOF },
			requestParser: nil,
			formatter:     func(proto.Message) (string, error) { return "", nil },
		}

		for i, sender := range senders {
			handle, err := sendData(
				ctx,
				descSource,
				sender.formatter,
				&result,
				cc,
				e.Service+"/"+e.Method,
				headers,
				sender.requestData,
				start,
			)
			if err != nil {
				return nil, fmt.Errorf("#%d error invoking method %s", i, err)
			}

			if handle.err != nil {
				result.Err = handle.err.Error()
			}

			//result.SystemoutJSON, result.SystemoutJSON = parseOut(result)
		}

		result.SystemoutJSON, result.SystemerrJSON = parseOut(result)

	}

	return executors.Dump(result)
}

func sendData(
	ctx context.Context,
	descSource grpcurl.DescriptorSource,
	formatter grpcurl.Formatter,
	result *Result,
	cc *grpc.ClientConn,
	methodName string,
	headers []string,
	requestData grpcurl.RequestSupplier,
	start time.Time,
) (customHandler, error) {
	// prepare custom handler to handle response
	handle := customHandler{
		formatter,
		result,
		nil,
	}

	// invoke the gRPC
	err := grpcurl.InvokeRPC(ctx, descSource, cc, methodName, append(headers), &handle, requestData)

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = elapsed.String()

	return handle, err
}

func parseOut(result Result) (interface{}, interface{}) {
	var outJSON interface{}
	var errJSON interface{}

	// parse stdout as JSON
	var outJSONArray []interface{}
	if err := json.Unmarshal([]byte(result.Systemout), &outJSONArray); err != nil {
		outJSONMap := map[string]interface{}{}
		if err2 := json.Unmarshal([]byte(result.Systemout), &outJSONMap); err2 == nil {
			outJSON = outJSONMap
		}
	} else {
		outJSON = outJSONArray
	}

	// parse stderr output as JSON
	var errJSONArray []interface{}
	if err := json.Unmarshal([]byte(result.Systemout), &errJSONArray); err != nil {
		errJSONMap := map[string]interface{}{}
		if err2 := json.Unmarshal([]byte(result.Systemout), &errJSONMap); err2 == nil {
			errJSON = errJSONMap
		}
	} else {
		errJSON = errJSONArray
	}

	return outJSON, errJSON
}
