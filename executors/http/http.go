package http

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
	"github.com/ovh/venom"
)

// Name of executor
const Name = "http"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Headers represents header HTTP for Request
type Headers map[string]string

// Executor struct. Json and yaml descriptor are used for json output
type Executor struct {
	Method            string      `json:"method" yaml:"method"`
	URL               string      `json:"url" yaml:"url"`
	Path              string      `json:"path" yaml:"path"`
	Body              string      `json:"body" yaml:"body"`
	BodyFile          string      `json:"bodyfile" yaml:"bodyfile"`
	MultipartForm     interface{} `json:"multipart_form" yaml:"multipart_form"`
	Headers           Headers     `json:"headers" yaml:"headers"`
	IgnoreVerifySSL   bool        `json:"ignore_verify_ssl" yaml:"ignore_verify_ssl" mapstructure:"ignore_verify_ssl"`
	BasicAuthUser     string      `json:"basic_auth_user" yaml:"basic_auth_user" mapstructure:"basic_auth_user"`
	BasicAuthPassword string      `json:"basic_auth_password" yaml:"basic_auth_password" mapstructure:"basic_auth_password"`
	SkipHeaders       bool        `json:"skip_headers" yaml:"skip_headers" mapstructure:"skip_headers"`
	SkipBody          bool        `json:"skip_body" yaml:"skip_body" mapstructure:"skip_body"`
	Proxy             string      `json:"proxy" yaml:"proxy" mapstructure:"proxy"`
	NoFollowRedirect  bool        `json:"no_follow_redirect" yaml:"no_follow_redirect" mapstructure:"no_follow_redirect"`
	UnixSock          string      `json:"unix_sock" yaml:"unix_sock" mapstructure:"unix_sock"`
}

// Result represents a step result. Json and yaml descriptor are used for json output
type Result struct {
	TimeSeconds float64     `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string      `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
	StatusCode  int         `json:"statuscode,omitempty" yaml:"statuscode,omitempty"`
	Body        string      `json:"body,omitempty" yaml:"body,omitempty"`
	BodyJSON    interface{} `json:"bodyjson,omitempty" yaml:"bodyjson,omitempty"`
	Headers     Headers     `json:"headers,omitempty" yaml:"headers,omitempty"`
	Err         string      `json:"err,omitempty" yaml:"err,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for this executor
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.statuscode ShouldEqual 200"}}
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, step venom.TestStep, workdir string) (interface{}, error) {
	// transform step to Executor Instance
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	// dirty: mapstructure doesn't like decoding map[interface{}]interface{}, let's force manually
	e.MultipartForm = step["multipart_form"]

	r := Result{}

	req, err := e.getRequest(workdir)
	if err != nil {
		return nil, err
	}

	for k, v := range e.Headers {
		req.Header.Set(k, v)
		if strings.ToLower(k) == "host" {
			req.Host = v
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: e.IgnoreVerifySSL},
		Proxy:           http.ProxyFromEnvironment,
	}

	if len(e.UnixSock) > 0 {
		tr.DialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.DialUnix("unix", nil, &net.UnixAddr{
				Name: e.UnixSock,
				Net:  "unix",
			})
		}
	}

	if len(e.Proxy) > 0 {
		proxyURL, err := url.Parse(e.Proxy)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	}
	client := &http.Client{Transport: tr}
	if e.NoFollowRedirect {
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
	r.TimeHuman = elapsed.String()

	var bb []byte
	if resp.Body != nil {
		defer resp.Body.Close()

		if !e.SkipBody {
			var errr error
			bb, errr = ioutil.ReadAll(resp.Body)
			if errr != nil {
				return nil, errr
			}
			r.Body = string(bb)

			bodyJSONArray := []interface{}{}
			if err := json.Unmarshal(bb, &bodyJSONArray); err != nil {
				bodyJSONMap := map[string]interface{}{}
				if err2 := json.Unmarshal(bb, &bodyJSONMap); err2 == nil {
					r.BodyJSON = bodyJSONMap
				}
			} else {
				r.BodyJSON = bodyJSONArray
			}
		}
	}

	if !e.SkipHeaders {
		r.Headers = make(map[string]string)
		for k, v := range resp.Header {
			if strings.ToLower(k) == "set-cookie" {
				r.Headers[k] = strings.Join(v[:], "; ")
			} else {
				r.Headers[k] = v[0]
			}
		}
	}

	r.StatusCode = resp.StatusCode
	return r, nil
}

// getRequest returns the request correctly set for the current executor
func (e Executor) getRequest(workdir string) (*http.Request, error) {
	path := fmt.Sprintf("%s%s", e.URL, e.Path)
	method := e.Method
	if method == "" {
		method = "GET"
	}
	if (e.Body != "" || e.BodyFile != "") && e.MultipartForm != nil {
		return nil, fmt.Errorf("Can only use one of 'body', 'body_file' and 'multipart_form'")
	}
	body := &bytes.Buffer{}
	var writer *multipart.Writer
	if e.Body != "" {
		body = bytes.NewBuffer([]byte(e.Body))
	} else if e.BodyFile != "" {
		path := filepath.Join(workdir, string(e.BodyFile))
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			temp, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(temp)
		}
	} else if e.MultipartForm != nil {
		form, ok := e.MultipartForm.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%T 'multipart_form' should be a map", e.MultipartForm)
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
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	if len(e.BasicAuthUser) > 0 || len(e.BasicAuthPassword) > 0 {
		req.SetBasicAuth(e.BasicAuthUser, e.BasicAuthPassword)
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
