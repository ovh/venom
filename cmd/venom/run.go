package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"runtime/pprof"
	"strings"

	"github.com/ovh/venom"
	"github.com/ovh/venom/lib/cmd"
)

var (
	commonFlags = []cmd.Flag{
		{
			Name:  "log",
			Usage: "Log Level : debug, info or warn",
			IsValid: func(s string) bool {
				s = strings.ToLower(s)
				s = strings.TrimSpace(s)
				return s == "" || s == "debug" || s == "info" || s == "warn"
			},
		}, {
			Name:    _ConfigurationDir,
			Default: "~/.venom.d",
			Usage:   "Configuration directory",
		},
	}

	runCmd = cmd.Cmd{
		Name: "run",
		Desc: "Run your test integrations",
		Flags: []cmd.Flag{
			{
				Name:  "profiling",
				Usage: "Enable memory / CPU Profiling with pprof",
				Kind:  reflect.Bool,
			}, {
				Name:      "report-dir",
				ShortHand: "d",
				Usage:     "Test report directory: create tests results file inside this directory",
			}, {
				Name:      "var",
				ShortHand: "v",
				Usage:     "Variables",
				Kind:      reflect.Slice,
			}, {
				Name:  "var-from-file",
				Usage: "Load variables from json or yaml files",
				Kind:  reflect.Slice,
			}, {
				Name:      "var-from-env",
				ShortHand: "e",
				Usage:     "Load variables environment variables",
				Kind:      reflect.Bool,
			}, {
				Name:    "report-format",
				Usage:   "Test report format: xml, yml, json or tap",
				Kind:    reflect.String,
				Default: "xml",
				IsValid: func(s string) bool {
					return s == "xml" || s == "json" || s == "yml" || s == "tap"
				},
			}, {
				Name:  "stop-on-failure",
				Usage: "Stop running Test Suite on first Test Case failure",
				Kind:  reflect.Bool,
			}, {
				Name:  "no-check-variables",
				Usage: "Don't check variables before run",
				Kind:  reflect.Bool,
			}, {
				Name:  "no-strict",
				Usage: "Don't exit with error status code on failure",
				Kind:  reflect.Bool,
			}, {
				Name:    "parallel",
				Usage:   "Run tests suites in parallel",
				Kind:    reflect.Int,
				Default: "1",
			},
		},
		VariadicArgs: cmd.Arg{
			Name:       "path",
			AllowEmpty: true,
			IsValid: func(s string) bool {
				//Check if either a directory, either a yaml file
				fi, _ := os.Stat(s)
				if fi != nil {
					if fi.IsDir() {
						return true
					}
					if strings.HasPrefix(s, ".yml") {
						return true
					}
					return false
				}
				// let it go, error with be raised later
				return true
			},
		},
	}
)

func checkConfigurationDirectory(v *venom.Venom, vals cmd.Values) *cmd.Error {
	// Checks configuration directory
	configurationDirectory := vals.GetString(_ConfigurationDir)
	if configurationDirectory == "~/.venom.d" || configurationDirectory == "" {
		u, _ := user.Current()
		if u != nil {
			configurationDirectory = filepath.Join(u.HomeDir, ".venom.d")
		} else {
			configurationDirectory = filepath.Join(os.Getenv("HOME"), ".venom.d")
		}
	}
	if err := os.MkdirAll(configurationDirectory, os.FileMode(0755)); err != nil {
		return cmd.NewError(128, "unable to create directory %s: %v", configurationDirectory, err)
	}
	v.ConfigurationDirectory = configurationDirectory
	return nil
}

var runFunc = func(vals cmd.Values) *cmd.Error {
	var v = venom.New()
	v.LogLevel = vals.GetString("log")
	if err := checkConfigurationDirectory(v, vals); err != nil {
		return err
	}

	v.StopOnFailure = vals.GetBool("stop-on-failure")

	// Checks profiling
	enableProfiling := vals.GetBool("profiling")
	reportDir := vals.GetString("report-dir")
	if enableProfiling {
		filenameCPU := filepath.Join(reportDir, "pprof_cpu_profile.prof")
		filenameMem := filepath.Join(reportDir, "pprof_mem_profile.prof")
		fCPU, err := os.Create(filenameCPU)
		if err != nil {
			return cmd.NewError(129, "unable to create file %s: %v", filenameCPU, err)
		}
		fMem, err := os.Create(filenameMem)
		if err != nil {
			return cmd.NewError(130, "unable to create file %s: %v", filenameCPU, err)
		}

		pprof.StartCPUProfile(fCPU) // nolint
		p := pprof.Lookup("heap")
		defer func() {
			p.WriteTo(fMem, 1)     // nolint
			pprof.StopCPUProfile() //nolint
		}()
	}

	mods, err := v.ListModules()
	if err != nil {
		return cmd.NewError(1, "venom intialization error: unable to list installed modules: %v", err)
	}

	if len(mods) == 0 {
		return cmd.NewError(2, "venom intialization error: no module found in configuration directory")
	}

	tests, err := v.Process(context.Background(), vals.GetStringSlice("path"))
	if err != nil {
		return cmd.NewError(1, "%v", err)
	}

	v.GetLogger().Infof("Generating reports")
	if err := v.GenerateReport(*tests); err != nil {
		return cmd.NewError(2, "%v", err)
	}

	fmt.Println()

	v.GetLogger().Infof("Enjoy your tests results")
	_ = v.Output.Close()

	return nil
}
