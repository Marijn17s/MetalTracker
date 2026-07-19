package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/minio/selfupdate"
	"golang.org/x/mod/semver"

	"MetalTracker/internal/domain"
	"MetalTracker/internal/version"
)

const (
	githubOwner       = "Marijn17s"
	githubRepo        = "MetalTracker"
	checksumAssetName = "SHA256SUMS"

	KindBinary    = "binary"
	KindInstaller = "installer"
)

var (
	ErrNoRelease = errors.New("no github release published")
	ErrNoAsset   = errors.New("no update package for this platform")
)

// Pending holds download details after a successful Check.
type Pending struct {
	Version      string
	AssetName    string
	DownloadURL  string
	ChecksumHex  string
	ReleaseNotes string
	Kind         string
}

type Client struct {
	httpClient *http.Client
	owner      string
	repo       string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		owner:      githubOwner,
		repo:       githubRepo,
	}
}

func (client *Client) Check(ctx context.Context) (domain.UpdateCheckResult, *Pending, error) {
	currentDisplay := version.Display()
	current := normalizeSemver(version.Version)
	result := domain.UpdateCheckResult{
		CurrentVersion: currentDisplay,
		LatestVersion:  currentDisplay,
	}

	release, err := client.fetchLatestRelease(ctx)
	if err != nil {
		if errors.Is(err, ErrNoRelease) {
			return result, nil, nil
		}
		return result, nil, err
	}

	latest := normalizeSemver(release.TagName)
	if latest == "" {
		return result, nil, nil
	}
	result.LatestVersion = latest
	result.ReleaseNotes = strings.TrimSpace(release.Body)

	if current != "" && semver.Compare(latest, current) <= 0 {
		return result, nil, nil
	}

	assetName, kind := ExpectedAsset()
	asset, found := findAsset(release.Assets, assetName)
	if !found {
		return result, nil, fmt.Errorf("%w: need %s", ErrNoAsset, assetName)
	}

	checksumHex, err := client.fetchChecksum(ctx, release.Assets, assetName)
	if err != nil {
		return result, nil, err
	}

	result.Available = true
	result.AssetName = asset.Name

	pending := &Pending{
		Version:      latest,
		AssetName:    asset.Name,
		DownloadURL:  asset.BrowserDownloadURL,
		ChecksumHex:  checksumHex,
		ReleaseNotes: result.ReleaseNotes,
		Kind:         kind,
	}
	return result, pending, nil
}

func (client *Client) Apply(ctx context.Context, pending *Pending) error {
	if pending == nil {
		return fmt.Errorf("no pending update")
	}

	tempPath, err := client.downloadVerified(ctx, pending)
	if err != nil {
		return err
	}

	if pending.Kind == KindInstaller {
		command := openInstallerCommand(tempPath)
		if err := command.Start(); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("start installer: %w", err)
		}
		return nil
	}

	binaryPath := tempPath
	if isTarGzName(pending.AssetName) {
		extractedPath, err := extractFirstFileFromTarGz(tempPath)
		_ = os.Remove(tempPath)
		if err != nil {
			return err
		}
		binaryPath = extractedPath
	}
	defer os.Remove(binaryPath)

	file, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := selfupdate.Apply(file, selfupdate.Options{}); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}
	return nil
}

func openInstallerCommand(path string) *exec.Cmd {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", path)
	}
	return exec.Command(path)
}

func isTarGzName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")
}

func extractFirstFileFromTarGz(archivePath string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return "", fmt.Errorf("archive has no files")
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		tempFile, err := os.CreateTemp("", "metaltracker-update-bin-*")
		if err != nil {
			return "", err
		}
		tempPath := tempFile.Name()
		if _, err := io.Copy(tempFile, tarReader); err != nil {
			tempFile.Close()
			_ = os.Remove(tempPath)
			return "", err
		}
		if err := tempFile.Close(); err != nil {
			_ = os.Remove(tempPath)
			return "", err
		}
		if err := os.Chmod(tempPath, 0o755); err != nil {
			_ = os.Remove(tempPath)
			return "", err
		}
		return tempPath, nil
	}
}

func (client *Client) downloadVerified(ctx context.Context, pending *Pending) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pending.DownloadURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "MetalTracker/"+version.Display())

	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %d", response.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "metaltracker-update-*")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()

	hasher := sha256.New()
	writer := io.MultiWriter(tempFile, hasher)
	if _, err := io.Copy(writer, response.Body); err != nil {
		tempFile.Close()
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}

	actual := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actual, pending.ChecksumHex) {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("checksum mismatch for %s", pending.AssetName)
	}

	finalPath := tempPath
	extension := filepath.Ext(pending.AssetName)
	if extension != "" {
		finalPath = tempPath + extension
		if err := os.Rename(tempPath, finalPath); err != nil {
			_ = os.Remove(tempPath)
			return "", err
		}
	}

	return finalPath, nil
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Body    string        `json:"body"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (client *Client) fetchLatestRelease(ctx context.Context) (githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", client.owner, client.repo)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "MetalTracker/"+version.Display())

	response, err := client.httpClient.Do(request)
	if err != nil {
		return githubRelease{}, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return githubRelease{}, ErrNoRelease
	}
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return githubRelease{}, fmt.Errorf("github releases: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return githubRelease{}, err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return githubRelease{}, ErrNoRelease
	}
	return release, nil
}

func findAsset(assets []githubAsset, wantName string) (githubAsset, bool) {
	for _, asset := range assets {
		if asset.Name == wantName {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func (client *Client) fetchChecksum(ctx context.Context, assets []githubAsset, assetName string) (string, error) {
	sumsAsset, found := findAsset(assets, checksumAssetName)
	if !found {
		return "", fmt.Errorf("SHA256SUMS missing from release")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sumsAsset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "MetalTracker/"+version.Display())
	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SHA256SUMS download failed: HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	checksum, ok := parseChecksum(string(body), assetName)
	if !ok {
		return "", fmt.Errorf("checksum not found for %s", assetName)
	}
	return checksum, nil
}

func parseChecksum(content string, assetName string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[len(fields)-1]
		name = strings.TrimPrefix(name, "*")
		if name == assetName || filepath.Base(name) == assetName {
			return strings.ToLower(fields[0]), true
		}
	}
	return "", false
}

func normalizeSemver(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "dev") {
		return ""
	}
	if !strings.HasPrefix(trimmed, "v") {
		trimmed = "v" + trimmed
	}
	if !semver.IsValid(trimmed) {
		return ""
	}
	return trimmed
}

// ExpectedAsset returns the release asset name and install kind for this build.
func ExpectedAsset() (name string, kind string) {
	switch runtime.GOOS {
	case "windows":
		return "MetalTracker-Windows-x64-Setup.exe", KindInstaller
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "MetalTracker-macOS-AppleSilicon.dmg", KindInstaller
		}
		return "MetalTracker-macOS-Intel.dmg", KindInstaller
	default:
		return "MetalTracker-Linux-x64.tar.gz", KindBinary
	}
}

// ExpectedAssetName returns the release asset filename for this platform.
func ExpectedAssetName() string {
	name, _ := ExpectedAsset()
	return name
}
