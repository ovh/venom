package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/ovh/venom"
)

// Name for test ssh
const (
	Name       = "ssh"
	sudoprompt = "sudo_venom"
)

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	Host                  string `json:"host,omitempty" yaml:"host,omitempty"`
	Command               string `json:"command,omitempty" yaml:"command,omitempty"`
	User                  string `json:"user,omitempty" yaml:"user,omitempty"`
	Password              string `json:"password,omitempty" yaml:"password,omitempty"`
	PrivateKey            string `json:"privatekey,omitempty" yaml:"privatekey,omitempty"`
	Sudo                  string `json:"sudo,omitempty" yaml:"sudo,omitempty"`
	SudoPassword          string `json:"sudopassword,omitempty" yaml:"sudopassword,omitempty"`
	InsecureIgnoreHostKey bool   `json:"insecure_ignore_host_key,omitempty" yaml:"insecure_ignore_host_key,omitempty"`
	Timeout               int    `json:"timeout,omitempty" yaml:"timeout,omitempty"` // connection timeout in seconds, default 30
}

const defaultSSHTimeoutSeconds = 30

// Result represents a step result
type Result struct {
	Systemout   string  `json:"systemout,omitempty" yaml:"systemout,omitempty"`
	Systemerr   string  `json:"systemerr,omitempty" yaml:"systemerr,omitempty"`
	Err         string  `json:"err,omitempty" yaml:"err,omitempty"`
	Code        string  `json:"code,omitempty" yaml:"code,omitempty"`
	TimeSeconds float64 `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []venom.Assertion{"result.code ShouldEqual 0"}}
}

// Run execute TestStep of type exec
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	if e.Command == "" {
		return nil, fmt.Errorf("Invalid command")
	}

	start := time.Now()
	result := Result{}

	workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")
	client, session, err := connectToHost(e.User, e.Password, e.PrivateKey, e.Host, e.Sudo, workdir, e.InsecureIgnoreHostKey, e.Timeout)
	if err != nil {
		result.Err = err.Error()
	} else {
		defer client.Close()
		stdout := &Buffer{}
		stderr := &Buffer{}

		session.Stderr = stderr
		session.Stdout = stdout
		stdin, _ := session.StdinPipe()

		// Handle sudo password
		command := e.Command
		quit := make(chan bool)
		if e.Sudo != "" {
			command = "TERM=xterm-mono sudo -S -p " + sudoprompt + " -u " + e.Sudo + " " + command
			if e.SudoPassword == "" {
				e.SudoPassword = e.Password
			}
			go handleSudo(stdin, stdout, quit, e.SudoPassword)
		}

		if err := session.Run(command); err != nil {
			if exiterr, ok := err.(*ssh.ExitError); ok {
				status := exiterr.ExitStatus()
				result.Code = strconv.Itoa(status)
			} else if _, ok := err.(*ssh.ExitMissingError); ok {
				result.Code = strconv.Itoa(127)
				result.Err = err.Error()
			} else {
				result.Code = strconv.Itoa(137)
				result.Err = err.Error()
			}
		} else {
			result.Code = "0"
		}

		if e.Sudo != "" {
			quit <- true
		}
		result.Systemerr = strings.TrimSpace(stderr.String())
		result.Systemout = strings.TrimSpace(stdout.String())
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()

	return result, nil
}

func handleSudo(in io.Writer, out *Buffer, quit chan bool, password string) {
	sudopromptlen := len(sudoprompt)
	for {
		select {
		case <-quit:
			return
		default:
			content := out.String()
			bufferLen := utf8.RuneCountInString(content)

			// Check if we have to enter password
			if bufferLen >= sudopromptlen && strings.Contains(content[bufferLen-sudopromptlen:], sudoprompt) {
				in.Write([]byte(password + "\n"))
				out.Truncate(0)
			}
		}
	}
}

func connectToHost(u, pass, key, host, sudo, workdir string, insecureIgnoreHostKey bool, timeoutSeconds int) (*ssh.Client, *ssh.Session, error) {
	// Default user is current username
	if u == "" {
		osUser, err := user.Current()
		if err != nil {
			return nil, nil, err
		}
		u = osUser.Username
	}

	// If password is set, and we don't have key use it
	var auth []ssh.AuthMethod
	if pass != "" && key == "" {
		auth = []ssh.AuthMethod{ssh.Password(pass)}
	} else {
		// Load the the private key
		key, err := privateKey(key, workdir)
		if err != nil {
			return nil, nil, err
		}
		auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	}

	hostKeyCallback, err := buildHostKeyCallback(insecureIgnoreHostKey)
	if err != nil {
		return nil, nil, err
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultSSHTimeoutSeconds
	}

	sshConfig := &ssh.ClientConfig{
		User:            u,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         time.Duration(timeoutSeconds) * time.Second,
	}

	// If host doen't contain port, set the default port
	if !strings.Contains(host, ":") {
		host += ":22"
	}

	// Open the tcp connection
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	// New ssh session
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	// Request PTY for sudo cmd
	if sudo != "" {
		modes := ssh.TerminalModes{
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
			return nil, nil, err
		}
	}

	return client, session, nil
}

func privateKey(file, workdir string) (key ssh.Signer, err error) {
	// Default private key is $HOME/.ssh/id_rsa
	if file == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		file = filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
	} else if filepath.IsAbs(file) {
		// Absolute paths are only allowed under the user's $HOME, to prevent
		// a malicious testsuite from pointing to /etc/shadow or similar.
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("unable to resolve current user to validate privatekey path: %w", err)
		}
		clean := filepath.Clean(file)
		homePrefix := filepath.Clean(usr.HomeDir) + string(os.PathSeparator)
		if !strings.HasPrefix(clean, homePrefix) {
			return nil, fmt.Errorf("absolute privatekey path %q must be located under the user's home directory %q", file, usr.HomeDir)
		}
		file = clean
	} else {
		// Resolve relative paths under the testsuite workdir.
		resolved, err := venom.ResolveWorkdirPath(workdir, file)
		if err != nil {
			return nil, fmt.Errorf("invalid privatekey path: %w", err)
		}
		file = resolved
	}

	// Read the file
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	// Parse it
	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// buildHostKeyCallback returns the ssh.HostKeyCallback to use for the
// connection. By default it verifies the server key against the user's
// $HOME/.ssh/known_hosts. If insecureIgnoreHostKey is true, all server
// keys are accepted (legacy behaviour, only for trusted environments).
func buildHostKeyCallback(insecureIgnoreHostKey bool) (ssh.HostKeyCallback, error) {
	if insecureIgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to resolve current user to locate known_hosts: %w", err)
	}
	knownHostsPath := filepath.Join(usr.HomeDir, ".ssh", "known_hosts")
	if _, err := os.Stat(knownHostsPath); err != nil {
		return nil, fmt.Errorf("known_hosts file %q is not available (%w); populate it or set 'insecure_ignore_host_key: true' to bypass host key verification", knownHostsPath, err)
	}
	cb, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to load known_hosts from %q: %w", knownHostsPath, err)
	}
	return cb, nil
}
