package main

import (
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/executor"
)

// Executor represents a Test Exec
type Executor struct {
	WithTLS  bool   `json:"withtls,omitempty" yaml:"withtls,omitempty" mapstructure:"withtls"`
	Host     string `json:"host,omitempty" yaml:"host,omitempty" mapstructure:"host"`
	Port     string `json:"port,omitempty" yaml:"port,omitempty" mapstructure:"port"`
	User     string `json:"user,omitempty" yaml:"user,omitempty" mapstructure:"user"`
	Password string `json:"password,omitempty" yaml:"password,omitempty" mapstructure:"password"`
	To       string `json:"to,omitempty" yaml:"to,omitempty" mapstructure:"to"`
	From     string `json:"from,omitempty" yaml:"from,omitempty" mapstructure:"from"`
	Subject  string `json:"subject,omitempty" yaml:"subject,omitempty" mapstructure:"subject"`
	Body     string `json:"body,omitempty" yaml:"body,omitempty" mapstructure:"body"`
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	Err         string   `json:"error" yaml:"error"`
	TimeSeconds float64  `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
	TimeHuman   string   `json:"timeHuman,omitempty" yaml:"timeHuman,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := venom.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldBeEmpty"}}
}

func (e Executor) Manifest() venom.VenomModuleManifest {
	return venom.VenomModuleManifest{
		Name:    "smtp",
		Type:    "executor",
		Version: venom.Version,
	}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx venom.TestContext, step venom.TestStep) (venom.ExecutorResult, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, fmt.Errorf("unable to decode step: %v", err)
	}

	start := time.Now()

	result := Result{Executor: e}
	errs := e.sendEmail()
	if errs != nil {
		result.Err = errs.Error()
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = fmt.Sprintf("%s", elapsed)
	result.Executor.Password = "****hidden****" // do not output password

	return venom.Dump(result)
}

func (e *Executor) sendEmail() error {
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

	var c *smtp.Client
	if e.WithTLS {
		conn, errc := tls.Dial("tcp", servername, tlsconfig)
		if errc != nil {
			return fmt.Errorf("Error with c.Dial:%s", errc.Error())
		}

		var errn error
		c, errn = smtp.NewClient(conn, e.Host)
		if errn != nil {
			return fmt.Errorf("Error with c.NewClient:%s", errn.Error())
		}
	} else {
		var errd error
		c, errd = smtp.Dial(servername)
		if errd != nil {
			return fmt.Errorf("Error while smtp.Dial:%s", errd)
		}
		defer c.Close()
	}

	// Auth
	if e.User != "" && e.Password != "" {
		auth := smtp.PlainAuth("", e.User, e.Password, e.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("Error with c.Auth:%s", err.Error())
		}
	}

	if err := c.Mail(mailFrom.Address); err != nil {
		return fmt.Errorf("Error with c.Mail:%s", err.Error())
	}

	for _, toaddr := range strings.Split(mailTo.Address, ",") {
		if err := c.Rcpt(toaddr); err != nil {
			return fmt.Errorf("Error with toaddr:%s c.Rcpt:%s", toaddr, err.Error())
		}
	}

	if err := c.Rcpt(mailTo.Address); err != nil {
		return fmt.Errorf("Error with c.Rcpt:%s", err.Error())
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("Error with c.Data:%s", err.Error())
	}

	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("Error with c.Write:%s", err.Error())
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("Error with c.Close:%s", err.Error())
	}

	executor.Debugf("Mail sent to %s", mailTo.Address)
	c.Quit()

	return nil
}
