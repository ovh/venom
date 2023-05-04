package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/cds/sdk/interpolate"
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
	Method            string            `json:"method" yaml:"method"`
	URL               string            `json:"url" yaml:"url"`
	Path              string            `json:"path" yaml:"path"`
	QueryParameters   map[string]string `json:"query_parameters" yaml:"query_parameters" mapstructure:"query_parameters"`
	Body              string            `json:"body" yaml:"body"`
	BodyFile          string            `json:"bodyfile" yaml:"bodyfile"`
	PreserveBodyFile  bool              `json:"preserve_bodyfile" yaml:"preserve_bodyfile" mapstructure:"preserve_bodyfile"`
	MultipartForm     interface{}       `json:"multipart_form" yaml:"multipart_form"`
	Headers           Headers           `json:"headers" yaml:"headers"`
	IgnoreVerifySSL   bool              `json:"ignore_verify_ssl" yaml:"ignore_verify_ssl" mapstructure:"ignore_verify_ssl"`
	BasicAuthUser     string            `json:"basic_auth_user" yaml:"basic_auth_user" mapstructure:"basic_auth_user"`
	BasicAuthPassword string            `json:"basic_auth_password" yaml:"basic_auth_password" mapstructure:"basic_auth_password"`
	SkipHeaders       bool              `json:"skip_headers" yaml:"skip_headers" mapstructure:"skip_headers"`
	SkipBody          bool              `json:"skip_body" yaml:"skip_body" mapstructure:"skip_body"`
	Proxy             string            `json:"proxy" yaml:"proxy" mapstructure:"proxy"`
	Resolve           []string          `json:"resolve" yaml:"resolve" mapstructure:"resolve"`
	NoFollowRedirect  bool              `json:"no_follow_redirect" yaml:"no_follow_redirect" mapstructure:"no_follow_redirect"`
	UnixSock          string            `json:"unix_sock" yaml:"unix_sock" mapstructure:"unix_sock"`
	TLSClientCert     string            `json:"tls_client_cert" yaml:"tls_client_cert" mapstructure:"tls_client_cert"`
	TLSClientKey      string            `json:"tls_client_key" yaml:"tls_client_key" mapstructure:"tls_client_key"`
	TLSRootCA         string            `json:"tls_root_ca" yaml:"tls_root_ca" mapstructure:"tls_root_ca"`
}

