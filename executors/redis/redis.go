package redis

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/gomodule/redigo/redis"
	shellwords "github.com/mattn/go-shellwords"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	"github.com/pkg/errors"
)

// Name of executor
const Name = "redis"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor represents the redis executor
type Executor struct {
	DialURL  string   `json:"dialURL,omitempty" yaml:"dialURL,omitempty" mapstructure:"dialURL"`
	Commands []string `json:"commands,omitempty" yaml:"commands,omitempty"`
	FilePath string   `json:"path,omitempty" yaml:"path,omitempty" mapstructure:"path"`
}

// Command represents a redis command and the result
type Command struct {
	Name     string        `json:"name,omitempty" yaml:"name,omitempty"`
	Args     []interface{} `json:"args,omitempty" yaml:"args,omitempty"`
	Response interface{}   `json:"response,omitempty" yaml:"response,omitempty"`
}

// Result represents a step result.
type Result struct {
	Commands []Command `json:"commands,omitempty" yaml:"commands,omitempty"`
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// Run execute TestStep
func (Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	if e.DialURL == "" {
		e.DialURL = venom.StringVarFromCtx(ctx, "redis.dialURL")
	}

	if e.DialURL == "" {
		return nil, fmt.Errorf("missing dialURL")
	}

	redisClient, err := redis.DialURL(e.DialURL)
	if err != nil {
		return nil, err
	}

	workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")

	var commands []string
	if e.FilePath != "" {
		commands, err = file2lines(path.Join(workdir, e.FilePath))
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to load file")
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
	switch p := res.(type) {
	case []interface{}:
		var result []interface{}
		for i := range p {
			u := p[i]
			k := handleRedisResponse(u, err)
			result = append(result, k)
		}
		return result
	default:
		t, _ := redis.String(res, err) // nolint
		return t
	}
}

func file2lines(filePath string) ([]string, error) {
	var lines []string
	f, err := os.Open(filePath)
	if err != nil {
		return lines, err
	}
	defer f.Close()

	/*
	Thanks to Mark Karamyar to write this blog post : https://devmarkpro.com/working-big-files-golang
	"bufio package has a maximum token size which equals 64 * 1024 (~65.6kb).
	So if one line of our lines is bigger than this size, we got this error token too long error."
	To avoid this error, we will use Readline method and check isPrefix return value
	*/
	reader := bufio.NewReader(f)
	for {
		line, err := read(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		lines = append(lines, string(line))
	}

	return lines, nil
}

func read(r *bufio.Reader) ([]byte, error) {
	var (
		isPrefix = true
		err      error
		line, ln []byte
	)

	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}

	return ln, err
}