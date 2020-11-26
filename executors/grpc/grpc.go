package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
)

// Name for test exec
const Name = "grpc"

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	URL                  string                 `json:"url" yaml:"url"`
	Service              string                 `json:"service" yaml:"service"`
	Method               string                 `json:"method" yaml:"method"`
	Plaintext            bool                   `json:"plaintext,omitempty" yaml:"plaintext,omitempty"`
	JSONDefaultFields    bool                   `json:"default_fields" yaml:"default_fields"`
	IncludeTextSeparator bool                   `json:"include_text_separator" yaml:"include_text_separator"`
	Data                 map[string]interface{} `json:"data" yaml:"data"`
	Headers              map[string]string      `json:"headers" yaml:"headers"`
	ConnectTimeout       *int64                 `json:"connect_timeout" yaml:"connect_timeout"`
}

// Result represents a step result
type Result struct {
	Systemout     string      `json:"systemout,omitempty" yaml:"systemout,omitempty"`
	SystemoutJSON interface{} `json:"systemoutjson,omitempty" yaml:"systemoutjson,omitempty"`
	Systemerr     string      `json:"systemerr,omitempty" yaml:"systemerr,omitempty"`
	SystemerrJSON interface{} `json:"systemerrjson,omitempty" yaml:"systemerrjson,omitempty"`
	Err           string      `json:"err,omitempty" yaml:"err,omitempty"`
	Code          string      `json:"code,omitempty" yaml:"code,omitempty"`
	TimeSeconds   float64     `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
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

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.code ShouldEqual 0"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
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

	// prepare data
	data, err := json.Marshal(e.Data)
	if err != nil {
		return nil, fmt.Errorf("runGrpcurl: Cannot marshal request data: %s", err)
	}

	result := Result{}
	start := time.Now()

	// prepare dial function
	dial := func() *grpc.ClientConn {
		dialTime := 10 * time.Second
		if e.ConnectTimeout != nil && *e.ConnectTimeout > 0 {
			dialTime = time.Duration(*e.ConnectTimeout * int64(time.Second))
		}
		ctx, cancel := context.WithTimeout(ctx, dialTime)
		defer cancel()
		var creds credentials.TransportCredentials
		cc, err := grpcurl.BlockingDial(ctx, "tcp", e.URL, creds)
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
	if cc == nil {
		return Result{Err: err.Error()}, fmt.Errorf("grpc dial error")
	}
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

	// prepare request and send
	in := bytes.NewReader(data)
	rf, formatter, err := grpcurl.RequestParserAndFormatterFor(
		grpcurl.FormatJSON,
		descSource,
		e.JSONDefaultFields,
		e.IncludeTextSeparator,
		in,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request parser and formatter %s", err)
	}

	// prepare custom handler to handle response
	handle := customHandler{
		formatter,
		&result,
		nil,
	}

	// invoke the gRPC
	err = grpcurl.InvokeRPC(ctx, descSource, cc, e.Service+"/"+e.Method, headers, &handle, rf.Next)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	if handle.err != nil {
		result.Err = handle.err.Error()
	}

	// parse stdout as JSON
	var outJSONArray []interface{}
	if err := json.Unmarshal([]byte(result.Systemout), &outJSONArray); err != nil {
		outJSONMap := map[string]interface{}{}
		if err2 := json.Unmarshal([]byte(result.Systemout), &outJSONMap); err2 == nil {
			result.SystemoutJSON = outJSONMap
		}
	} else {
		result.SystemoutJSON = outJSONArray
	}

	// parse stderr output as JSON
	var errJSONArray []interface{}
	if err := json.Unmarshal([]byte(result.Systemout), &errJSONArray); err != nil {
		errJSONMap := map[string]interface{}{}
		if err2 := json.Unmarshal([]byte(result.Systemout), &errJSONMap); err2 == nil {
			result.SystemoutJSON = errJSONMap
		}
	} else {
		result.SystemoutJSON = errJSONArray
	}

	return result, nil
}
