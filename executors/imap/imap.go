package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	"github.com/pkg/errors"
	"github.com/yesnault/go-imap/imap"
)

const (
	// Name for test imap
	Name = "imap"
	// imapClientTimeout represents the timeout for the IMAP client and, as a result, the timeout for the testcase
	imapClientTimeout = 5 * time.Second
)

var (
	imapLogMask     = imap.LogNone
	imapSafeLogMask = imap.LogNone

	errMailNotFound = errors.New("mail not found")
	errEmptyMailbox = errors.New("empty mailbox")
)

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

type CommandName string

const (
	// CommandAppend creates a new message at the end of a specified mailbox.
	CommandAppend CommandName = "append"
	// CommandCreate creates a new mailbox.
	CommandCreate CommandName = "create"
	// CommandClear permanently removes all messages from given mailboxes.
	CommandClear CommandName = "clear"
	// CommandDelete permanently removes a message retrieved through Search field
	CommandDelete CommandName = "delete"
	// CommandFetch retrieves a mail through Search field (no args)
	CommandFetch CommandName = "fetch"
	// CommandFlag adds, removes or sets flags to a message retrieved through Search field
	CommandFlag CommandName = "flag"
	// CommandMove moves a message retrieved through Search field from one mailbox to another.
	CommandMove CommandName = "move"
)

// commandAppendArgs represents the arguments to the append command.
// from and mailbox are mandatory fields
type commandAppendArgs struct {
	mailbox string
	from    string
	to      string
	subject string
	body    string
	flags   []string
}

func (a *commandAppendArgs) initFrom(args map[string]any) {
	if v, ok := args["mailbox"]; ok {
		a.mailbox = v.(string)
	}
	if v, ok := args["from"]; ok {
		a.from = v.(string)
	}
	if v, ok := args["to"]; ok {
		a.to = v.(string)
	}
	if v, ok := args["subject"]; ok {
		a.subject = v.(string)
	}
	if v, ok := args["body"]; ok {
		a.body = v.(string)
	}
	if v, ok := args["flags"]; ok {
		flagsAny := v.([]any)
		flagsStr := make([]string, len(flagsAny))
		for i, mailbox := range flagsAny {
			flagsStr[i] = mailbox.(string)
		}
		a.flags = flagsStr
	}
}

func (a *commandAppendArgs) isAnyMandatoryFieldEmpty() bool {
	return a.mailbox == "" || a.from == ""
}

// commandCreateArgs represents the arguments to the create command
type commandCreateArgs struct {
	// The name of the mailbox to create
	mailbox string
}

func (a *commandCreateArgs) initFrom(args map[string]any) {
	if v, ok := args["mailbox"]; ok {
		a.mailbox = v.(string)
	}
}

func (a *commandCreateArgs) isAnyMandatoryFieldEmpty() bool {
	return a.mailbox == ""
}

// commandClearArgs represents the arguments to the clear command
type commandClearArgs struct {
	// The names of the mailboxes to create
	mailboxes []string
}

func (a *commandClearArgs) initFrom(args map[string]any) {
	if v, ok := args["mailboxes"]; ok {
		mailboxesAny := v.([]any)
		mailboxesStr := make([]string, len(mailboxesAny))
		for i, mailbox := range mailboxesAny {
			mailboxesStr[i] = mailbox.(string)
		}
		a.mailboxes = mailboxesStr
	}
}

func (a *commandClearArgs) isAnyMandatoryFieldEmpty() bool {
	return len(a.mailboxes) == 0
}

// commandFlagArgs represents the arguments to the flag command.
// One of add, remove or set fields is mandatory
type commandFlagArgs struct {
	// add new flags to the mail
	add []string
	// remove flags from the mail
	remove []string
	// set mail flags (overwrites current mail flags).
	// If set is not empty, add and remove fields will not be considered
	set []string
}

func (a *commandFlagArgs) initFrom(args map[string]any) {
	if v, ok := args["add"]; ok {
		addAny := v.([]any)
		AddStr := make([]string, len(addAny))
		for i, mailbox := range addAny {
			AddStr[i] = mailbox.(string)
		}
		a.add = AddStr
	}
	if v, ok := args["remove"]; ok {
		removeAny := v.([]any)
		RemoveStr := make([]string, len(removeAny))
		for i, mailbox := range removeAny {
			RemoveStr[i] = mailbox.(string)
		}
		a.remove = RemoveStr
	}
	if v, ok := args["set"]; ok {
		setAny := v.([]any)
		SetStr := make([]string, len(setAny))
		for i, mailbox := range setAny {
			SetStr[i] = mailbox.(string)
		}
		a.set = SetStr
	}
}

