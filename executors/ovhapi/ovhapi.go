package ovhapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
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
	Endpoint          string   `json:"endpoint" yaml:"endpoint"`
	ApplicationKey    string   `json:"applicationKey" yaml:"applicationKey"`
	ApplicationSecret string   `json:"applicationSecret" yaml:"applicationSecret"`
	ConsumerKey       string   `json:"consumerKey" yaml:"consumerKey"`
	NoAuth            *bool    `json:"noAuth" yaml:"noAuth"`
	Headers           Headers  `json:"headers" yaml:"headers"`
	Resolve           []string `json:"resolve" yaml:"resolve"`
	Proxy             string   `json:"proxy" yaml:"proxy"`
	TLSRootCA         string   `json:"tlsRootCA" yaml:"tlsRootCA"`

	Method   string `json:"method" yaml:"method"`
	Path     string `json:"path" yaml:"path"`
	Body     string `json:"body" yaml:"body"`
	BodyFile string `json:"bodyFile" yaml:"bodyFile"`
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
	return &venom.StepAssertions{Assertions: []venom.Assertion{"result.statuscode ShouldEqual 200"}}
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	// transform step to Executor Instance
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	// Get context
	if e.Endpoint == "" {
		e.Endpoint = venom.StringVarFromCtx(ctx, "ovh.endpoint")
	}
	if e.ApplicationKey == "" {
		e.ApplicationKey = venom.StringVarFromCtx(ctx, "ovh.applicationKey")
	}
	if e.ApplicationSecret == "" {
		e.ApplicationSecret = venom.StringVarFromCtx(ctx, "ovh.applicationSecret")
	}
	if e.ConsumerKey == "" {
		e.ConsumerKey = venom.StringVarFromCtx(ctx, "ovh.consumerKey")
	}
	if e.NoAuth == nil {
		noauth := venom.BoolVarFromCtx(ctx, "ovh.noAuth")
		e.NoAuth = &noauth
	}
	noAuth := *(e.NoAuth)
	if len(e.Resolve) == 0 {
		e.Resolve = venom.StringSliceVarFromCtx(ctx, "ovh.resolve")
	}
	if e.Proxy == "" {
		e.Proxy = venom.StringVarFromCtx(ctx, "ovh.proxy")
	}
	if e.TLSRootCA == "" {
		e.TLSRootCA = venom.StringVarFromCtx(ctx, "ovh.tlsRootCA")
	}

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
		e.Endpoint,
		e.ApplicationKey,
		e.ApplicationSecret,
		e.ConsumerKey,
	)
	if err != nil {
		return nil, err
	}

	var tr *http.Transport

	if e.TLSRootCA != "" {
		if tr == nil {
			tr = http.DefaultTransport.(*http.Transport).Clone()
		}

		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			venom.Warning(ctx, "http: tls: failed to load default system cert pool, fallback to an empty cert pool")
			caCertPool = x509.NewCertPool()
		}

		if ok := caCertPool.AppendCertsFromPEM([]byte(e.TLSRootCA)); !ok {
			return nil, errors.New("TLSRootCA: failed to add a certificate to the cert pool")
		}

		tr.TLSClientConfig.RootCAs = caCertPool
	}

	if len(e.Resolve) > 0 {
		if tr == nil {
			tr = http.DefaultTransport.(*http.Transport).Clone()
		}

		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// resolve can contains foo.com:443:127.0.0.1
			for _, r := range e.Resolve {
				tuple := strings.Split(r, ":")
				if len(tuple) != 3 {
					return nil, fmt.Errorf("invalid value for resolve attribute: %v", e.Resolve)
				}
				if addr == tuple[0]+":"+tuple[1] {
					addr = tuple[2] + ":" + tuple[1]
				}
			}

			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	if e.Proxy != "" {
		proxyURL, err := url.Parse(e.Proxy)
		if err != nil {
			return nil, err
		}

		if tr == nil {
			tr = http.DefaultTransport.(*http.Transport).Clone()
		}

		tr.Proxy = http.ProxyURL(proxyURL)
	}

	if tr != nil {
		client.Client.Transport = tr
	}

	// get request body from file or from field
	requestBody, err := e.getRequestBody(workdir)
	if err != nil {
		return nil, err
	}

	req, err := client.NewRequest(e.Method, e.Path, requestBody, !noAuth)
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

	if h := req.Header.Get("Host"); h != "" {
		req.Host = h
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
			bytes, err = os.ReadFile(path)
			if err != nil {
				return nil, err
			}
		}
	}
	if len(bytes) > 0 {
		res = new(interface{})
		err = venom.JSONUnmarshal(bytes, res)
		return
	}
	return nil, nil
}
