package tavern

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/venom"
	httputil "github.com/ovh/venom/executors/http"
)

// Name of executor
const Name = "tavern"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Headers represents header HTTP for Request
type Headers map[string]string

// BasicAuth contains basic authentication fields
type BasicAuth struct {
	User     string `json:"user" yaml:"user" mapstructure:"user"`
	Password string `json:"password" yaml:"password" mapstructure:"password"`
}

// TLS contains fields for TLS support
type TLS struct {
	ClientCert string `json:"client_cert" yaml:"client_cert" mapstructure:"client_cert"`
	ClientKey  string `json:"client_key" yaml:"client_key" mapstructure:"client_key"`
	RootCA     string `json:"root_ca" yaml:"root_ca" mapstructure:"root_ca"`
}

// Request describes HTTP request to perform for test
type Request struct {
	URL              string      `json:"url" yaml:"url"`
	Method           string      `json:"method" yaml:"method"`
	Headers          Headers     `json:"headers" yaml:"headers"`
	Body             string      `json:"body" yaml:"body"`
	File             string      `json:"file" yaml:"file"`
	JSON             interface{} `json:"json" yaml:"json"`
	MultipartForm    interface{} `json:"multipart_form" yaml:"multipart_form"`
	BasicAuth        BasicAuth   `json:"basic_auth" yaml:"basic_auth"`
	IgnoreVerifySSL  bool        `json:"ignore_verify_ssl" yaml:"ignore_verify_ssl" mapstructure:"ignore_verify_ssl"`
	Proxy            string      `json:"proxy" yaml:"proxy" mapstructure:"proxy"`
	Resolve          []string    `json:"resolve" yaml:"resolve" mapstructure:"resolve"`
	NoFollowRedirect bool        `json:"no_follow_redirect" yaml:"no_follow_redirect" mapstructure:"no_follow_redirect"`
	UnixSock         string      `json:"unix_sock" yaml:"unix_sock" mapstructure:"unix_sock"`
	TLS              TLS         `json:"tls" yaml:"tls"`
	SkipBody         bool        `json:"skip_body" yaml:"skip_body"`
	SkipHeaders      bool        `json:"skip_headers" yaml:"skip_headers"`
}