func (a *commandFlagArgs) isEmpty() bool {
	return len(a.add)+len(a.remove)+len(a.set) == 0
}

// commandMoveArgs represents the arguments to the move command.
// mailbox is a mandatory field
type commandMoveArgs struct {
	// The name of the mailbox to move the mail to
	mailbox string
}

func (a *commandMoveArgs) initFrom(args map[string]any) {
	if v, ok := args["mailbox"]; ok {
		a.mailbox = v.(string)
	}
}

func (a *commandMoveArgs) isAnyMandatoryFieldEmpty() bool {
	return a.mailbox == ""
}

// SearchCriteria represents the search criteria to fetch mails through FETCH command (https://www.rfc-editor.org/rfc/rfc3501#section-6.4.5)
type SearchCriteria struct {
	Mailbox string `json:"mailbox,omitempty" yaml:"mailbox,omitempty"`
	UID     uint32 `json:"uid,omitempty" yaml:"uid,omitempty"`
	From    string `json:"from,omitempty" yaml:"from,omitempty"`
	To      string `json:"to,omitempty" yaml:"to,omitempty"`
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
	Body    string `json:"body,omitempty" yaml:"body,omitempty"`
}

func (s *SearchCriteria) isAnyMandatoryFieldEmpty() bool {
	return s.Mailbox == ""
}

// Command represents a command that can be performed to messages or mailboxes.
type Command struct {
	// Search defines the search criteria to retrieve a mail. Some commands need to act upon a mail retrieved through Search.
	Search SearchCriteria `json:"search" yaml:"search"`
	// Name defines an IMAP command to execute.
	Name CommandName `json:"name" yaml:"name"`
	// Args defines the arguments to the command. Arguments associated to the command are listed in the README file
	Args map[string]any `json:"args" yaml:"args"`
}

type Mail struct {
	UID     uint32   `json:"uid,omitempty" yaml:"uid,omitempty"`
	From    string   `json:"from,omitempty" yaml:"from,omitempty"`
	To      string   `json:"to,omitempty" yaml:"to,omitempty"`
	Subject string   `json:"subject,omitempty" yaml:"subject,omitempty"`
	Body    string   `json:"body,omitempty" yaml:"body,omitempty"`
	Flags   []string `json:"flags,omitempty" yaml:"flags,omitempty"`
}

func (m *Mail) containsFlag(flag string) bool {
	for _, mFlag := range m.Flags {
		if flag == mFlag {
			return true
		}
	}
	return false
}

func (m *Mail) containsFlags(flags []string) bool {
	for _, flag := range flags {
		found := m.containsFlag(flag)
		if !found {
			return false
		}
	}
	return true
}

// containsAnyButSeenOrRecent returns true if m contains any flag in flags (except "\Seen" or "\Recent")
func (m *Mail) containsAnyButSeenOrRecent(flags []string) bool {
	for _, flag := range flags {
		if flag == `\Seen` || flag == `\Recent` {
			continue
		}
		found := m.containsFlag(flag)
		if found {
			return true
		}
	}
	return false
}

func (m *Mail) hasSameFlags(flags []string) bool {
	slices.Sort(m.Flags)
	slices.Sort(flags)
	return slices.Equal(m.Flags, flags)
}

