package run

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rockbears/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

var (
	path []string
	v    *venom.Venom

	variables     []string
	format        string = "xml" // Set the default value for formatFlag
	varFiles      []string
	outputDir     string
	libDir        string
	htmlReport    bool
	stopOnFailure bool
	verbose       int = 0 // Set the default value for verboseFlag

	variablesFlag     *[]string
	formatFlag        *string
	varFilesFlag      *[]string
	outputDirFlag     *string
	libDirFlag        *string
	stopOnFailureFlag *bool
	htmlReportFlag    *bool
	verboseFlag       *int
)

func init() {
	formatFlag = Cmd.Flags().String("format", "xml", "--format:json, tap, xml, yaml")
	stopOnFailureFlag = Cmd.Flags().Bool("stop-on-failure", false, "Stop running Test Suite on first Test Case failure")
	htmlReportFlag = Cmd.Flags().Bool("html-report", false, "Generate HTML Report")
	verboseFlag = Cmd.Flags().CountP("verbose", "v", "verbose. -v (INFO level in venom.log file), -vv to very verbose (DEBUG level) and -vvv to very verbose with CPU Profiling")
	varFilesFlag = Cmd.Flags().StringSlice("var-from-file", []string{""}, "--var-from-file filename.yaml --var-from-file filename2.yaml: yaml, must contains a dictionary")
	variablesFlag = Cmd.Flags().StringArray("var", nil, "--var cds='cds -f config.json' --var cds2='cds -f config.json'")
	outputDirFlag = Cmd.PersistentFlags().String("output-dir", "", "Output Directory: create tests results file inside this directory")
	libDirFlag = Cmd.PersistentFlags().String("lib-dir", "", "Lib Directory: can contain user executors. example:/etc/venom/lib:$HOME/venom.d/lib")
}

func initArgs(cmd *cobra.Command) {
	// command line flags overrides the configuration file.
	// Configuration file overrides the environment variables.
	if _, err := initFromEnv(os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		venom.OSExit(2)
	}

	if err := initFromConfigFile(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		venom.OSExit(2)
	}
	cmd.LocalFlags().VisitAll(initFromCommandArguments)
}

func initFromCommandArguments(f *pflag.Flag) {
	if !f.Changed {
		return
	}

	switch f.Name {
	case "format":
		if formatFlag != nil {
			format = *formatFlag
		}
	case "stop-on-failure":
		if stopOnFailureFlag != nil {
			stopOnFailure = *stopOnFailureFlag
		}
	case "html-report":
		if htmlReportFlag != nil {
			htmlReport = *htmlReportFlag
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
	HtmlReport     *bool     `json:"html_report,omitempty" yaml:"html_report,omitempty"`
	Variables      *[]string `json:"variables,omitempty" yaml:"variables,omitempty"`
	VariablesFiles *[]string `json:"variables_files,omitempty" yaml:"variables_files,omitempty"`
	Verbosity      *int      `json:"verbosity,omitempty" yaml:"verbosity,omitempty"`
}

// Configuration file overrides the environment variables.
func initFromReaderConfigFile(reader io.Reader) error {
	btes, err := io.ReadAll(reader)

	if err != nil && err != io.EOF {
		return err
	}

	var configFileData ConfigFileData
	if err := yaml.Unmarshal(btes, &configFileData); err != nil && err != io.EOF {
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
	if configFileData.HtmlReport != nil {
		htmlReport = *configFileData.HtmlReport
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
	if os.Getenv("VENOM_HTML_REPORT") != "" {
		var err error
		htmlReport, err = strconv.ParseBool(os.Getenv("VENOM_HTML_REPORT"))
		if err != nil {
			return nil, fmt.Errorf("invalid value for VENOM_HTML_REPORT")
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
	venom.Debug(ctx, "option htmlReport=%v", htmlReport)
	venom.Debug(ctx, "option variables=%v", strings.Join(variables, " "))
	venom.Debug(ctx, "option varFiles=%v", strings.Join(varFiles, " "))
	venom.Debug(ctx, "option verbose=%v", verbose)
}

// Cmd run
var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run Tests",
	Example: `  Run all testsuites containing in files ending with *.yml or *.yaml: venom run
  Run a single testsuite: venom run mytestfile.yml
  Run a single testsuite and export the result in JSON format in test/ folder: venom run mytestfile.yml --format=json --output-dir=test
  Run a single testsuite and export the result in XML and HTML formats in test/ folder: venom run mytestfile.yml --format=xml --output-dir=test --html-report
  Run a single testsuite and specify a variable: venom run mytestfile.yml --var="foo=bar"
  Run a single testsuite and load all variables from a file: venom run mytestfile.yml --var-from-file variables.yaml
  Run all testsuites containing in files ending with *.yml or *.yaml with verbosity: VENOM_VERBOSE=2 venom run
  
  Notice that variables initialized with -var-from-file argument can be overrided with -var argument
  
  More info: https://github.com/ovh/venom`,
	Long: `run integration tests`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			path = append(path, ".")
		} else {
			path = args[0:]
		}

		v = venom.New()
		for name, executorFunc := range executors.Registry {
			v.RegisterExecutorBuiltin(name, executorFunc())
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		initArgs(cmd)

		v.OutputDir = outputDir
		v.LibDir = libDir
		v.OutputFormat = format
		v.StopOnFailure = stopOnFailure
		v.HtmlReport = htmlReport
		v.Verbose = verbose

		if err := v.InitLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			venom.OSExit(2)
		}

		if v.Verbose == 3 {
			fCPU, err := os.Create(filepath.Join(v.OutputDir, "pprof_cpu_profile.prof"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "error while create profile file %v\n", err)
				venom.OSExit(2)
			}
			fMem, err := os.Create(filepath.Join(v.OutputDir, "pprof_mem_profile.prof"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "error while create profile file %v\n", err)
				venom.OSExit(2)
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
				fmt.Fprintf(os.Stderr, "unable to open var-from-file %s: %v\n", f, err)
				venom.OSExit(2)
			}
			defer fi.Close()
			readers = append(readers, fi)
		}

		mapvars, err := readInitialVariables(context.Background(), variables, readers, os.Environ())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			venom.OSExit(2)
		}
		v.AddVariables(mapvars)

		if err := v.Parse(context.Background(), path); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			venom.OSExit(2)
		}

		if err := v.Process(context.Background(), path); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			venom.OSExit(2)
		}

		if err := v.OutputResult(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			venom.OSExit(2)
		}

		if v.Tests.Status == venom.StatusPass {
			fmt.Fprintf(os.Stdout, "final status: %v\n", venom.Green(v.Tests.Status))
			venom.OSExit(0)
		}
		fmt.Fprintf(os.Stdout, "final status: %v\n", venom.Red(v.Tests.Status))
		venom.OSExit(2)

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
		btes, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}

		stemp, err := interpolate.Do(string(btes), nil)
		if err != nil {
			return nil, errors.Wrap(err, "unable to interpolate file")
		}

		if err := yaml.Unmarshal([]byte(stemp), &tmpResult); err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal file")
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
		stemp, err := interpolate.Do(tuple[1], nil)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to interpolate arg %s", arg)
		}
		result[tuple[0]] = cast(stemp)
		venom.Debug(ctx, "Adding variable from vars arg %s=%s", tuple[0], result[tuple[0]])
	}

	return result, nil
}