// Response describes expected response from server
type Response struct {
	StatusCode   int         `json:"statusCode,omitempty" yaml:"statusCode,omitempty"`
	Headers      Headers     `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body         string      `json:"body,omitempty" yaml:"body,omitempty"`
	JSON         interface{} `json:"json,omitempty" yaml:"json,omitempty"`
	JSONExcludes []string    `json:"json_excludes,omitempty" yaml:"json_excludes,omitempty"`
}

// Executor struct. Json and yaml descriptor are used for json output
type Executor struct {
	Request  Request  `json:"request" yaml:"request"`
	Response Response `json:"response" yaml:"response"`
}

// Result represents a step result. Json and yaml descriptor are used for json output
type Result struct {
	Actual      Response
	Expected    Response
	TimeSeconds float64 `json:"time_seconds,omitempty" yaml:"time_seconds,omitempty"`
	Err         string  `json:"err,omitempty" yaml:"err,omitempty"`
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for this executor
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result AssertResponse"}}
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	// transform step to Executor Instance
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	// dirty: mapstructure doesn't like decoding map[interface{}]interface{}, let's force manually
	e.Request.MultipartForm = step["multipart_form"]

	r := Result{}

	r.Expected = e.Response

	workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")

	req, err := e.getRequest(ctx, workdir)
	if err != nil {
		return nil, err
	}

	for k, v := range e.Request.Headers {
		req.Header.Set(k, v)
		if strings.ToLower(k) == "host" {
			req.Host = v
		}
	}

	var opts []func(*http.Transport) error
	opts = append(opts, httputil.WithProxyFromEnv())

	if e.Request.IgnoreVerifySSL {
		opts = append(opts, httputil.WithTLSInsecureSkipVerify(true))
	}

	if e.Request.TLS.ClientCert != "" {
		cert, err := tls.X509KeyPair([]byte(e.Request.TLS.ClientCert), []byte(e.Request.TLS.ClientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse x509 mTLS certificate or key: %s", err)
		}
		opts = append(opts, httputil.WithTLSClientAuth(cert))
	}
	if e.Request.TLS.RootCA != "" {
		opts = append(opts, httputil.WithTLSRootCA(ctx, []byte(e.Request.TLS.RootCA)))
	}

	tr, err := httputil.GetTransport(opts...)
	if err != nil {
		return nil, err
	}

	if len(e.Request.Resolve) > 0 && len(e.Request.UnixSock) > 0 {
		return nil, fmt.Errorf("you can't use resolve and unix_sock attributes in the same time")
	}

	if len(e.Request.Resolve) > 0 {
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// resolve can contains foo.com:443:127.0.0.1
			for _, r := range e.Request.Resolve {
				tuple := strings.Split(r, ":")
				if len(tuple) != 3 {
					return nil, fmt.Errorf("invalid value for resolve attribute: %v", e.Request.Resolve)
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
	} else if len(e.Request.UnixSock) > 0 {
		tr.DialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.DialUnix("unix", nil, &net.UnixAddr{
				Name: e.Request.UnixSock,
				Net:  "unix",
			})
		}
	}

	if len(e.Request.Proxy) > 0 {
		proxyURL, err := url.Parse(e.Request.Proxy)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{Transport: tr}
	if e.Request.NoFollowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()

	var bb []byte
	if resp.Body != nil {
		defer resp.Body.Close()

		if !e.Request.SkipBody {
			var errr error
			bb, errr = ioutil.ReadAll(resp.Body)
			if errr != nil {
				return nil, errr
			}
			r.Actual.Body = string(bb)

			var m interface{}
			decoder := json.NewDecoder(strings.NewReader(string(bb)))
			if err := decoder.Decode(&m); err == nil {
				r.Actual.JSON = m
			}
		}
	}

	if !e.Request.SkipHeaders {
		r.Actual.Headers = make(map[string]string)
		for k, v := range resp.Header {
			if strings.ToLower(k) == "set-cookie" {
				r.Actual.Headers[k] = strings.Join(v, "; ")
			} else {
				r.Actual.Headers[k] = v[0]
			}
		}
	}

	r.Actual.StatusCode = resp.StatusCode
	return r, nil
}

// getRequest returns the request correctly set for the current executor
func (e Executor) getRequest(ctx context.Context, workdir string) (*http.Request, error) {
	method := e.Request.Method
	if method == "" {
		method = "GET"
	}
	if (e.Request.Body != "" || e.Request.File != "" || e.Request.JSON != "") && e.Request.MultipartForm != nil {
		return nil, fmt.Errorf("can only use one of 'body', 'body_file' and 'multipart_form'")
	}
	body := &bytes.Buffer{}
	var writer *multipart.Writer
	if e.Request.Body != "" {
		body = bytes.NewBuffer([]byte(e.Request.Body))
	} else if e.Request.File != "" {
		path := filepath.Join(workdir, e.Request.File)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			temp, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, err
			}
			h := venom.AllVarsFromCtx(ctx)
			vars, _ := venom.DumpStringPreserveCase(h)
			stemp, err := interpolate.Do(string(temp), vars)
			if err != nil {
				return nil, fmt.Errorf("unable to interpolate file %s: %v", path, err)
			}
			body = bytes.NewBufferString(stemp)
		}
	} else if e.Request.JSON != "" {
		jsonBytes, err := json.Marshal(e.Request.JSON)
		if err != nil {
			return nil, fmt.Errorf("could not marshal bodyJson: %v", err)
		}
		body = bytes.NewBuffer(jsonBytes)
	} else if e.Request.MultipartForm != nil {
		form, ok := e.Request.MultipartForm.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%T 'multipart_form' should be a map", e.Request.MultipartForm)
		}
		writer = multipart.NewWriter(body)
		for key, v := range form {
			value, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("'multipart_form' should be a map with values as strings")
			}
			// Considering file will be prefixed by @ (since you could also post regular data in the body)
			if strings.HasPrefix(value, "@") {
				// todo: how can we be sure the @ is not the value we wanted to use ?
				if _, err := os.Stat(value[1:]); !os.IsNotExist(err) {
					part, err := writer.CreateFormFile(key, filepath.Base(value[1:]))
					if err != nil {
						return nil, err
					}
					if err := writeFile(part, value[1:]); err != nil {
						return nil, err
					}
					continue
				}
			}
			if err := writer.WriteField(key, value); err != nil {
				return nil, err
			}
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, e.Request.URL, body)
	if err != nil {
		return nil, err
	}

	if len(e.Request.BasicAuth.User) > 0 || len(e.Request.BasicAuth.Password) > 0 {
		req.SetBasicAuth(e.Request.BasicAuth.User, e.Request.BasicAuth.Password)
	}

	if writer != nil {
		req.Header.Set("Content-Type", writer.FormDataContentType())
	}
	return req, err
}

// writeFile writes the content of the file to an io.Writer
func writeFile(part io.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(part, file)
	return err
}
