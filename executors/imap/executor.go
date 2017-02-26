package imap

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"
	"github.com/yesnault/go-imap/imap"

	"github.com/runabove/venom"
)

// Name for test imap
const Name = "imap"

var imapLogMask = imap.LogNone
var imapSafeLogMask = imap.LogNone

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	IMAPHost      string `json:"imaphost,omitempty" yaml:"imaphost,omitempty"`
	IMAPUser      string `json:"imapuser,omitempty" yaml:"imapuser,omitempty"`
	IMAPPassword  string `json:"imappassword,omitempty" yaml:"imappassword,omitempty"`
	MBox          string `json:"mbox,omitempty" yaml:"mbox,omitempty"`
	MBoxIfSuccess string `json:"mboxifsuccess,omitempty" yaml:"mboxifsuccess,omitempty"`
	SearchFrom    string `json:"searchfrom,omitempty" yaml:"searchfrom,omitempty"`
	SearchSubject string `json:"searchsubject,omitempty" yaml:"searchsubject,omitempty"`
}

// Mail contains an analyzed mail
type Mail struct {
	From    string
	Subject string
	UID     uint32
	Date    time.Time
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

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.err ShouldNotExist"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx context.Context, l *log.Entry, aliases venom.Aliases, step venom.TestStep) (venom.ExecutorResult, error) {

	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}

	start := time.Now()

	if l.Level == log.DebugLevel {
		imapLogMask = imap.LogConn | imap.LogState | imap.LogCmd
		imapSafeLogMask = imap.LogConn | imap.LogState
	}

	result := Result{Executor: t}
	find, errs := t.getMail(l)
	if errs != nil {
		result.Err = errs.Error()
	}
	if find != nil {
		result.Subject = find.Subject
		result.Body = find.Body
	} else {
		result.Err = "searched mail not found"
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = fmt.Sprintf("%s", elapsed)
	result.Executor.IMAPPassword = "****hide****" // do not output password

	return dump.ToMap(result, dump.WithDefaultLowerCaseFormatter())
}

func (e *Executor) getMail(l *log.Entry) (*Mail, error) {

	if e.SearchFrom == "" && e.SearchSubject == "" {
		return nil, fmt.Errorf("You have to use searchfrom and/or searchsubject")
	}

	c, errc := connect(e.IMAPHost, e.IMAPUser, e.IMAPPassword)
	if errc != nil {
		return nil, fmt.Errorf("Error while connecting:%s", errc.Error())
	}

	var box string

	if e.MBox == "" {
		box = "INBOX"
	}

	count, err := queryCount(c, box)
	if err != nil {
		disconnect(c)
		return nil, fmt.Errorf("Error while queryCount:%s", err.Error())
	}

	l.Debugf("count messages:%d", count)

	if count == 0 {
		return nil, fmt.Errorf("No message to fetch")
	}

	messages, err := fetch(c, box, count, l)
	if err != nil {
		disconnect(c)
		return nil, fmt.Errorf("Error while feching messages:%s", err.Error())
	}

	for _, msg := range messages {
		m, erre := extract(msg, l)
		if erre != nil {
			return nil, erre
		}

		if e.isSearched(m) {
			if e.MBoxIfSuccess != "" {
				l.Debugf("Move to %s", e.MBoxIfSuccess)
				if err := m.move(c, e.MBoxIfSuccess); err != nil {
					return nil, err
				}
			}
			return m, nil
		}
	}

	return nil, fmt.Errorf("Mail not found")
}

func (e *Executor) isSearched(m *Mail) bool {
	if e.SearchFrom != "" && e.SearchSubject != "" {
		return strings.Contains(m.From, e.SearchFrom) ||
			strings.Contains(m.Subject, e.SearchSubject)
	}
	if e.SearchFrom != "" {
		return strings.Contains(m.From, e.SearchFrom)
	}
	if e.SearchSubject != "" {
		return strings.Contains(m.Subject, e.SearchSubject)
	}

	return false
}

func (m *Mail) move(c *imap.Client, mbox string) error {
	seq, _ := imap.NewSeqSet("")
	seq.AddNum(m.UID)

	if _, err := c.UIDMove(seq, mbox); err != nil {
		return fmt.Errorf("Error while move msg to %s, err:%s", mbox, err.Error())
	}
	return nil
}

func connect(host, imapUsername, imapPassword string) (*imap.Client, error) {

	c, errd := imap.DialTLS(host+":993", nil)
	if errd != nil {
		return nil, fmt.Errorf("Unable to dial: %s", errd)
	}

	if c.Caps["STARTTLS"] {
		if _, err := check(c.StartTLS(nil)); err != nil {
			return nil, fmt.Errorf("Unable to start TLS: %s\n", err)
		}
	}

	c.SetLogMask(imapSafeLogMask)
	if _, err := check(c.Login(imapUsername, imapPassword)); err != nil {
		return nil, fmt.Errorf("Unable to login: %s", err)
	}
	c.SetLogMask(imapLogMask)

	return c, nil
}

func fetch(c *imap.Client, box string, nb uint32, l *log.Entry) ([]imap.Response, error) {
	l.Debugf("call Select")
	if _, err := c.Select(box, true); err != nil {
		l.Errorf("Error with select %s", err.Error())
		return []imap.Response{}, err
	}

	seqset, _ := imap.NewSeqSet("1:*")

	cmd, err := c.Fetch(seqset, "ENVELOPE", "RFC822.HEADER", "RFC822.TEXT", "UID")
	if err != nil {
		l.Errorf("Error with fetch:%s", err)
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
	l.Debugf("Nb messages fetch:%d", len(messages))
	return messages, nil
}

func disconnect(c *imap.Client) {
	if c != nil {
		c.Close(false)
	}
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