// CommandResult contains the results of a command with the states of the mail before and after the command.
type CommandResult struct {
	// Search represents the result of the Command's search field
	Search Mail `json:"search,omitempty" yaml:"search,omitempty"`
	// Mail represents the state of the searched mail after the command was executed
	Mail        Mail    `json:"mail,omitempty" yaml:"mail,omitempty"`
	Err         string  `json:"err,omitempty" yaml:"err,omitempty"`
	TimeSeconds float64 `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
}

// Result represents a step result. It contains the results of every command that was executed
type Result struct {
	Commands []CommandResult `json:"commands,omitempty" yaml:"commands,omitempty"`
}

type Client struct {
	*imap.Client
}

type AuthConfig struct {
	WithTLS  bool   `json:"withtls,omitempty" yaml:"withtls,omitempty"`
	Host     string `json:"host,omitempty" yaml:"host,omitempty"`
	Port     string `json:"port,omitempty" yaml:"port,omitempty"`
	User     string `json:"user,omitempty" yaml:"user,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

// Executor represents a Test Executor
type Executor struct {
	Auth     AuthConfig `json:"auth" yaml:"auth"`
	Commands []Command  `json:"commands,omitempty" yaml:"commands,omitempty"`
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for type imap
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []venom.Assertion{"result.err ShouldBeEmpty"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	c, err := e.connect(e.Auth.Host, e.Auth.Port, e.Auth.User, e.Auth.Password)
	if err != nil {
		return nil, fmt.Errorf("Error while connecting: %w", err)
	}
	defer c.Logout(imapClientTimeout) // nolint
	client := &Client{c}

	result := Result{}
	result.Commands = e.handleCommands(ctx, client)

	return result, nil
}

// handleCommands executes every Command from Executor and returns their result
func (e Executor) handleCommands(ctx context.Context, c *Client) []CommandResult {
	var results = make([]CommandResult, len(e.Commands))
	for i, command := range e.Commands {
		switch command.Name {
		case CommandAppend:
			args := commandAppendArgs{}
			args.initFrom(command.Args)
			results[i] = c.appendMail(ctx, args)
		case CommandClear:
			args := commandClearArgs{}
			args.initFrom(command.Args)
			results[i] = c.clearMailboxes(ctx, args)
		case CommandCreate:
			args := commandCreateArgs{}
			args.initFrom(command.Args)
			results[i] = c.createMailbox(ctx, args)
		case CommandDelete:
			results[i] = c.deleteMail(ctx, command.Search)
		case CommandFetch:
			results[i] = c.fetchMail(ctx, command.Search)
		case CommandFlag:
			args := commandFlagArgs{}
			args.initFrom(command.Args)
			results[i] = c.flagMail(ctx, command.Search, args)
		case CommandMove:
			args := commandMoveArgs{}
			args.initFrom(command.Args)
			results[i] = c.moveMail(ctx, command.Search, args)
		default:
			results[i].Err = "unknown command"
		}
		if results[i].Err != "" {
			venom.Debug(ctx, fmt.Sprintf("Failed to handle command %s: %s", command.Name, results[i].Err))
		}
	}
	return results
}

// clearMailboxes clears the given mailbox. If mailbox is '*', clears all mailboxes.
func (c *Client) clearMailboxes(ctx context.Context, args commandClearArgs) CommandResult {
	start := time.Now()
	result := CommandResult{}

	if args.mailboxes[0] == "*" {
		venom.Debug(ctx, "\"*\" argument found, clearing all mailboxes")

		cmd, err := imap.Wait(c.List("", "*"))
		if err != nil {
			return CommandResult{Err: fmt.Sprintf("Error while retrieving mailboxes: %v", err)}
		}
		mailboxes := make([]string, len(cmd.Data))
		for i, rsp := range cmd.Data {
			mailboxes[i] = rsp.MailboxInfo().Name
		}
		args.mailboxes = mailboxes
	}
	for _, mailbox := range args.mailboxes {
		err := c.clearMailbox(ctx, mailbox)
		if err != nil {
			return CommandResult{Err: err.Error()}
		}
	}

	result.TimeSeconds = time.Since(start).Seconds()
	venom.Debug(ctx, "Clear command executed in %.2f seconds", result.TimeSeconds)
	return result
}

func (c *Client) clearMailbox(ctx context.Context, mailbox string) error {
	venom.Debug(ctx, "Clearing mailbox %q", mailbox)

	seq, err := imap.NewSeqSet("1:*")
	if err != nil {
		return fmt.Errorf("error while building request to select all messages: %v", err)
	}

	venom.Debug(ctx, "Selecting mailbox %q", mailbox)
	if _, err = c.Select(mailbox, false); err != nil {
		return fmt.Errorf("error while selecting mailbox %q: %v", mailbox, err)
	}
	venom.Debug(ctx, "Adding tag '\\Deleted' to all messages")
	_, err = imap.Wait(c.UIDStore(seq, "+FLAGS.SILENT", imap.NewFlagSet(`\Deleted`)))
	if err != nil {
		return fmt.Errorf("error while adding flag '\\Deleted' to all messages in mailbox %q: %v", mailbox, err)
	}

	venom.Debug(ctx, "Expunging mailbox")
	if _, err = imap.Wait(c.Expunge(seq)); err != nil {
		return fmt.Errorf("error while expunging mails in mailbox %q: %v", mailbox, err)
	}

	// Command execution verifications
	venom.Debug(ctx, "Searching mails in mailbox %q to make sure they were all deleted", mailbox)
	count, err := c.countNumberOfMessagesInMailbox(mailbox)
	if err != nil {
		return fmt.Errorf("error while counting number of messages in mailbox %q after clear command: %v", mailbox, err)
	}
	if count > 0 {
		return fmt.Errorf("%d message(s) were found in mailbox %q", count, mailbox)
	} else {
		venom.Debug(ctx, "No message found in mailbox %q: clear command successfully completed!", mailbox)
	}
	return nil
}

func (c *Client) appendMail(ctx context.Context, args commandAppendArgs) CommandResult {
	start := time.Now()
	result := CommandResult{}

	message := []string{
		fmt.Sprintf("From: %s", args.from),
		fmt.Sprintf("Subject: %s", args.subject),
		fmt.Sprintf("To: %s", args.to),
		"Content-Type: text/plain; charset=utf-8",
		"",
		args.body,
		"",
	}
	messageBytes := []byte(strings.Join(message, "\r\n"))
	literal := imap.NewLiteral(messageBytes)
	flags := imap.NewFlagSet(args.flags...)

	venom.Debug(ctx, "Appending message with fields: %+v", args)
	_, err := imap.Wait(c.Append(args.mailbox, flags, nil, literal))
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("Error while appending message: %v", err)}
	}
	venom.Debug(ctx, "Successfully appended message!")

	// Command execution verifications
	venom.Debug(ctx, "Searching mail to make sure it was created")
	mail, err := c.getFirstFoundMail(ctx, SearchCriteria{
		Mailbox: args.mailbox,
		From:    args.from,
		To:      args.to,
		Subject: args.subject,
		Body:    args.body,
	})
	if err != nil {
		result.Err = fmt.Sprintf("Error while retrieving mail after append command was executed: %v", err)
		venom.Error(ctx, result.Err)
	} else {
		venom.Debug(ctx, "Mail was retrieved: append command successfully completed")
		result.Mail = mail
	}

	result.TimeSeconds = time.Since(start).Seconds()
	venom.Debug(ctx, "Append command executed in %.2f seconds", result.TimeSeconds)
	return result
}

