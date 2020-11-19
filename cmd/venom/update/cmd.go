package update

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
	"github.com/inconshreveable/go-update"
	"github.com/ovh/venom"
	"github.com/ovh/venom/cmd"
	"github.com/spf13/cobra"
)

var urlGitubReleases = "https://github.com/ovh/venom/releases"

// Cmd update
var Cmd = &cobra.Command{
	Use:   "update",
	Short: "Update venom to the latest release version: venom update",
	Long:  `venom update`,
	Run: func(cmd *cobra.Command, args []string) {
		doUpdate()
	},
}

func getURLArtifactFromGithub() string {
	client := github.NewClient(nil)
	release, resp, err := client.Repositories.GetLatestRelease(context.TODO(), "ovh", "venom")
	if err != nil {
		cmd.Exit("Repositories.GetLatestRelease returned error: %v\n%v", err, resp.Body)
	}

	if *release.TagName == venom.Version {
		cmd.Exit(fmt.Sprintf("you already have the latest release: %s", *release.TagName))
	}

	if len(release.Assets) > 0 {
		for _, asset := range release.Assets {
			assetName := strings.ReplaceAll(*asset.Name, ".", "-")
			current := fmt.Sprintf("venom-%s-%s", runtime.GOOS, runtime.GOARCH)
			if assetName == current {
				return *asset.BrowserDownloadURL
			}
		}
	}

	text := "Invalid Artifacts on latest release. Please try again in few minutes.\n"
	text += "If the problem persists, please open an issue on https://github.com/ovh/venom/issues\n"
	cmd.Exit(text)
	return ""
}

func getContentType(resp *http.Response) string {
	for k, v := range resp.Header {
		if k == "Content-Type" && len(v) >= 1 {
			return v[0]
		}
	}
	return ""
}

func doUpdate() {
	url := getURLArtifactFromGithub()
	fmt.Printf("Url to update venom: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		cmd.Exit("Error when downloading venom from url %s: %v\n", url, err)
	}

	if contentType := getContentType(resp); contentType != "application/octet-stream" {
		fmt.Printf("Url: %s\n", url)
		cmd.Exit("Invalid Binary (Content-Type: %s). Please try again or download it manually from %s\n", contentType, urlGitubReleases)
	}

	if resp.StatusCode != 200 {
		cmd.Exit("Error http code: %d, url called: %s\n", resp.StatusCode, url)
	}

	fmt.Printf("Getting latest release from: %s ...\n", url)
	defer resp.Body.Close()
	if err = update.Apply(resp.Body, update.Options{}); err != nil {
		cmd.Exit("Error when updating venom from url: %s err:%s\n", url, err.Error())
	}
	fmt.Println("Update done.")
}
