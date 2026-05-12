package update

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"

	"github.com/ovh/venom"
	"github.com/ovh/venom/cmd"
)

var urlGitubReleases = "https://github.com/ovh/venom/releases"

const checksumsAssetName = "checksums.txt"

// Cmd update
var Cmd = &cobra.Command{
	Use:   "update",
	Short: "Update venom to the latest release version: venom update",
	Long:  `venom update`,
	Run: func(cmd *cobra.Command, args []string) {
		doUpdate()
	},
}

// releaseAssets returns, for the latest release, the download URL of the
// binary for the current OS/arch and the URL of the checksums.txt file.
func releaseAssets() (binaryURL, checksumsURL, binaryAssetName string) {
	client := github.NewClient(nil)
	release, resp, err := client.Repositories.GetLatestRelease(context.TODO(), "ovh", "venom")
	if err != nil {
		cmd.Exit("Repositories.GetLatestRelease returned error: %v\n%v", err, resp.Body)
	}

	if *release.TagName == venom.Version {
		cmd.Exit("you already have the latest release: %s", *release.TagName)
	}

	current := fmt.Sprintf("venom-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		normalised := strings.ReplaceAll(*asset.Name, ".", "-")
		if normalised == current {
			binaryURL = *asset.BrowserDownloadURL
			binaryAssetName = *asset.Name
		}
		if *asset.Name == checksumsAssetName {
			checksumsURL = *asset.BrowserDownloadURL
		}
	}

	if binaryURL == "" {
		const text = `Invalid Artifacts on latest release. Please try again in few minutes.
If the problem persists, please open an issue on https://github.com/ovh/venom/issues
`
		cmd.Exit(text)
	}
	return binaryURL, checksumsURL, binaryAssetName
}

// fetchExpectedChecksum downloads checksums.txt from the release and
// returns the expected SHA256 (hex) for assetName. The checksums.txt
// format is the standard `sha256sum` output: "<hex>  <filename>" lines.
func fetchExpectedChecksum(checksumsURL, assetName string) (string, error) {
	if checksumsURL == "" {
		return "", fmt.Errorf("this release does not publish %s; refusing to update for security reasons", checksumsAssetName)
	}
	resp, err := http.Get(checksumsURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", checksumsAssetName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch %s: HTTP %d", checksumsAssetName, resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Accept both "<hex>  <name>" (two spaces) and "<hex> <name>"
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		// The filename in checksums.txt may be a relative path; match on basename.
		name := fields[1]
		if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
			name = name[idx+1:]
		}
		if name == assetName {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error parsing %s: %w", checksumsAssetName, err)
	}
	return "", fmt.Errorf("no checksum entry for asset %q in %s", assetName, checksumsAssetName)
}

// downloadAndVerify reads the response body fully into memory, verifies its
// SHA256 against expectedHex (constant-time compare), and returns a Reader
// suitable for update.Apply. Memory cost is acceptable: venom binaries are
// only a few tens of MB.
func downloadAndVerify(body io.Reader, expectedHex string) (io.Reader, error) {
	hasher := sha256.New()
	var buf bytes.Buffer
	if _, err := io.Copy(io.MultiWriter(&buf, hasher), body); err != nil {
		return nil, fmt.Errorf("failed to download binary: %w", err)
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	expected, err := hex.DecodeString(expectedHex)
	if err != nil {
		return nil, fmt.Errorf("invalid expected checksum %q: %w", expectedHex, err)
	}
	gotBytes := hasher.Sum(nil)
	if subtle.ConstantTimeCompare(gotBytes, expected) != 1 {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHex, got)
	}
	return &buf, nil
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
	binaryURL, checksumsURL, assetName := releaseAssets()
	fmt.Printf("Url to update venom: %s\n", binaryURL)

	expected, err := fetchExpectedChecksum(checksumsURL, assetName)
	if err != nil {
		cmd.Exit("%s\nDownload the binary manually from %s if needed.\n", err.Error(), urlGitubReleases)
	}

	resp, err := http.Get(binaryURL)
	if err != nil {
		cmd.Exit("Error when downloading venom from url %s: %v\n", binaryURL, err)
	}
	defer resp.Body.Close()

	if contentType := getContentType(resp); contentType != "application/octet-stream" {
		fmt.Printf("Url: %s\n", binaryURL)
		cmd.Exit("Invalid Binary (Content-Type: %s). Please try again or download it manually from %s\n", contentType, urlGitubReleases)
	}

	if resp.StatusCode != 200 {
		cmd.Exit("Error http code: %d, url called: %s\n", resp.StatusCode, binaryURL)
	}

	fmt.Printf("Getting latest release from: %s ...\n", binaryURL)
	verified, err := downloadAndVerify(resp.Body, expected)
	if err != nil {
		cmd.Exit("Error when verifying venom binary from url: %s err:%s\n", binaryURL, err.Error())
	}

	if err = update.Apply(verified, update.Options{}); err != nil {
		cmd.Exit("Error when updating venom from url: %s err:%s\n", binaryURL, err.Error())
	}
	fmt.Println("Update done.")
}