func (c *Client) createMailbox(ctx context.Context, args commandCreateArgs) CommandResult {
	start := time.Now()
	result := CommandResult{}

	venom.Debug(ctx, "Creating mailbox %q", args.mailbox)
	_, err := c.Create(args.mailbox)
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("Error while creating mailbox: %v", err)}
	}
	venom.Debug(ctx, "Successfully created mailbox!")

	// Command execution verifications
	venom.Debug(ctx, "Selecting mailbox after command to make sure it exists")
	_, err = imap.Wait(c.Status(args.mailbox))
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("Error while waiting mailbox status after create command was executed: %v", err)}
	}
	venom.Debug(ctx, "Mailbox was indeed created: create command successfully completed!")

	result.TimeSeconds = time.Since(start).Seconds()
	venom.Debug(ctx, "Create command executed in %.2f seconds", result.TimeSeconds)
	return result
}

func (c *Client) flagMail(ctx context.Context, mailToFind SearchCriteria, args commandFlagArgs) CommandResult {
	start := time.Now()
	result := CommandResult{}

	// As setting flags overrides any existing one, there is no need to add or remove flags if 'set' field is set
	if len(args.set) > 0 {
		result = c.setFlagsOfMail(ctx, mailToFind, args.set)
	} else {
		if len(args.add) > 0 {
			result = c.addFlagsToMail(ctx, mailToFind, args.add)
		}
		if len(args.remove) > 0 {
			result = c.removeFlagsFromMail(ctx, mailToFind, args.remove)
		}
	}

	result.TimeSeconds = time.Since(start).Seconds()
	venom.Debug(ctx, "Flag command executed in %.2f seconds", result.TimeSeconds)
	return result
}

