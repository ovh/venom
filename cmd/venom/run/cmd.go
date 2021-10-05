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

	"github.com/ghodss/yaml"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors/amqp"
	"github.com/ovh/venom/executors/dbfixtures"
	"github.com/ovh/venom/executors/exec"
	"github.com/ovh/venom/executors/grpc"
	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/imap"
	"github.com/ovh/venom/executors/kafka"
	"github.com/ovh/venom/executors/mqtt"
	"github.com/ovh/venom/executors/ovhapi"
	"github.com/ovh/venom/executors/rabbitmq"
	"github.com/ovh/venom/executors/readfile"
	"github.com/ovh/venom/executors/redis"
	"github.com/ovh/venom/executors/smtp"
	"github.com/ovh/venom/executors/sql"
	"github.com/ovh/venom/executors/ssh"
	"github.com/ovh/venom/executors/tavern"
	"github.com/ovh/venom/executors/web"
)

var (
	path []string
	v    *venom.Venom

	variables     []string
	format        string
	varFiles      []string
	outputDir     string
	libDir        string
	stopOnFailure bool
	verbose       int

	variablesFlag     *[]string
	formatFlag        *string
	varFilesFlag      *[]string
	outputDirFlag     *string
	libDirFlag        *string
	stopOnFailureFlag *bool
	verboseFlag       *int
)

func init() {
	formatFlag = Cmd.Flags().String("format", "xml", "--format:yaml, json, xml, tap")
	stopOnFailureFlag = Cmd.Flags().Bool("stop-on-failure", false, "Stop running Test Suite on first Test Case failure")
	verboseFlag = Cmd.Flags().CountP("verbose", "v", "verbose. -vv to very verbose and -vvv to very verbose with CPU Profiling")
	varFilesFlag = Cmd.Flags().StringSlice("var-from-file", []string{""}, "--var-from-file filename.yaml --var-from-file filename2.yaml: yaml, must contains a dictionnary")
	variablesFlag = Cmd.Flags().StringArray("var", nil, "--var cds='cds -f config.json' --var cds2='cds -f config.json'")
	outputDirFlag = Cmd.PersistentFlags().String("output-dir", "", "Output Directory: create tests results file inside this directory")
	libDirFlag = Cmd.PersistentFlags().String("lib-dir", "", "Lib Directory: can contain user executors. example:/etc/venom/lib:$HOME/venom.d/lib")
}

