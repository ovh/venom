package run

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/hashicorp/hcl"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/venom"
	defaultctx "github.com/ovh/venom/context/default"
	redisctx "github.com/ovh/venom/context/redis"
	"github.com/ovh/venom/context/webctx"

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
	exclude         []string
	format          string
	varFiles        []string
	withEnv         bool
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
	Cmd.Flags().StringSliceVarP(&varFiles, "var-from-file", "", []string{""}, "--var-from-file filename.yaml --var-from-file filename2.yaml: hcl|json|yaml, must contains map[string]string'")
	Cmd.Flags().StringSliceVarP(&exclude, "exclude", "", []string{""}, "--exclude filaA.yaml --exclude filaB.yaml --exclude fileC*.yaml")
	Cmd.Flags().StringVarP(&format, "format", "", "xml", "--format:yaml, json, xml, tap")
	Cmd.Flags().BoolVarP(&withEnv, "env", "", true, "Inject environment variables. export FOO=BAR -> you can use {{.FOO}} in your tests")
	Cmd.Flags().BoolVarP(&strict, "strict", "", false, "Exit with an error code if one test fails")
	Cmd.Flags().BoolVarP(&stopOnFailure, "stop-on-failure", "", false, "Stop running Test Suite on first Test Case failure")
	Cmd.Flags().BoolVarP(&noCheckVars, "no-check-variables", "", false, "Don't check variables before run")
	Cmd.Flags().IntVarP(&parallel, "parallel", "", 1, "--parallel=2 : launches 2 Test Suites in parallel")
	Cmd.PersistentFlags().StringVarP(&logLevel, "log", "", "warn", "Log Level : debug, info or warn")
	Cmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "", "", "Output Directory: create tests results file inside this directory")
	Cmd.PersistentFlags().BoolVarP(&enableProfiling, "profiling", "", false, "Enable Mem / CPU Profile with pprof")
}

// Cmd run
var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run Tests",
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

		// Register Context
		v.RegisterTestCaseContext(defaultctx.Name, defaultctx.New())
		v.RegisterTestCaseContext(webctx.Name, webctx.New())
		v.RegisterTestCaseContext(redisctx.Name, redisctx.New())
	},
	Run: func(cmd *cobra.Command, args []string) {
		v.EnableProfiling = enableProfiling
		v.LogLevel = logLevel
		v.OutputDir = outputDir
		v.OutputFormat = format
		v.Parallel = parallel
		v.StopOnFailure = stopOnFailure

		if v.EnableProfiling {
			var filename, filenameCPU, filenameMem string
			if v.OutputDir != "" {
				filename = v.OutputDir + "/"
			}
			filenameCPU = filename + "pprof_cpu_profile.prof"
			filenameMem = filename + "pprof_mem_profile.prof"
			fCPU, errCPU := os.Create(filenameCPU)
			fMem, errMem := os.Create(filenameMem)
			if errCPU != nil || errMem != nil {
				log.Errorf("error while create profile file for root process CPU:%v MEM:%v", errCPU, errMem)
			} else {
				pprof.StartCPUProfile(fCPU)
				p := pprof.Lookup("heap")
				defer p.WriteTo(fMem, 1)
				defer pprof.StopCPUProfile()
			}
		}

		mapvars := make(map[string]string)
		if withEnv {
			variables = append(variables, os.Environ()...)
		}

		for _, f := range varFiles {
			if f == "" {
				continue
			}
			varFileMap := make(map[string]string)
			bytes, err := ioutil.ReadFile(f)
			if err != nil {
				log.Fatal(err)
			}
			switch filepath.Ext(f) {
			case ".hcl":
				err = hcl.Unmarshal(bytes, &varFileMap)
			case ".json":
				err = json.Unmarshal(bytes, &varFileMap)
			case ".yaml", ".yml":
				err = yaml.Unmarshal(bytes, &varFileMap)
			default:
				log.Fatal("unsupported varFile format")
			}
			if err != nil {
				log.Fatal(err)
			}

			for key, value := range varFileMap {
				mapvars[key] = value
			}
		}

		for _, a := range variables {
			t := strings.SplitN(a, "=", 2)
			if len(t) < 2 {
				continue
			}
			mapvars[t[0]] = strings.Join(t[1:], "")
		}

		v.AddVariables(mapvars)

		start := time.Now()

		if !noCheckVars {
			if err := v.Parse(path, exclude); err != nil {
				log.Fatal(err)
			}
		}

		tests, err := v.Process(path, exclude)
		if err != nil {
			log.Fatal(err)
		}

		elapsed := time.Since(start)
		if err := v.OutputResult(*tests, elapsed); err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
		if strict && tests.TotalKO > 0 {
			os.Exit(2)
		}
	},
}