// addFlagsToMail adds the flags contained in flags to the retrieved mail
func (c *Client) addFlagsToMail(ctx context.Context, mailToFind SearchCriteria, flags []string) CommandResult {
	result := CommandResult{}

	venom.Debug(ctx, "Searching mail with criteria: %+v", mailToFind)
	mail, err := c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("add: error while retrieving mail: %v", err)}
	}
	result.Search = mail

	seq, _ := imap.NewSeqSet("")
	seq.AddNum(mail.UID)

	venom.Debug(ctx, "Adding following flags from mail: %v", flags)
	_, err = imap.Wait(c.UIDStore(seq, "+FLAGS.SILENT", imap.NewFlagSet(flags...)))
	if err != nil {
		result.Err = fmt.Sprintf("add: error while calling UID Store command: %v", err)
		return result
	}
	venom.Debug(ctx, "Successfully added flags to mail!")

	// Making sure new flags are the expected ones
	mail, err = c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		result.Err = fmt.Sprintf("add: error while retrieving mail: %v", err)
		return result
	}
	result.Mail = mail

	if !result.Mail.containsFlags(flags) {
		result.Err = fmt.Sprintf("add: mail has flags %v while it should contain flags %v", mail.Flags, flags)
		return result
	}

	return result
}

// removeFlagsFromMail removes the flags contained in flags from the retrieved mail
func (c *Client) removeFlagsFromMail(ctx context.Context, mailToFind SearchCriteria, flags []string) CommandResult {
	result := CommandResult{}

	venom.Debug(ctx, "Searching mail with criteria: %+v", mailToFind)
	mail, err := c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("remove: error while retrieving mail: %v", err)}
	}
	result.Search = mail

	seq, _ := imap.NewSeqSet("")
	seq.AddNum(mail.UID)

	venom.Debug(ctx, "Removing following flags from mail: %v", flags)
	_, err = imap.Wait(c.UIDStore(seq, "-FLAGS.SILENT", imap.NewFlagSet(flags...)))
	if err != nil {
		result.Err = fmt.Sprintf("remove: error while calling UID Store command: %v", err)
		return result
	}
	venom.Debug(ctx, "Successfully removed flags from mail!")

	// Making sure new flags are the expected ones
	mail, err = c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		result.Err = fmt.Sprintf("remove: error while retrieving mail: %v", err)
		return result
	}
	result.Mail = mail

	// Fetching the mail and reading its text section added the "\Seen" flag.
	if result.Mail.containsAnyButSeenOrRecent(flags) {
		result.Err = fmt.Sprintf("remove: mail has flags %v while it should not have flags %v", mail.Flags, flags)
		return result
	}

	return result
}

// setFlagsOfMail sets the flags of the retrieved mail from flags
func (c *Client) setFlagsOfMail(ctx context.Context, mailToFind SearchCriteria, flags []string) CommandResult {
	result := CommandResult{}

	venom.Debug(ctx, "Searching mail with criteria: %+v", mailToFind)
	mail, err := c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("set: error while retrieving mail: %v", err)}
	}
	result.Search = mail

	seq, _ := imap.NewSeqSet("")
	seq.AddNum(mail.UID)

	venom.Debug(ctx, "Setting following mail flags: %v", flags)
	_, err = imap.Wait(c.UIDStore(seq, "FLAGS.SILENT", imap.NewFlagSet(flags...)))
	if err != nil {
		result.Err = fmt.Sprintf("set: error while calling UID Store command: %v", err)
		return result
	}
	venom.Debug(ctx, "Successfully set mail flags!")

	// Making sure new flags are the expected ones
	mail, err = c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		result.Err = fmt.Sprintf("set: error while retrieving mail: %v", err)
		return result
	}
	result.Mail = mail

	// Fetching the mail and reading its text section added the "\Seen" flag.
	flags = append(flags, `\Seen`)
	if !result.Mail.hasSameFlags(flags) {
		result.Err = fmt.Sprintf("set: mail has flags %v while it should have flags %v", mail.Flags, flags)
		return result
	}

	return result
}

func (c *Client) fetchMail(ctx context.Context, mailToFind SearchCriteria) CommandResult {
	start := time.Now()
	result := CommandResult{}

	venom.Debug(ctx, "Searching mail with criteria: %+v", mailToFind)
	mail, err := c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("error while retrieving mail: %v", err)}
	}
	venom.Debug(ctx, "Found mail: %+v", mail)
	// To avoid confusing the user about whether field to test, both are valid as the command does not modify the mail
	result.Search = mail
	result.Mail = mail

	result.TimeSeconds = time.Since(start).Seconds()
	venom.Debug(ctx, "Move command executed in %.2f seconds", result.TimeSeconds)
	return result
}

