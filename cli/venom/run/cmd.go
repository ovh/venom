package run

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	yml "github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/ovh/venom"

	"github.com/ovh/venom/executors/dbfixtures"
	"github.com/ovh/venom/executors/exec"
	"github.com/ovh/venom/executors/grpc"
	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/imap"
	"github.com/ovh/venom/executors/kafka"
	"github.com/ovh/venom/executors/ovhapi"
	"github.com/ovh/venom/executors/rabbitmq"
	"github.com/ovh/venom/executors/readfile"
	"github.com/ovh/venom/executors/redis"
	"github.com/ovh/venom/executors/smtp"
	"github.com/ovh/venom/executors/sql"
	"github.com/ovh/venom/executors/ssh"
	"github.com/ovh/venom/executors/web"
)

var (
	path            []string
	variables       []string
	format          string
	varFiles        []string
	logLevel        string
	outputDir       string
	strict          bool
	noCheckVars     bool
	parallel        int
	stopOnFailure   bool
	enableProfiling bool
	v               *venom.Venom
)

func init() {
	Cmd.Flags().StringSliceVarP(&variables, "var", "", []string{""}, "--var cds='cds -f config.json' --var cds2='cds -f config.json'")
	Cmd.Flags().StringSliceVarP(&varFiles, "var-from-file", "", []string{""}, "--var-from-file filename.yaml --var-from-file filename2.yaml: yaml, must contains a dictionnary")
	Cmd.Flags().StringVarP(&format, "format", "", "xml", "--format:yaml, json, xml, tap")
	Cmd.Flags().BoolVarP(&strict, "strict", "", false, "Exit with an error code if one test fails")
	Cmd.Flags().BoolVarP(&stopOnFailure, "stop-on-failure", "", false, "Stop running Test Suite on first Test Case failure")
	Cmd.Flags().BoolVarP(&noCheckVars, "no-check-variables", "", false, "Don't check variables before run")
	Cmd.Flags().IntVarP(&parallel, "parallel", "", 1, "--parallel=2 : launches 2 Test Suites in parallel")
	Cmd.PersistentFlags().StringVarP(&logLevel, "log", "", "warn", "Log Level : debug, info, warn or disable")
	Cmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "", "", "Output Directory: create tests results file inside this directory")
	Cmd.PersistentFlags().BoolVarP(&enableProfiling, "profiling", "", false, "Enable Mem / CPU Profile with pprof")
}

// Cmd run
var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run Tests",
	Long: `
$ venom run *.yml

# to have more information about what's wrong on a test,
# you can use the output-dir. *.dump files will be created
# in this directory, with a lot of useful debug lines:

$ venom run *.yml --output-dir=results

Notice that variables initialized with -var-from-file argument can be overrided with -var argument.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			path = append(path, ".")
		} else {
			path = args[0:]
		}

		v = venom.New()
		v.RegisterExecutor(exec.Name, exec.New())
		v.RegisterExecutor(http.Name, http.New())
		v.RegisterExecutor(imap.Name, imap.New())
		v.RegisterExecutor(readfile.Name, readfile.New())
		v.RegisterExecutor(smtp.Name, smtp.New())
		v.RegisterExecutor(ssh.Name, ssh.New())
		v.RegisterExecutor(web.Name, web.New())
		v.RegisterExecutor(ovhapi.Name, ovhapi.New())
		v.RegisterExecutor(dbfixtures.Name, dbfixtures.New())
		v.RegisterExecutor(redis.Name, redis.New())
		v.RegisterExecutor(kafka.Name, kafka.New())
		v.RegisterExecutor(grpc.Name, grpc.New())
		v.RegisterExecutor(rabbitmq.Name, rabbitmq.New())
		v.RegisterExecutor(sql.Name, sql.New())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		v.EnableProfiling = enableProfiling
		v.LogLevel = logLevel
		v.OutputDir = outputDir
		v.OutputFormat = format
		v.Parallel = parallel
		v.StopOnFailure = stopOnFailure

		if v.EnableProfiling {
			fCPU, err := os.Create(filepath.Join(v.OutputDir, "pprof_cpu_profile.prof"))
			if err != nil {
				log.Errorf("error while create profile file %v", err)
			}
			fMem, err := os.Create(filepath.Join(v.OutputDir, "pprof_mem_profile.prof"))
			if err != nil {
				log.Errorf("error while create profile file %v", err)
			}
			if fCPU != nil && fMem != nil {
				pprof.StartCPUProfile(fCPU)
				p := pprof.Lookup("heap")
				defer p.WriteTo(fMem, 1)
				defer pprof.StopCPUProfile()
			}
		}

		var readers = []io.Reader{}
		for _, f := range varFiles {
			if f == "" {
				continue
			}
			fi, err := os.Open(f)
			if err != nil {
				return fmt.Errorf("unable to open var-file %s: %v", f, err)
			}
			defer fi.Close()
			readers = append(readers, fi)
		}

		mapvars, err := readInitialVariables(variables, readers, os.Environ())
		if err != nil {
			return err
		}
		v.AddVariables(mapvars)

		start := time.Now()

		if !noCheckVars {
			if err := v.Parse(path); err != nil {
				fmt.Println(err)
				os.Exit(2)
				return err
			}
		}

		tests, err := v.Process(context.Background(), path)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
			return err
		}

		elapsed := time.Since(start)
		if err := v.OutputResult(*tests, elapsed); err != nil {
			fmt.Println(err)
			os.Exit(2)
			return err
		}
		if strict && tests.TotalKO > 0 {
			os.Exit(2)
		}

		return nil
	},
}

func readInitialVariables(argsVars []string, argVarsFiles []io.Reader, environ []string) (map[string]interface{}, error) {
	var cast = func(vS string) interface{} {
		var v interface{}
		_ = yml.Unmarshal([]byte(vS), &v) // ignore errors
		return v
	}

	var result = map[string]interface{}{}
	for _, env := range environ {
		if strings.HasPrefix(env, "VENOM_VAR_") {
			tuple := strings.Split(env, "=")
			k := strings.TrimPrefix(tuple[0], "VENOM_VAR_")
			result[k] = cast(tuple[1])
		}
	}

	for _, r := range argVarsFiles {
		var tmpResult = map[string]interface{}{}
		btes, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		if err := yml.Unmarshal(btes, &tmpResult); err != nil {
			return nil, err
		}
		for k, v := range tmpResult {
			result[k] = v
		}
	}

	for _, arg := range argsVars {
		if arg == "" {
			continue
		}
		tuple := strings.Split(arg, "=")
		if len(tuple) != 2 {
			return nil, fmt.Errorf("invalid variable declaration: %v", arg)
		}
		result[tuple[0]] = cast(tuple[1])
	}

	return result, nil
}
