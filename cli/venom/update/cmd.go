package update

import (
	"fmt"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"

	"github.com/runabove/venom"
)

// used by CI to inject architecture (linux-amd64, etc...) at build time
var architecture string
var urlGitubReleases = "https://github.com/runabove/venom/releases"

// Cmd update
var Cmd = &cobra.Command{
	Use:   "update",
	Short: "Update venom to the latest release version: venom update",
	Long:  `venom update`,
	Run: func(cmd *cobra.Command, args []string) {
		doUpdate("", architecture)
	},
}

func getURLArtifactFromGithub(architecture string) string {
	client := github.NewClient(nil)
	release, resp, err := client.Repositories.GetLatestRelease("runabove", "venom")
	if err != nil {
		venom.Exit("Repositories.GetLatestRelease returned error: %v\n%v", err, resp.Body)
	}

	if len(release.Assets) > 0 {
		for _, asset := range release.Assets {
			if *asset.Name == "venom-"+architecture {
				return *asset.BrowserDownloadURL
			}
		}
	}

	text := "Invalid Artifacts on latest release. Please try again in few minutes.\n"
	text += "If the problem persists, please open an issue on https://github.com/runabove/venom/issues\n"
	venom.Exit(text)
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

func doUpdate(baseurl, architecture string) {
	if architecture == "" {
		text := "You seem to have a custom build of venom.\n"
		text += "Please download latest release on %s\n"
		venom.Exit(text, urlGitubReleases)
	}

	url := getURLArtifactFromGithub(architecture)
	fmt.Printf("Url to update venom: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		venom.Exit("Error when downloading venom from url: %s, err:%s\n", url, err.Error())
	}

	if contentType := getContentType(resp); contentType != "application/octet-stream" {
		fmt.Printf("Url: %s\n", url)
		venom.Exit("Invalid Binary (Content-Type: %s). Please try again or download it manually from %s\n", contentType, urlGitubReleases)
	}

	if resp.StatusCode != 200 {
		venom.Exit("Error http code: %d, url called: %s\n", resp.StatusCode, url)
	}

	fmt.Printf("Getting latest release from : %s ...\n", url)
	defer resp.Body.Close()
	if err = update.Apply(resp.Body, update.Options{}); err != nil {
		venom.Exit("Error when updating venom from url: %s err:%s\n", url, err.Error())
	}
	fmt.Println("Update done.")
}
