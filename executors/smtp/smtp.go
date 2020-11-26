package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/ovh/venom"
)

// Name for test smtp
const Name = "smtp"

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	WithTLS  bool   `json:"withtls,omitempty" yaml:"withtls,omitempty"`
	Host     string `json:"host,omitempty" yaml:"host,omitempty"`
	Port     string `json:"port,omitempty" yaml:"port,omitempty"`
	User     string `json:"user,omitempty" yaml:"user,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	To       string `json:"to,omitempty" yaml:"to,omitempty"`
	From     string `json:"from,omitempty" yaml:"from,omitempty"`
	Subject  string `json:"subject,omitempty" yaml:"subject,omitempty"`
	Body     string `json:"body,omitempty" yaml:"body,omitempty"`
}

// Result represents a step result
type Result struct {
	Err         string  `json:"error,omitempty" yaml:"error"`
	TimeSeconds float64 `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldBeEmpty"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	start := time.Now()

	result := Result{}
	err := e.sendEmail(ctx)
	if err != nil {
		result.Err = err.Error()
		return result, err
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	return result, nil
}

func (e *Executor) sendEmail(ctx context.Context) error {
	if e.To == "" {
		return fmt.Errorf("Invalid To")
	}
	if e.From == "" {
		return fmt.Errorf("Invalid From")
	}

	mailFrom := mail.Address{
		Name:    "",
		Address: e.From,
	}
	mailTo := mail.Address{
		Name:    "",
		Address: e.To,
	}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = e.From
	headers["To"] = e.To
	headers["Subject"] = e.Subject

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + e.Body

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         e.Host,
	}

	// Connect to the SMTP Server
	servername := fmt.Sprintf("%s:%s", e.Host, e.Port)
	venom.Info(ctx, "connecting to %s", servername)

	var c *smtp.Client
	if e.WithTLS {
		conn, err := tls.Dial("tcp", servername, tlsconfig)
		if err != nil {
			return errors.Wrapf(err, "tls dial error")
		}

		c, err = smtp.NewClient(conn, e.Host)
		if err != nil {
			return err
		}
	} else {
		var err error
		c, err = smtp.Dial(servername)
		if err != nil {
			return errors.Wrapf(err, "tls dial error")
		}
		defer c.Close()
	}

	// Auth
	if e.User != "" && e.Password != "" {
		auth := smtp.PlainAuth("", e.User, e.Password, e.Host)
		if err := c.Auth(auth); err != nil {
			return err
		}
	}

	if err := c.Mail(mailFrom.Address); err != nil {
		return err
	}

	for _, toaddr := range strings.Split(mailTo.Address, ",") {
		if err := c.Rcpt(toaddr); err != nil {
			return fmt.Errorf("%s: %v", toaddr, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	if _, err := w.Write([]byte(message)); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	venom.Info(ctx, "mail sent to %s", mailTo.Address)

	return c.Quit()
}