func (c *Client) deleteMail(ctx context.Context, mailToFind SearchCriteria) CommandResult {
	start := time.Now()
	result := CommandResult{}

	venom.Debug(ctx, "Searching mail with criteria: %+v", mailToFind)
	mail, err := c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("error while retrieving mail: %v", err)}
	}
	venom.Debug(ctx, "Found mail: %+v", mail)
	result.Search = mail

	seq, _ := imap.NewSeqSet("")
	seq.AddNum(mail.UID)

	venom.Debug(ctx, "Selecting mailbox %q", mailToFind.Mailbox)
	if _, err = c.Select(mailToFind.Mailbox, false); err != nil {
		result.Err = fmt.Sprintf("Error while selecting mailbox %q: %v", mailToFind.Mailbox, err)
		return result
	}
	venom.Debug(ctx, "Adding flag '\\Deleted' to mail")
	if _, err = imap.Wait(c.UIDStore(seq, "+FLAGS.SILENT", imap.NewFlagSet(`\Deleted`))); err != nil {
		result.Err = fmt.Sprintf("Error while adding '\\Deleted' flag to mail %d: %v", mail.UID, err)
		return result
	}
	venom.Debug(ctx, "Expunging mailbox %q", mailToFind.Mailbox)
	if _, err = imap.Wait(c.Expunge(seq)); err != nil {
		result.Err = fmt.Sprintf("Error while expunging mailbox %q: %v", mailToFind.Mailbox, err)
		return result
	}
	venom.Debug(ctx, "Mailbox successfully expunged")

	// Command execution verifications
	venom.Debug(ctx, "Searching mail to make sure it was deleted")
	mailToFind.UID = mail.UID
	mail, err = c.getFirstFoundMail(ctx, mailToFind)
	if err != nil && (err == errMailNotFound || err == errEmptyMailbox) {
		venom.Debug(ctx, "Mail was successfully deleted: delete command successfully completed!")
	} else if err == nil {
		result.Err = "Supposedly deleted mail was found after delete command was executed"
		venom.Error(ctx, result.Err)
		result.Mail = mail
	} else {
		result.Err = fmt.Sprintf("Error while trying to confirm mail was deleted: %v", err)
		venom.Warn(ctx, result.Err)
	}

	result.TimeSeconds = time.Since(start).Seconds()
	venom.Debug(ctx, "Delete command executed in %.2f seconds", result.TimeSeconds)
	return result
}

func (c *Client) moveMail(ctx context.Context, mailToFind SearchCriteria, args commandMoveArgs) CommandResult {
	start := time.Now()
	result := CommandResult{}

	venom.Debug(ctx, "Searching mail with criteria: %+v", mailToFind)
	mail, err := c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		return CommandResult{Err: fmt.Sprintf("error while retrieving mail: %v", err)}
	}
	venom.Debug(ctx, "Found mail: %+v", mail)
	result.Search = mail

	seq, _ := imap.NewSeqSet("")
	seq.AddNum(mail.UID)

	venom.Debug(ctx, "Moving message to mailbox %q", mailToFind.Mailbox)
	if _, err = imap.Wait(c.UIDMove(seq, args.mailbox)); err != nil {
		return CommandResult{Err: fmt.Sprintf("Error while selecting mailbox %q: %v", mailToFind.Mailbox, err)}
	}
	venom.Debug(ctx, "Successfully moved message to mailbox %q", mailToFind.Mailbox)

	// Command execution verifications
	venom.Debug(ctx, "Searching mail to make sure it was moved")
	mailToFind.Mailbox = args.mailbox
	mail, err = c.getFirstFoundMail(ctx, mailToFind)
	if err != nil {
		result.Err = fmt.Sprintf("Error while trying to confirm mail was moved: %v", err)
		venom.Warn(ctx, result.Err)
	}
	result.Mail = mail
	venom.Debug(ctx, "Mail was successfully moved: move command successfully completed!")

	result.TimeSeconds = time.Since(start).Seconds()
	venom.Debug(ctx, "Move command executed in %.2f seconds", result.TimeSeconds)
	return result
}

