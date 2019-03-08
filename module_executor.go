package venom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/phayes/freeport"
	"gopkg.in/mcuadros/go-syslog.v2"
)

type executorModule struct {
	retry      int
	delay      int
	timeout    int
	entrypoint string
	manifest   VenomModuleManifest
}

func (e executorModule) Manifest() VenomModuleManifest {
	return e.manifest
}

func (e executorModule) New(ctx context.Context, v *Venom, l Logger) (Executor, error) {
	var starter executorStarter
	starter.l = LoggerWithField(l, "executor", e.manifest.Name)
	starter.v = v
	starter.executorModule = e
	starter.logServer = syslog.NewServer()
	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	starter.logServerAddress = "0.0.0.0:" + strconv.Itoa(port)
	starter.logServer.SetHandler(starter.logsHandler(ctx))
	starter.logServer.SetFormat(syslog.Automatic)
	l.Debugf("starting syslog server on %s", starter.logServerAddress)
	if err := starter.logServer.ListenUDP(starter.logServerAddress); err != nil {
		return nil, err
	}
	if err := starter.logServer.ListenTCP(starter.logServerAddress); err != nil {
		return nil, err
	}
	go func(s *syslog.Server) {
		s.Boot()
		s.Wait()
	}(starter.logServer)

	go func(s *syslog.Server) {
		<-ctx.Done()
		log.Println("syslog server killed")

		s.Kill()
	}(starter.logServer)

	return &starter, nil
}

func (e executorModule) GetDefaultAssertions(ctx TestContext) (*StepAssertions, error) {
	// Instanciate the execute command
	cmd := exec.CommandContext(ctx, e.entrypoint, "assertions")

	output := new(bytes.Buffer)
	cmd.Stdout = output
	cmd.Stderr = output
	// TODO: start the command in the right working directory

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to start command: %v", err)
	}

	// Run the command and wait for the result
	waitErr := cmd.Wait()

	btes := output.Bytes()

	// Check error
	if waitErr != nil {
		return nil, fmt.Errorf("error: %v", waitErr)
	}

	if strings.TrimSpace(string(btes)) == "" {
		return nil, nil
	}

	// Unmarshal the result
	var res StepAssertions
	if err := json.Unmarshal(btes, &res); err != nil {
		return nil, fmt.Errorf("unable to parse module output: %v", err)
	}
	return &res, nil
}

// manage the logger
// manage the working directory
// manage the log-level
type executorStarter struct {
	v                *Venom
	l                Logger
	logServer        *syslog.Server
	logServerAddress string
	executorModule
}

func (e *executorStarter) Run(ctx TestContext, step TestStep) (ExecutorResult, error) {
	if step == nil {
		return nil, nil
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Instanciate the execute command
	cmd := exec.CommandContext(ctx, e.entrypoint, "execute", "--logger", e.logServerAddress, "--log-level", e.v.LogLevel)
	cmd.Dir = ctx.GetWorkingDirectory()

	// Write in the stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("unable to open stdin: %v", err)
	}

	encoder := json.NewEncoder(stdin)
	if err := encoder.Encode(step); err != nil {
		return nil, fmt.Errorf("unable to write to stdin: %v", err)
	}

	if err := stdin.Close(); err != nil {
		return nil, fmt.Errorf("unable to close stdin: %v", err)
	}

	output := new(bytes.Buffer)
	cmd.Stdout = output
	cmd.Stderr = output

	e.l.Debugf("starting command %s", cmd.Path)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to start command: %v", err)
	}

	// Run the command and wait for the result
	waitErr := cmd.Wait()

	btes := output.Bytes()

	// Check error
	if waitErr != nil {
		return nil, fmt.Errorf("error: %v", waitErr)
	}

	// Unmarshal the result
	var res ExecutorResult
	if err := json.Unmarshal(btes, &res); err != nil {
		return nil, fmt.Errorf("unable to parse module output: %v", err)
	}

	return res, nil
}

var (
	levelRegexp = regexp.MustCompile(`level=([a-z]*)`)
	msgRegexp   = regexp.MustCompile(`msg=(".*"|\w)`)
)

func (e *executorStarter) logsHandler(ctx context.Context) syslog.Handler {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case logParts := <-channel:
				content, has := logParts["content"]
				if !has {
					continue
				}
				scontent, ok := content.(string)
				if !ok {
					continue
				}

				levelMatch := levelRegexp.FindStringSubmatch(scontent)
				if len(levelMatch) != 2 {
					continue
				}
				level := levelMatch[1]

				msgMatch := msgRegexp.FindStringSubmatch(scontent)
				if len(msgMatch) != 2 {
					continue
				}
				msg := msgMatch[1]

				msg = strings.TrimPrefix(msg, "\"")
				msg = strings.TrimSuffix(msg, "\"")

				switch level {
				case "debug":
					e.l.Debugf(msg)
				case "info":
					e.l.Infof(msg)
				case "warning":
					e.l.Warningf(msg)
				case "error":
					e.l.Errorf(msg)
				case "fatal":
					e.l.Fatalf(msg)
				default:
					log.Println(level, msg)
				}

				//TODO: remap to logrus
			}
		}
	}()
	return handler
}

func (v *Venom) getExecutorModule(step TestStep) (*executorModule, error) {
	allModules, err := v.ListModules()
	if err != nil {
		return nil, err
	}
	var mod *executorModule
	for _, m := range allModules {
		var manifest = m.Manifest()
		e, ok := m.(executorModule)
		var stepType = step.GetType()
		if stepType == "" {
			stepType = "exec"
		}
		if ok && manifest.Type == "executor" && stepType == manifest.Name {
			mod = &e
			break
		}
	}
	if mod == nil {
		return nil, fmt.Errorf("unrecognized type %s", step.GetType())
	}
	return mod, nil
}
