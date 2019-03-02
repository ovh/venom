package venom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
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

func (e executorModule) New(ctx context.Context, v *Venom) (Executor, error) {
	var starter executorStarter
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
	if err := starter.logServer.ListenUDP(starter.logServerAddress); err != nil {
		return nil, err
	}
	if err := starter.logServer.ListenTCP(starter.logServerAddress); err != nil {
		return nil, err
	}
	go func(s *syslog.Server) {
		s.Boot()
		log.Println("syslog server booted", starter.logServerAddress)
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
	logServer        *syslog.Server
	logServerAddress string
	executorModule
}

func (e *executorStarter) Run(ctx TestContext, logger Logger, step TestStep) (ExecutorResult, error) {
	if step == nil {
		return nil, nil
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Instanciate the execute command
	cmd := exec.CommandContext(ctx, e.entrypoint, "execute", "--logger", e.logServerAddress, "--log-level", e.v.LogLevel)

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

	// Unmarshal the result
	var res ExecutorResult
	if err := json.Unmarshal(btes, &res); err != nil {
		return nil, fmt.Errorf("unable to parse module output: %v", err)
	}

	return res, nil
}

func (e *executorStarter) logsHandler(ctx context.Context) syslog.Handler {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	log.Println("starting syslog server handler...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case logParts := <-channel:
				fmt.Println(">>>", logParts)
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
		v.logger.Debugf("checking step %s against %+v", step.GetType(), manifest)
		if ok && manifest.Type == "executor" && step.GetType() == manifest.Name {
			mod = &e
			break
		}
	}
	if mod == nil {
		return nil, fmt.Errorf("unrecognized type %s", step.GetType())
	}
	return mod, nil
}