func (m *Mail) isSearched(mailToFind SearchCriteria) (bool, error) {
	var (
		matched bool
		err     error
	)
	if mailToFind.UID != 0 {
		if mailToFind.UID != m.UID {
			return false, nil
		}
	}
	if mailToFind.From != "" {
		matched, err = regexp.MatchString(mailToFind.From, m.From)
		if err != nil || !matched {
			return false, err
		}
	}
	if mailToFind.To != "" {
		matched, err = regexp.MatchString(mailToFind.To, m.To)
		if err != nil || !matched {
			return false, err
		}
	}
	if mailToFind.Subject != "" {
		matched, err = regexp.MatchString(mailToFind.Subject, m.Subject)
		if err != nil || !matched {
			return false, err
		}
	}
	if mailToFind.Body != "" {
		// Remove all new line, return and tab characters that can make the match fail
		replacer := strings.NewReplacer("\n", "", "\r", "", "\t", "")
		m.Body = replacer.Replace(m.Body)
		mailToFind.Body = replacer.Replace(mailToFind.Body)
		matched, err = regexp.MatchString(mailToFind.Body, m.Body)
		if err != nil || !matched {
			return false, err
		}
	}
	return true, nil
}

// getFirstFoundMail returns the first mail found through search criteria
// If peek is true, mail body text will not be read thus leaving flags unchanged (for more information, refer to RFC 3501 section 6.4.5)
func (c *Client) getFirstFoundMail(ctx context.Context, mailToFind SearchCriteria) (Mail, error) {
	if mailToFind.isAnyMandatoryFieldEmpty() {
		return Mail{}, errors.New("empty search criteria: 'mailbox' is a mandatory field")
	}

	count, err := c.countNumberOfMessagesInMailbox(mailToFind.Mailbox)
	if err != nil {
		return Mail{}, fmt.Errorf("error while counting number of messages in mailbox %q: %w", mailToFind.Mailbox, err)
	}
	if count == 0 {
		return Mail{}, errEmptyMailbox
	}

	messages := make([]imap.Response, 0, count)
	messages, err = c.fetchMails(ctx, mailToFind.Mailbox)
	if err != nil {
		return Mail{}, fmt.Errorf("error while fetching messages in mailbox %q: %v", mailToFind.Mailbox, err)
	}

	for _, msg := range messages {
		mail, err := extract(ctx, msg)
		if err != nil {
			venom.Warn(ctx, "Cannot extract the content of the mail: %v", err)
			continue
		}

		// Handle the first found mail only
		found, err := mail.isSearched(mailToFind)
		if err != nil {
			return Mail{}, err
		}

		if found {
			return mail, nil
		}
	}

	return Mail{}, errMailNotFound
}

func (c *Client) fetchMails(ctx context.Context, mailbox string) ([]imap.Response, error) {
	venom.Debug(ctx, "Selecting mailbox %q", mailbox)
	if _, err := c.Select(mailbox, false); err != nil {
		venom.Error(ctx, "Error with select %s", err)
		return []imap.Response{}, err
	}

	seqset, _ := imap.NewSeqSet("1:*")

	messages := []imap.Response{}
	cmd, err := imap.Wait(c.Fetch(seqset, "UID", "ENVELOPE", "FLAGS", "RFC822.HEADER", "RFC822.TEXT", "BODY.PEEK[TEXT]"))
	if err != nil {
		venom.Error(ctx, "Error with FETCH command: %v", err)
		return []imap.Response{}, err
	}

	// Process command data
	for _, rsp := range cmd.Data {
		messages = append(messages, *rsp)
	}

	venom.Debug(ctx, "Fetched %d message(s) from mailbox %q", len(messages), mailbox)
	return messages, nil
}

func (c *Client) countNumberOfMessagesInMailbox(mailbox string) (uint32, error) {
	cmd, err := imap.Wait(c.Status(mailbox))
	if err != nil {
		return 0, err
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

func (e Executor) connect(host, port, imapUsername, imapPassword string) (*imap.Client, error) {
	var (
		err error
		c   *imap.Client
	)
	if e.Auth.WithTLS {
		c, err = imap.DialTLS(host+":"+port, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to dialTLS: %s", err)
		}
		if c.Caps["STARTTLS"] {
			if _, err = imap.Wait(c.StartTLS(nil)); err != nil {
				return nil, fmt.Errorf("unable to start TLS: %s", err)
			}
		}
	} else {
		c, err = imap.Dial(host + ":" + port)
		if err != nil {
			return nil, fmt.Errorf("unable to dial: %s", err)
		}
	}

	c.SetLogMask(imapSafeLogMask)
	if _, err = imap.Wait(c.Login(imapUsername, imapPassword)); err != nil {
		return nil, fmt.Errorf("unable to login: %s", err)
	}
	c.SetLogMask(imapLogMask)

	return c, nil
}
