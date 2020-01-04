package venom

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"

	yaml "gopkg.in/yaml.v2"
)

type VenomModuleManifest struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	Author           string `json:"author"`
	InstallationPath string `json:"-"`
	Description      string `json:"description"`
	Homepage         string `json:"homepage"`
	URL              string `json:"url"`
	Version          string `json:"version"`
}

type VenomModule interface {
	Manifest() VenomModuleManifest
}

func (v *Venom) ListModules() ([]VenomModule, error) {
	ls, err := ioutil.ReadDir(v.ConfigurationDirectory)
	if err != nil {
		return nil, fmt.Errorf("unable to open venom configuration directory: %v", err)
	}
	var dirs []os.FileInfo
	for _, fi := range ls {
		if fi.IsDir() {
			dirs = append(dirs, fi)
		}
	}

	var modList []VenomModule
	for _, dir := range dirs {
		modulePath := filepath.Join(v.ConfigurationDirectory, dir.Name())
		mod, err := getModule(modulePath)
		if err != nil {
			v.logger.Errorf(err.Error())
			continue
		}
		modList = append(modList, mod)
	}
	return modList, nil
}

type moduleFileNameChecker struct {
	regex *regexp.Regexp
	os    func(s ...string) string
	arch  func(s ...string) string
}

var (
	modulesChekers = []moduleFileNameChecker{
		{
			regex: regexp.MustCompile(`^([a-zA-Z-\+\d]+)_([a-zA-Z-\+\d]+)_([a-zA-Z-\+\d]+)(\.exe)?$`),
			os: func(s ...string) string {
				//	r, _ := regexp.Compile(`^([a-zA-Z-\+\d]+)_([a-zA-Z-\+\d]+)_([a-zA-Z-\+\d]+)(\.exe)?$`)
				//	fmt.Println(r.FindStringSubmatch("venom_windows_amd64.exe"))
				// 	output:
				//		[venom_windows_amd64.exe venom windows amd64 .exe]
				if len(s) < 4 || len(s) > 5 {
					return ""
				}
				return s[2]
			},
			arch: func(s ...string) string {
				if len(s) < 4 || len(s) > 5 {
					return ""
				}
				return s[3]
			},
		}, {
			regex: regexp.MustCompile(`^([a-zA-Z-\+\d]+)_([a-zA-Z-\+\d]+)(\.exe)?$`),
			os: func(s ...string) string {
				if len(s) < 3 || len(s) > 4 {
					return ""
				}
				return s[2]
			},
			arch: nil,
		}, {
			regex: regexp.MustCompile(`^([a-zA-Z-\+\d]+)(\.exe)?$`),
			os:    nil,
			arch:  nil,
		},
	}
)

func getModule(modulePath string) (VenomModule, error) {
	dir, err := os.Stat(modulePath)
	if err != nil {
		return nil, fmt.Errorf("%s is not a venom module: %v", modulePath, err)
	}
	name := filepath.Base(dir.Name())

	ls, err := ioutil.ReadDir(modulePath)
	if err != nil {
		return nil, fmt.Errorf("%s is not a venom module: %v", dir.Name(), err)
	}

	sort.Slice(ls, func(i, j int) bool {
		return len(ls[i].Name()) > len(ls[j].Name())
	})

	var moduleEntryPointPath string

loop:
	for _, fi := range ls {
		fName := filepath.Base(fi.Name())

		for _, chk := range modulesChekers {
			if !chk.regex.MatchString(fName) {
				continue loop
			}
			if chk.os != nil {
				os := chk.os(chk.regex.FindStringSubmatch(fName)...)
				if os != runtime.GOOS {
					continue loop
				}
			}
			if chk.arch != nil {
				arch := chk.arch(chk.regex.FindStringSubmatch(fName)...)
				if arch != runtime.GOARCH {
					continue loop
				}
			}
			if fi.IsDir() {
				continue
			}
			isExecutable := fi.Mode().Perm()&0111 != 0
			if !isExecutable {
				// Check if it is really executable
				return nil, fmt.Errorf("%s is not a venom module: %s is not executable", name, fi.Name())
			}

			moduleEntryPointPath = filepath.Join(modulePath, fi.Name())

			break loop
		}
		return nil, fmt.Errorf("%s is not a venom module: no suitable entry point has been found", name)
	}

	manifest, err := getModuleManifest(moduleEntryPointPath)
	if err != nil {
		return nil, err
	}

	moduleEntryPointPath, err = filepath.Abs(moduleEntryPointPath)
	if err != nil {
		return nil, err
	}

	mod := newModule(moduleEntryPointPath, manifest)
	return mod, err
}

func getModuleManifest(moduleEntryPointPath string) (VenomModuleManifest, error) {
	var manifest VenomModuleManifest
	cmd := exec.CommandContext(context.Background(), moduleEntryPointPath, "info") // nolint

	output, err := cmd.CombinedOutput()
	if err != nil {
		return manifest, err
	}

	if err := yaml.Unmarshal(output, &manifest); err != nil {
		return manifest, fmt.Errorf("module error: unexpected output: %s: %v", string(output), err)
	}

	return manifest, err
}

func newModule(moduleEntryPointPath string, manifest VenomModuleManifest) VenomModule {
	var mod VenomModule
	switch manifest.Type {
	case "executor":
		return executorModule{
			entrypoint: moduleEntryPointPath,
			manifest:   manifest,
		}
		//case "context":
		//	return contextModule{
		//		entrypoint: moduleEntryPointPath,
		//		manifest:   manifest,
		//	}
	}
	return mod
}
