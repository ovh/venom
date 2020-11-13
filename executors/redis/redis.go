package redis

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"

	"github.com/garyburd/redigo/redis"
	shellwords "github.com/mattn/go-shellwords"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
)

// Name of executor
const Name = "redis"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

//Executor represents the redis executor
type Executor struct {
	Commands []string `json:"commands,omitempty" yaml:"commands,omitempty"`
	FilePath string   `json:"path,omitempty" yaml:"path,omitempty" mapstructure:"path"`
}

//Command represents a redis command and the result
type Command struct {
	Name     string        `json:"name,omitempty" yaml:"name,omitempty"`
	Args     []interface{} `json:"args,omitempty" yaml:"args,omitempty"`
	Response interface{}   `json:"response,omitempty" yaml:"response,omitempty"`
}

// Result represents a step result.
type Result struct {
	Commands []Command `json:"commands,omitempty" yaml:"commands,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return the default assertions of the executor.
func (e Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{}}
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, testCaseContext venom.TestCaseContext, step venom.TestStep, workdir string) (interface{}, error) {
	dialURL := venom.StringVarFromCtx(ctx, "redis.dialURL")
	if dialURL == "" {
		return nil, fmt.Errorf("missing redis.dialURL variable")
	}

	redisClient, err := redis.DialURL(dialURL)
	if err != nil {
		return nil, err
	}

	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	var commands []string
	if e.FilePath != "" {
		commands, err = file2lines(path.Join(workdir, e.FilePath))
		if err != nil {
			return nil, fmt.Errorf("Failed to load file %v", err)
		}
	} else {
		commands = e.Commands
	}
	result := Result{Commands: []Command{}}

	for i := range commands {
		if commands[i] == "" {
			continue
		}
		name, args, err := getCommandDetails(commands[i])
		if err != nil {
			return nil, err
		}

		res, err := redisClient.Do(name, args...)
		if err != nil {
			arg := fmt.Sprint(args)
			return nil, fmt.Errorf("redis executor failed to execute command %s %s : %s", name, arg, res)
		}

		r := handleRedisResponse(res, err)
		result.Commands = append(result.Commands, Command{
			Name:     name,
			Args:     args,
			Response: r,
		})

	}
	return result, nil
}

func getCommandDetails(command string) (name string, arg []interface{}, err error) {
	cmd, err := shellwords.Parse(command)
	if err != nil {
		return "", nil, err
	}

	name = cmd[0]
	arguments := append(cmd[:0], cmd[1:]...)

	args := sliceStringToSliceInterface(arguments)

	return name, args, nil
}

func sliceStringToSliceInterface(args []string) []interface{} {
	s := make([]interface{}, len(args))
	for i, v := range args {
		s[i] = v
	}
	return s
}

func handleRedisResponse(res interface{}, err error) interface{} {
	var r interface{}
	switch p := res.(type) {
	case []interface{}:
		var result = []string{}
		for i := range p {
			u := p[i]
			k, _ := redis.String(u, err)
			result = append(result, k)
		}
		r = result
	default:
		var result = []string{}
		t, _ := redis.String(res, err)
		result = append(result, t)
		r = t
	}

	return r
}

func file2lines(filePath string) ([]string, error) {
	var lines []string
	f, err := os.Open(filePath)
	if err != nil {
		return lines, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
