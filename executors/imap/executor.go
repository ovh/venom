package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/yesnault/go-imap/imap"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/executor"
)

var imapLogMask = imap.LogNone
var imapSafeLogMask = imap.LogNone

// Executor represents a Test Exec
type Executor struct {
	IMAPHost           string `json:"imaphost,omitempty" yaml:"imaphost,omitempty"`
	IMAPPort           string `json:"imapport,omitempty" yaml:"imapport,omitempty"`
	IMAPUser           string `json:"imapuser,omitempty" yaml:"imapuser,omitempty"`
	IMAPPassword       string `json:"imappassword,omitempty" yaml:"imappassword,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
	MBox               string `json:"mbox,omitempty" yaml:"mbox,omitempty"`
	MBoxOnSuccess      string `json:"mboxonsuccess,omitempty" yaml:"mboxonsuccess,omitempty"`
	DeleteOnSuccess    bool   `json:"deleteonsuccess,omitempty" yaml:"deleteonsuccess,omitempty"`
	SearchFrom         string `json:"searchfrom,omitempty" yaml:"searchfrom,omitempty"`
	SearchTo           string `json:"searchto,omitempty" yaml:"searchto,omitempty"`
	SearchSubject      string `json:"searchsubject,omitempty" yaml:"searchsubject,omitempty"`
	SearchBody         string `json:"searchbody,omitempty" yaml:"searchbody,omitempty"`
}

// Mail contains an analyzed mail
type Mail struct {
	From    string
	To      string
	Subject string
	UID     uint32
	Body    string
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	Err         string   `json:"error" yaml:"error"`
	Subject     string   `json:"subject,omitempty" yaml:"subject,omitempty"`
	Body        string   `json:"body,omitempty" yaml:"body,omitempty"`
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
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldNotBeEmpty"}}
}

func (e Executor) Manifest() venom.VenomModuleManifest {
	return venom.VenomModuleManifest{
		Name:    "imap",
		Type:    "executor",
		Version: venom.Version,
	}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx venom.TestContext, step venom.TestStep) (venom.ExecutorResult, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	start := time.Now()

	result := Result{Executor: e}
	find, errs := e.getMail()
	if errs != nil {
		result.Err = errs.Error()
	}
	if find != nil {
		result.Subject = find.Subject
		result.Body = find.Body
	} else if result.Err == "" {
		result.Err = "searched mail not found"
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = fmt.Sprintf("%s", elapsed)
	result.Executor.IMAPPassword = "****hidden****" // do not output password

	return venom.Dump(result)
}

func (e *Executor) getMail() (*Mail, error) {
	if e.SearchFrom == "" && e.SearchSubject == "" && e.SearchBody == "" && e.SearchTo == "" {
		return nil, fmt.Errorf("you have to use one of searchfrom, searchto, searchsubject or subjectbody parameters")
	}

	c, errc := connect(e.IMAPHost, e.IMAPPort, e.IMAPUser, e.IMAPPassword, e.InsecureSkipVerify)
	if errc != nil {
		return nil, fmt.Errorf("Error while connecting:%s", errc.Error())
	}
	defer c.Logout(5 * time.Second)

	var box string

	if e.MBox == "" {
		box = "INBOX"
	} else {
		box = e.MBox
	}

	count, err := queryCount(c, box)
	if err != nil {
		return nil, fmt.Errorf("Error while queryCount:%s", err.Error())
	}

	executor.Debugf("count messages:%d", count)

	if count == 0 {
		return nil, errors.New("No message to fetch")
	}

	messages, err := fetch(c, box, count)
	if err != nil {
		return nil, fmt.Errorf("Error while feching messages:%s", err.Error())
	}
	defer c.Close(false)

	for _, msg := range messages {
		m, erre := extract(msg)
		if erre != nil {
			executor.Warnf("Cannot extract the content of the mail: %s", erre)
			continue
		}

		found, errs := e.isSearched(m)
		if errs != nil {
			return nil, errs
		}

		if found {
			if e.DeleteOnSuccess {
				executor.Debugf("Delete message %s", m.UID)
				if err := m.delete(c); err != nil {
					return nil, err
				}
			} else if e.MBoxOnSuccess != "" {
				executor.Debugf("Move to %s", e.MBoxOnSuccess)
				if err := m.move(c, e.MBoxOnSuccess); err != nil {
					return nil, err
				}
			}
			return m, nil
		}
	}

	return nil, errors.New("Mail not found")
}

func (e *Executor) isSearched(m *Mail) (bool, error) {
	if e.SearchFrom != "" {
		ma, erra := regexp.MatchString(e.SearchFrom, m.From)
		if erra != nil || !ma {
			return false, erra
		}
	}
	if e.SearchTo != "" {
		mt, erra := regexp.MatchString(e.SearchTo, m.To)
		if erra != nil || !mt {
			return false, erra
		}
	}
	if e.SearchSubject != "" {
		mb, errb := regexp.MatchString(e.SearchSubject, m.Subject)
		if errb != nil || !mb {
			return false, errb
		}
	}
	if e.SearchBody != "" {
		mc, errc := regexp.MatchString(e.SearchBody, m.Body)
		if errc != nil || !mc {
			return false, errc
		}
	}
	return true, nil
}

func (m *Mail) move(c *imap.Client, mbox string) error {
	seq, _ := imap.NewSeqSet("")
	seq.AddNum(m.UID)

	if _, err := c.UIDMove(seq, mbox); err != nil {
		return fmt.Errorf("Error while move msg to %s: %v", mbox, err.Error())
	}
	return nil
}

func (m *Mail) delete(c *imap.Client) error {
	seq, _ := imap.NewSeqSet("")
	seq.AddNum(m.UID)

	if _, err := c.UIDStore(seq, "+FLAGS.SILENT", imap.NewFlagSet(`\Deleted`)); err != nil {
		return fmt.Errorf("Error while deleting msg, err: %s", err.Error())
	}
	if _, err := c.Expunge(nil); err != nil {
		return fmt.Errorf("Error while expunging messages: err: %s", err.Error())
	}
	return nil
}

func connect(host, port, imapUsername, imapPassword string, insecureSkipVerify bool) (*imap.Client, error) {
	if !strings.Contains(host, ":") {
		if port == "" {
			port = ":993"
		} else if port != "" && !strings.HasPrefix(port, ":") {
			port = ":" + port
		}
	}

	tlsconfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		ServerName:         host,
	}

	c, errd := imap.DialTLS(host+port, tlsconfig)
	if errd != nil {
		return nil, fmt.Errorf("Unable to dial: %v", errd)
	}

	if c.Caps["STARTTLS"] {
		if _, err := check(c.StartTLS(nil)); err != nil {
			return nil, fmt.Errorf("unable to start TLS: %v", err)
		}
	}

	c.SetLogMask(imapSafeLogMask)
	if _, err := check(c.Login(imapUsername, imapPassword)); err != nil {
		return nil, fmt.Errorf("Unable to login: %v", err)
	}
	c.SetLogMask(imapLogMask)

	return c, nil
}

func fetch(c *imap.Client, box string, nb uint32) ([]imap.Response, error) {
	executor.Debugf("call Select")
	if _, err := c.Select(box, false); err != nil {
		executor.Errorf("Error with select %s", err.Error())
		return []imap.Response{}, err
	}

	seqset, _ := imap.NewSeqSet("1:*")

	cmd, err := c.Fetch(seqset, "ENVELOPE", "RFC822.HEADER", "RFC822.TEXT", "UID")
	if err != nil {
		executor.Errorf("Error with fetch:%s", err)
		return []imap.Response{}, err
	}

	messages := []imap.Response{}
	for cmd.InProgress() {
		// Wait for the next response (no timeout)
		c.Recv(-1)

		// Process command data
		for _, rsp := range cmd.Data {
			messages = append(messages, *rsp)
		}
		cmd.Data = nil
		c.Data = nil
	}
	executor.Debugf("Nb messages fetch:%d", len(messages))
	return messages, nil
}

func queryCount(imapClient *imap.Client, box string) (uint32, error) {
	cmd, errc := check(imapClient.Status(box))
	if errc != nil {
		return 0, errc
	}

	var count uint32
	for _, result := range cmd.Data {
		mailboxStatus := result.MailboxStatus()
		if mailboxStatus != nil {
			count += mailboxStatus.Messages
		}
	}

	return count, nil
}

func check(cmd *imap.Command, erri error) (*imap.Command, error) {
	if erri != nil {
		return nil, erri
	}

	if _, err := cmd.Result(imap.OK); err != nil {
		return nil, err
	}

	return cmd, nil
}
