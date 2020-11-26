package ovhapi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/go-ovh/ovh"

	"github.com/ovh/venom"
)

// Name of executor
const Name = "ovhapi"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Headers represents header HTTP for Request
type Headers map[string]string

// Executor struct. Json and yaml descriptor are used for json output
type Executor struct {
	Method   string  `json:"method" yaml:"method"`
	NoAuth   bool    `json:"no_auth" yaml:"noAuth"`
	Path     string  `json:"path" yaml:"path"`
	Body     string  `json:"body" yaml:"body"`
	BodyFile string  `json:"bodyfile" yaml:"bodyfile"`
	Headers  Headers `json:"headers" yaml:"headers"`
}

// Result represents a step result. Json and yaml descriptor are used for json output
type Result struct {
	TimeSeconds float64     `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	StatusCode  int         `json:"statuscode,omitempty" yaml:"statuscode,omitempty"`
	Body        string      `json:"body,omitempty" yaml:"body,omitempty"`
	BodyJSON    interface{} `json:"bodyjson,omitempty" yaml:"bodyjson,omitempty"`
	Err         string      `json:"err,omitempty" yaml:"err,omitempty"`
	Headers     Headers     `json:"headers" yaml:"headers"`
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for this executor
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.statuscode ShouldEqual 200"}}
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	// transform step to Executor Instance
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	// Get context
	var endpoint = venom.StringVarFromCtx(ctx, "ovh.endpoint")
	var applicationKey = venom.StringVarFromCtx(ctx, "ovh.applicationKey")
	var applicationSecret = venom.StringVarFromCtx(ctx, "ovh.applicationSecret")
	var consumerKey = venom.StringVarFromCtx(ctx, "ovh.consumerKey")
	var insecure = venom.BoolVarFromCtx(ctx, "ovh.insecureTLS")
	var headers = venom.StringMapStringVarFromCtx(ctx, "ovh.headers")
	var workdir = venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")

	// set default values
	if e.Method == "" {
		e.Method = "GET"
	}

	// init result
	r := Result{}

	start := time.Now()
	// prepare ovh api client
	client, err := ovh.NewClient(
		endpoint,
		applicationKey,
		applicationSecret,
		consumerKey,
	)
	if err != nil {
		return nil, err
	}

	if insecure {
		client.Client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// get request body from file or from field
	requestBody, err := e.getRequestBody(workdir)
	if err != nil {
		return nil, err
	}

	req, err := client.NewRequest(e.Method, e.Path, requestBody, !e.NoAuth)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if e.Headers != nil {
		for key := range e.Headers {
			req.Header.Set(key, e.Headers[key])
		}
	}

	// do api call

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	r.Headers = Headers{}
	for k, v := range resp.Header {
		r.Headers[k] = strings.Join(v, ",")
	}

	res := new(interface{})
	if err = client.UnmarshalResponse(resp, res); err != nil {
		apiError, ok := err.(*ovh.APIError)
		if !ok {
			return nil, err
		}
		r.StatusCode = apiError.Code
		r.Err = apiError.Message
	} else {
		r.StatusCode = 200
	}

	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()

	// Add response to result body
	if res != nil {
		r.BodyJSON = *res
		bb, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}
		r.Body = string(bb)
	}

	return r, nil
}

func (e Executor) getRequestBody(workdir string) (res interface{}, err error) {
	var bytes []byte
	if e.Body != "" {
		bytes = []byte(e.Body)
	} else if e.BodyFile != "" {
		path := filepath.Join(workdir, e.BodyFile)
		if _, err = os.Stat(path); !os.IsNotExist(err) {
			bytes, err = ioutil.ReadFile(path)
			if err != nil {
				return nil, err
			}
		}
	}
	if len(bytes) > 0 {
		res = new(interface{})
		err = json.Unmarshal(bytes, res)
		return
	}
	return nil, nil
}
