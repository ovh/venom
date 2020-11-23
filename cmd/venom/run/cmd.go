package run

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
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
	path          []string
	variables     []string
	format        string
	varFiles      []string
	outputDir     string
	stopOnFailure bool
	verbose       *int
	v             *venom.Venom
)

func init() {
	Cmd.Flags().StringSliceVarP(&variables, "var", "", []string{""}, "--var cds='cds -f config.json' --var cds2='cds -f config.json'")
	Cmd.Flags().StringSliceVarP(&varFiles, "var-from-file", "", []string{""}, "--var-from-file filename.yaml --var-from-file filename2.yaml: yaml, must contains a dictionnary")
	Cmd.Flags().StringVarP(&format, "format", "", "xml", "--format:yaml, json, xml, tap")
	Cmd.Flags().BoolVarP(&stopOnFailure, "stop-on-failure", "", false, "Stop running Test Suite on first Test Case failure")
	Cmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "", "", "Output Directory: create tests results file inside this directory")
	verbose = Cmd.Flags().CountP("verbose", "v", "verbose. -vv to very verbose and -vvv to very verbose with CPU Profiling")

	if err := initFromEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
}

func initFromEnv() error {
	if os.Getenv("VENOM_VAR") != "" {
		variables = strings.Split(os.Getenv("VENOM_VAR"), " ")
	}
	if os.Getenv("VENOM_VAR_FROM_FILE") != "" {
		varFiles = strings.Split(os.Getenv("VENOM_VAR_FROM_FILE"), " ")
	}
	if os.Getenv("VENOM_FORMAT") != "" {
		format = os.Getenv("VENOM_FORMAT")
	}
	if os.Getenv("VENOM_STOP_ON_FAILURE") != "" {
		var err error
		stopOnFailure, err = strconv.ParseBool(os.Getenv("VENOM_STOP_ON_FAILURE"))
		if err != nil {
			return fmt.Errorf("invalid value for VENOM_STOP_ON_FAILURE")
		}
	}
	if os.Getenv("VENOM_OUTPUT_DIR") != "" {
		outputDir = os.Getenv("VENOM_OUTPUT_DIR")
	}
	if os.Getenv("VENOM_VERBOSE") != "" {
		v, err := strconv.ParseInt(os.Getenv("VENOM_VERBOSE"), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid value for VENOM_VERBOSE, must be 1, 2 or 3")
		}
		v2 := int(v)
		verbose = &v2
	}
	return nil
}

func displayArg(ctx context.Context) {
	venom.Debug(ctx, "arg variables=%v", strings.Join(variables, " "))
	venom.Debug(ctx, "arg varFiles=%v", strings.Join(varFiles, " "))
	venom.Debug(ctx, "arg format=%v", format)
	venom.Debug(ctx, "arg stopOnFailure=%v", stopOnFailure)
	venom.Debug(ctx, "arg outputDir=%v", outputDir)
	venom.Debug(ctx, "arg verbose=%v", *verbose)
}

// Cmd run
var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run Tests",
	Long: `
$ venom run *.yml

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
		v.OutputDir = outputDir
		v.OutputFormat = format
		v.StopOnFailure = stopOnFailure
		v.Verbose = *verbose

		if err := v.InitLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
			return err
		}

		if v.Verbose == 3 {
			fCPU, err := os.Create(filepath.Join(v.OutputDir, "pprof_cpu_profile.prof"))
			if err != nil {
				log.Errorf("error while create profile file %v", err)
			}
			fMem, err := os.Create(filepath.Join(v.OutputDir, "pprof_mem_profile.prof"))
			if err != nil {
				log.Errorf("error while create profile file %v", err)
			}
			if fCPU != nil && fMem != nil {
				pprof.StartCPUProfile(fCPU) //nolint
				p := pprof.Lookup("heap")
				defer p.WriteTo(fMem, 1) //nolint
				defer pprof.StopCPUProfile()
			}
		}
		if *verbose >= 2 {
			displayArg(context.Background())
		}

		var readers = []io.Reader{}
		for _, f := range varFiles {
			if f == "" {
				continue
			}
			fi, err := os.Open(f)
			if err != nil {
				return fmt.Errorf("unable to open var-from-file %s: %v", f, err)
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

		if err := v.Parse(path); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
			return err
		}

		tests, err := v.Process(context.Background(), path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
			return err
		}

		elapsed := time.Since(start)
		if err := v.OutputResult(*tests, elapsed); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
			return err
		}
		if tests.TotalKO > 0 {
			os.Exit(2)
		}

		return nil
	},
}

func readInitialVariables(argsVars []string, argVarsFiles []io.Reader, environ []string) (map[string]interface{}, error) {
	var cast = func(vS string) interface{} {
		var v interface{}
		_ = yml.Unmarshal([]byte(vS), &v) //nolint
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