// Result represents a step result. Json and yaml descriptor are used for json output
type Result struct {
	TimeSeconds float64     `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	StatusCode  int         `json:"statuscode,omitempty" yaml:"statuscode,omitempty"`
	Request     HTTPRequest `json:"request,omitempty" yaml:"request,omitempty"`
	Body        string      `json:"body,omitempty" yaml:"body,omitempty"`
	BodyJSON    interface{} `json:"bodyjson,omitempty" yaml:"bodyjson,omitempty"`
	Headers     Headers     `json:"headers,omitempty" yaml:"headers,omitempty"`
	Err         string      `json:"err,omitempty" yaml:"err,omitempty"`
}

type HTTPRequest struct {
	Method   string      `json:"method,omitempty"`
	URL      string      `json:"url,omitempty"`
	Header   http.Header `json:"header,omitempty"`
	Body     string      `json:"body,omitempty"`
	Form     url.Values  `json:"form,omitempty"`
	PostForm url.Values  `json:"post_form,omitempty"`
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

	// dirty: mapstructure doesn't like decoding map[interface{}]interface{}, let's force manually
	e.MultipartForm = step["multipart_form"]

	r := Result{}

	workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")

	req, err := e.getRequest(ctx, workdir)
	if err != nil {
		return nil, err
	}

	for k, v := range e.Headers {
		req.Header.Set(k, v)
		if strings.ToLower(k) == "host" {
			req.Host = v
		}
	}

	var opts []func(*http.Transport) error
	opts = append(opts, WithProxyFromEnv())

	tlsOptions, err := e.TLSOptions(ctx)
	if err != nil {
		return nil, err
	}
	opts = append(opts, tlsOptions...)

	tr, err := GetTransport(opts...)
	if err != nil {
		return nil, err
	}

	if len(e.Resolve) > 0 && len(e.UnixSock) > 0 {
		return nil, fmt.Errorf("you can't use resolve and unix_sock attributes in the same time")
	}

	if len(e.Resolve) > 0 {
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
	} else if len(e.UnixSock) > 0 {
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

	cReq := req.Clone(ctx)
	r.Request.Method = cReq.Method
	r.Request.URL = req.URL.String()
	if cReq.Body != nil {
		body, err := cReq.GetBody()
		if err != nil {
			return nil, err
		}
		btes, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		defer cReq.Body.Close()
		r.Request.Body = string(btes)
	}
	r.Request.Header = cReq.Header
	r.Request.Form = cReq.Form
	r.Request.PostForm = cReq.PostForm

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

		if !e.SkipBody && isBodySupported(resp) {
			var errr error
			bb, errr = io.ReadAll(resp.Body)
			if errr != nil {
				return nil, errr
			}
			r.Body = string(bb)

			if isBodyJSONSupported(resp) {
				var m interface{}
				decoder := json.NewDecoder(strings.NewReader(string(bb)))
				decoder.UseNumber()
				if err := decoder.Decode(&m); err == nil {
					r.BodyJSON = m
				}
			}
		}
	}

	if !e.SkipHeaders {
		r.Headers = make(map[string]string)
		for k, v := range resp.Header {
			if strings.ToLower(k) == "set-cookie" {
				r.Headers[k] = strings.Join(v, "; ")
			} else {
				r.Headers[k] = v[0]
			}
		}
	}

	requestContentType := r.Request.Header.Get("Content-Type")
	// if PreserveBodyFile == true, the body is not interpolated.
	// So, no need to keep it in request here (to re-inject it in vars)
	// this will avoid to be interpolated after in vars too.
	if e.PreserveBodyFile || !isContentTypeSupported(requestContentType) {
		r.Request.Body = ""
	}

	r.StatusCode = resp.StatusCode
	return r, nil
}

// getRequest returns the request correctly set for the current executor
func (e Executor) getRequest(ctx context.Context, workdir string) (*http.Request, error) {
	path := fmt.Sprintf("%s%s", e.URL, e.Path)
	method := e.Method
	if method == "" {
		method = "GET"
	}
	if (e.Body != "" || e.BodyFile != "") && e.MultipartForm != nil {
		return nil, fmt.Errorf("can only use one of 'body', 'body_file' and 'multipart_form'")
	}

	body := &bytes.Buffer{}
	var writer *multipart.Writer
	if e.Body != "" {
		body = bytes.NewBuffer([]byte(e.Body))
	} else if e.BodyFile != "" {
		bodyfilePath := e.BodyFile
		if !filepath.IsAbs(e.BodyFile) {
			// Only join with the workdir with relative path
			bodyfilePath = filepath.Join(workdir, e.BodyFile)
		}
		if _, err := os.Stat(bodyfilePath); !os.IsNotExist(err) {
			temp, err := os.ReadFile(bodyfilePath)
			if err != nil {
				return nil, err
			}
			if e.PreserveBodyFile {
				body = bytes.NewBuffer(temp)
			} else {
				h := venom.AllVarsFromCtx(ctx)
				vars, _ := venom.DumpStringPreserveCase(h)
				str := string(temp)
				upperlimit := len(vars)
				counter := 0
				for {
					if !strings.Contains(str, "{{.") {
						break
					}
					stemp, err := interpolate.Do(str, vars)
					if err != nil {
						return nil, fmt.Errorf("unable to interpolate file %s: %v", path, err)
					}
					if strings.Compare(str, stemp) == 0 && counter > upperlimit {
						r, _ := regexp.Compile(`{{\..*}}`)
						return nil, fmt.Errorf("unable to interpolate file due to unresolved variables %s", strings.Join(r.FindAllString(str, -1), ","))
					}
					str = stemp
					counter++
				}
				body = bytes.NewBufferString(str)
			}
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

	if len(e.QueryParameters) > 0 {
		baseURL, err := url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("unable to parse url %s: %v", path, err)
		}
		params := url.Values{}
		for k, v := range e.QueryParameters {
			params.Add(k, v)
		}
		baseURL.RawQuery = params.Encode()
		path = baseURL.String()
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

func parseContentType(contentType string) string {
	parsed, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return contentType
	}
	return parsed
}

// given https://developer.mozilla.org/fr/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types
func isBodySupported(resp *http.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	return isContentTypeSupported(contentType)
}

func isContentTypeSupported(contentType string) bool {
	contentType = parseContentType(contentType)
	switch {
	case strings.HasSuffix(contentType, "+json"):
		return true
	case strings.HasPrefix(contentType, "image/"), strings.HasPrefix(contentType, "audio/"), strings.HasPrefix(contentType, "video/"),
		strings.HasPrefix(contentType, "font/"), strings.HasPrefix(contentType, "application/vnd."):
		return false
	case strings.HasPrefix(contentType, "application/"):
		x := strings.SplitN(contentType, "/", 2)[1]
		switch x {
		case "octet-stream", "x-abiword", "vnd.amazon.ebook", "x-bzip", "x-bzip2", "x-csh", "msword", "epub+zip", "java-archive", "ogg", "pdf",
			"x-rar-compressed", "rtf", "x-sh", "x-shockwave-flash", "x-tar", "zip", "x-7z-compressed":
			return false
		}
	case strings.Contains(contentType, "multipart/form-data"):
		return false
	}
	return true
}

func isBodyJSONSupported(resp *http.Response) bool {
	contentType := parseContentType(resp.Header.Get("Content-Type"))
	return strings.Contains(contentType, "application/json") || strings.HasSuffix(contentType, "+json")
}

func (e Executor) TLSOptions(ctx context.Context) ([]func(*http.Transport) error, error) {
	var opts []func(*http.Transport) error

	if e.IgnoreVerifySSL {
		opts = append(opts, WithTLSInsecureSkipVerify(true))
	}

	workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")

	if e.TLSRootCA != "" {
		TLSRootCAFilepath := filepath.Join(workdir, e.TLSRootCA)
		var TLSRootCA []byte
		if _, err := os.Stat(TLSRootCAFilepath); err == nil {
			TLSRootCA, err = os.ReadFile(TLSRootCAFilepath)
			if err != nil {
				return nil, fmt.Errorf("unable to read TLSRootCA from file %s", TLSRootCAFilepath)
			}
		} else {
			TLSRootCA = []byte(e.TLSRootCA)
		}
		opts = append(opts, WithTLSRootCA(ctx, TLSRootCA))
	}

	var TLSClientCert, TLSClientKey []byte
	if e.TLSClientCert != "" {
		TLSClientCertFilepath := filepath.Join(workdir, e.TLSClientCert)
		if _, err := os.Stat(TLSClientCertFilepath); err == nil {
			TLSClientCert, err = os.ReadFile(TLSClientCertFilepath)
			if err != nil {
				return nil, fmt.Errorf("unable to read TLSClientCert from file %s", TLSClientCertFilepath)
			}
		} else {
			TLSClientCert = []byte(e.TLSClientCert)
		}
	}

	if e.TLSClientKey != "" {
		TLSClientKeyFilepath := filepath.Join(workdir, e.TLSClientKey)
		if _, err := os.Stat(TLSClientKeyFilepath); err == nil {
			TLSClientKey, err = os.ReadFile(TLSClientKeyFilepath)
			if err != nil {
				return nil, fmt.Errorf("unable to read TLSClientKey from file %s", TLSClientKeyFilepath)
			}
		} else {
			TLSClientKey = []byte(e.TLSClientKey)
		}
	}

	if len(TLSClientCert) > 0 && len(TLSClientKey) > 0 {
		cert, err := tls.X509KeyPair(TLSClientCert, TLSClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse x509 mTLS certificate or key: %s", err)
		}
		opts = append(opts, WithTLSClientAuth(cert))
	}

	return opts, nil
}