func initArgs(cmd *cobra.Command) {
	// command line flags overrides the configuration file.
	// Configuration file overrides the environment variables.
	if _, err := initFromEnv(os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	if err := initFromConfigFile(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
	cmd.LocalFlags().VisitAll(initFromCommandArguments)
}

func initFromCommandArguments(f *pflag.Flag) {
	if f.Name == "format" && formatFlag != nil {
		format = *formatFlag
	}

	if !f.Changed {
		return
	}

	switch f.Name {
	case "stop-on-failure":
		if stopOnFailureFlag != nil {
			stopOnFailure = *stopOnFailureFlag
		}
	case "output-dir":
		if outputDirFlag != nil {
			outputDir = *outputDirFlag
		}
	case "lib-dir":
		if libDirFlag != nil {
			libDir = *libDirFlag
		}
	case "verbose":
		if verboseFlag != nil {
			verbose = *verboseFlag
		}
	case "var-from-file":
		if varFilesFlag != nil {
			for _, varFile := range *varFilesFlag {
				if !isInArray(varFile, varFiles) {
					varFiles = append(varFiles, varFile)
				}
			}
		}
	case "var":
		if variablesFlag != nil {
			for _, varFlag := range *variablesFlag {
				variables = mergeVariables(varFlag, variables)
			}
		}
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func initFromConfigFile() error {
	if fileExists(".venomrc") {
		fi, err := os.Open(".venomrc")
		if err != nil {
			return err
		}
		defer fi.Close()
		return initFromReaderConfigFile(fi)
	}

	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	if fileExists(filepath.Join(home, ".venomrc")) {
		fi, err := os.Open(filepath.Join(home, ".venomrc"))
		if err != nil {
			return err
		}
		defer fi.Close()
		return initFromReaderConfigFile(fi)
	}
	return nil
}

type ConfigFileData struct {
	Format         *string   `json:"format,omitempty" yaml:"format,omitempty"`
	LibDir         *string   `json:"lib_dir,omitempty" yaml:"lib_dir,omitempty"`
	OutputDir      *string   `json:"output_dir,omitempty" yaml:"output_dir,omitempty"`
	StopOnFailure  *bool     `json:"stop_on_failure,omitempty" yaml:"stop_on_failure,omitempty"`
	Variables      *[]string `json:"variables,omitempty" yaml:"variables,omitempty"`
	VariablesFiles *[]string `json:"variables_files,omitempty" yaml:"variables_files,omitempty"`
	Verbosity      *int      `json:"verbosity,omitempty" yaml:"verbosity,omitempty"`
}

// Configuration file overrides the environment variables.
func initFromReaderConfigFile(reader io.Reader) error {
	btes, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	var configFileData ConfigFileData
	if err := yaml.Unmarshal(btes, &configFileData); err != nil {
		return err
	}

	if configFileData.Format != nil {
		format = *configFileData.Format
	}
	if configFileData.LibDir != nil {
		libDir = *configFileData.LibDir
	}
	if configFileData.OutputDir != nil {
		outputDir = *configFileData.OutputDir
	}
	if configFileData.StopOnFailure != nil {
		stopOnFailure = *configFileData.StopOnFailure
	}
	if configFileData.Variables != nil {
		for _, varFromFile := range *configFileData.Variables {
			variables = mergeVariables(varFromFile, variables)
		}
	}
	if configFileData.VariablesFiles != nil {
		for _, varFile := range *configFileData.VariablesFiles {
			if !isInArray(varFile, varFiles) {
				varFiles = append(varFiles, varFile)
			}
		}
	}
	if configFileData.Verbosity != nil {
		verbose = *configFileData.Verbosity
	}

	return nil
}

func mergeVariables(varToMerge string, existingVariables []string) []string {
	idx := strings.Index(varToMerge, "=")
	nameConfigFile := varToMerge[0:idx]
	for i, variable := range existingVariables {
		idx := strings.Index(variable, "=")
		if idx > 1 {
			nameEnv := variable[0:idx]
			if nameEnv == nameConfigFile {
				existingVariables[i] = varToMerge
				return existingVariables
			}
		}
	}
	existingVariables = append(existingVariables, varToMerge)
	return existingVariables
}

func isInArray(elt string, array []string) bool {
	for _, item := range array {
		if item == elt {
			return true
		}
	}
	return false
}

func initFromEnv(environ []string) ([]string, error) {
	if os.Getenv("VENOM_VAR") != "" {
		v := strings.Split(os.Getenv("VENOM_VAR"), " ")
		variables = v
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
			return nil, fmt.Errorf("invalid value for VENOM_STOP_ON_FAILURE")
		}
	}
	if os.Getenv("VENOM_LIB_DIR") != "" {
		libDir = os.Getenv("VENOM_LIB_DIR")
	}
	if os.Getenv("VENOM_OUTPUT_DIR") != "" {
		outputDir = os.Getenv("VENOM_OUTPUT_DIR")
	}
	if os.Getenv("VENOM_VERBOSE") != "" {
		v, err := strconv.ParseInt(os.Getenv("VENOM_VERBOSE"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for VENOM_VERBOSE, must be 1, 2 or 3")
		}
		v2 := int(v)
		verbose = v2
	}

	var cast = func(vS string) interface{} {
		var v interface{}
		_ = yaml.Unmarshal([]byte(vS), &v) //nolint
		return v
	}

	for _, env := range environ {
		if strings.HasPrefix(env, "VENOM_VAR_") {
			tuple := strings.Split(env, "=")
			k := strings.TrimPrefix(tuple[0], "VENOM_VAR_")
			variables = append(variables, fmt.Sprintf("%v=%v", k, cast(tuple[1])))
		}
	}

	return variables, nil
}

func displayArg(ctx context.Context) {
	venom.Debug(ctx, "option format=%v", format)
	venom.Debug(ctx, "option libDir=%v", libDir)
	venom.Debug(ctx, "option outputDir=%v", outputDir)
	venom.Debug(ctx, "option stopOnFailure=%v", stopOnFailure)
	venom.Debug(ctx, "option variables=%v", strings.Join(variables, " "))
	venom.Debug(ctx, "option varFiles=%v", strings.Join(varFiles, " "))
	venom.Debug(ctx, "option verbose=%v", verbose)
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
		v.RegisterExecutorBuiltin(amqp.Name, amqp.New())
		v.RegisterExecutorBuiltin(dbfixtures.Name, dbfixtures.New())
		v.RegisterExecutorBuiltin(exec.Name, exec.New())
		v.RegisterExecutorBuiltin(grpc.Name, grpc.New())
		v.RegisterExecutorBuiltin(http.Name, http.New())
		v.RegisterExecutorBuiltin(imap.Name, imap.New())
		v.RegisterExecutorBuiltin(kafka.Name, kafka.New())
		v.RegisterExecutorBuiltin(mqtt.Name, mqtt.New())
		v.RegisterExecutorBuiltin(ovhapi.Name, ovhapi.New())
		v.RegisterExecutorBuiltin(rabbitmq.Name, rabbitmq.New())
		v.RegisterExecutorBuiltin(readfile.Name, readfile.New())
		v.RegisterExecutorBuiltin(redis.Name, redis.New())
		v.RegisterExecutorBuiltin(smtp.Name, smtp.New())
		v.RegisterExecutorBuiltin(sql.Name, sql.New())
		v.RegisterExecutorBuiltin(ssh.Name, ssh.New())
		v.RegisterExecutorBuiltin(web.Name, web.New())
		v.RegisterExecutorBuiltin(tavern.Name, tavern.New())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		initArgs(cmd)

		v.OutputDir = outputDir
		v.LibDir = libDir
		v.OutputFormat = format
		v.StopOnFailure = stopOnFailure
		v.Verbose = verbose

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
		if verbose >= 2 {
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

		mapvars, err := readInitialVariables(context.Background(), variables, readers, os.Environ())
		if err != nil {
			return err
		}
		v.AddVariables(mapvars)

		start := time.Now()

		if err := v.Parse(context.Background(), path); err != nil {
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

func readInitialVariables(ctx context.Context, argsVars []string, argVarsFiles []io.Reader, environ []string) (map[string]interface{}, error) {
	var cast = func(vS string) interface{} {
		var v interface{}
		_ = yaml.Unmarshal([]byte(vS), &v) //nolint
		return v
	}

	var result = map[string]interface{}{}

	for _, r := range argVarsFiles {
		var tmpResult = map[string]interface{}{}
		btes, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(btes, &tmpResult); err != nil {
			return nil, err
		}
		for k, v := range tmpResult {
			result[k] = v
			venom.Debug(ctx, "Adding variable from vars-files %s=%s", k, v)
		}
	}

	for _, arg := range argsVars {
		if arg == "" {
			continue
		}
		tuple := strings.SplitN(arg, "=", 2)
		if len(tuple) < 2 {
			return nil, fmt.Errorf("invalid variable declaration: %v", arg)
		}
		result[tuple[0]] = cast(tuple[1])
		venom.Debug(ctx, "Adding variable from vars arg %s=%s", tuple[0], result[tuple[0]])
	}

	return result, nil
}
