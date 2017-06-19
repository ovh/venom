package run

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/runabove/venom"
	"github.com/runabove/venom/context/default"
	"github.com/runabove/venom/context/webctx"
	"github.com/runabove/venom/executors/exec"
	"github.com/runabove/venom/executors/http"
	"github.com/runabove/venom/executors/imap"
	"github.com/runabove/venom/executors/readfile"
	"github.com/runabove/venom/executors/smtp"
	"github.com/runabove/venom/executors/ssh"
	"github.com/runabove/venom/executors/web"
)

var (
	path           []string
	variables      []string
	exclude        []string
	format         string
	withEnv        bool
	parallel       int
	logLevel       string
	outputDir      string
	detailsLevel   string
	resumeFailures bool
	resume         bool
)

func init() {
	Cmd.Flags().StringSliceVarP(&variables, "var", "", []string{""}, "--var cds='cds -f config.json' --var cds2='cds -f config.json'")
	Cmd.Flags().StringSliceVarP(&exclude, "exclude", "", []string{""}, "--exclude filaA.yaml --exclude filaB.yaml --exclude fileC*.yaml")
	Cmd.Flags().StringVarP(&format, "format", "", "xml", "--format:yaml, json, xml")
	Cmd.Flags().BoolVarP(&withEnv, "env", "", true, "Inject environment variables. export FOO=BAR -> you can use {{.FOO}} in your tests")
	Cmd.Flags().IntVarP(&parallel, "parallel", "", 1, "--parallel=2 : launches 2 Test Suites in parallel")
	Cmd.PersistentFlags().StringVarP(&logLevel, "log", "", "warn", "Log Level : debug, info or warn")
	Cmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "", "", "Output Directory: create tests results file inside this directory")
	Cmd.PersistentFlags().StringVarP(&detailsLevel, "details", "", "medium", "Output Details Level : low, medium, high")
	Cmd.PersistentFlags().BoolVarP(&resume, "resume", "", true, "Output Resume: one line with Total, TotalOK, TotalKO, TotalSkipped, TotalTestSuite")
	Cmd.PersistentFlags().BoolVarP(&resumeFailures, "resumeFailures", "", true, "Output Resume Failures")
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

		venom.RegisterExecutor(exec.Name, exec.New())
		venom.RegisterExecutor(http.Name, http.New())
		venom.RegisterExecutor(imap.Name, imap.New())
		venom.RegisterExecutor(readfile.Name, readfile.New())
		venom.RegisterExecutor(smtp.Name, smtp.New())
		venom.RegisterExecutor(ssh.Name, ssh.New())
		venom.RegisterExecutor(web.Name, web.New())

		// Register Context
		venom.RegisterTestCaseContext(defaultctx.Name, defaultctx.New())
		venom.RegisterTestCaseContext(webctx.Name, webctx.New())
	},
	Run: func(cmd *cobra.Command, args []string) {
		if parallel < 0 {
			parallel = 1
		}

		mapvars := make(map[string]string)

		if withEnv {
			variables = append(variables, os.Environ()...)
		}

		for _, a := range variables {
			t := strings.SplitN(a, "=", 2)
			if len(t) < 2 {
				continue
			}
			mapvars[t[0]] = strings.Join(t[1:], "")
		}

		start := time.Now()
		tests, err := venom.Process(path, mapvars, exclude, parallel, logLevel, detailsLevel, os.Stdout)
		if err != nil {
			log.Fatal(err)
		}

		elapsed := time.Since(start)
		if err := venom.OutputResult(format, resume, resumeFailures, outputDir, *tests, elapsed, detailsLevel); err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
}
