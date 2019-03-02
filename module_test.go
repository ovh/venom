package venom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getModule(t *testing.T) {

	tt := []struct {
		name     string
		filename string
		hasError bool
		mod      VenomModule
	}{
		{
			name:     "plain text file should failed",
			filename: "venom.go",
			hasError: true,
			mod:      nil,
		}, {
			name:     "no file should failed",
			filename: "foobarbiz",
			hasError: true,
			mod:      nil,
		}, {
			name:     "no entrypoint",
			filename: "tests/fixtures/test-module-should-failed-with-no-entrypoint",
			hasError: true,
			mod:      nil,
		}, {
			name:     "no entrypoint.1",
			filename: "tests/fixtures/test-module-should-failed-with-no-entrypoint.1",
			hasError: true,
			mod:      nil,
		}, {
			name:     "not executable file should failed",
			filename: "tests/fixtures/test-module-should-failed",
			hasError: true,
			mod:      nil,
		}, {
			name:     "should succeed",
			filename: "tests/fixtures/test-module-should-succeed",
			hasError: false,
			mod: executorModule{
				entrypoint: "tests/fixtures/test-module-should-succeed/test-module_linux_amd64",
				manifest: VenomModuleManifest{
					Type:             "executor",
					Name:             "dummy",
					Author:           "François SAMIN",
					InstallationPath: "",
					Description:      "this is a dummy executor",
					Homepage:         "http://nowhere.lolcat",
					URL:              "http://nowhere.lolcat",
					Version:          "0.1",
				},
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			mod, err := getModule(test.filename)
			if test.hasError {
				assert.Error(t, err)
				t.Logf("error is: %v", err)

			} else {
				assert.NoError(t, err)
			}

			if test.mod == nil {
				assert.Nil(t, mod)
			} else {
				expectedManifest := test.mod.Manifest()
				actualManifest := mod.Manifest()
				t.Logf("manifest is: %+v", actualManifest)

				assert.EqualValues(t, expectedManifest, actualManifest)
			}
		})
	}
}

func TestVenom_ListModules(t *testing.T) {
	v := New()
	v.ConfigurationDirectory = "tests/fixtures"
	v.init()
	mods, err := v.ListModules()
	assert.NoError(t, err)
	assert.Len(t, mods, 1)
}
