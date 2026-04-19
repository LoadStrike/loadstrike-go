package loadstrike

import (
	"bytes"
	"encoding/json"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (r runtimeArtifactResolver) expectedRuntimePath() string {
	cacheDir := os.Getenv(envRuntimeCacheDir)
	if cacheDir == "" {
		if userCacheDir, err := os.UserCacheDir(); err == nil && userCacheDir != "" {
			cacheDir = userCacheDir
		} else {
			cacheDir = os.TempDir()
		}
	}

	fileName := "loadstrike-runtime"
	if r.config.GOOS == "windows" {
		fileName += ".exe"
	}

	return filepath.Join(
		cacheDir,
		"runtime",
		r.config.Version,
		r.config.GOOS+"-"+r.config.GOARCH,
		fileName,
	)
}

func verifyRuntimeArtifact(content []byte, expectedSHA256 string, _ string) error {
	sum := sha256.Sum256(content)
	actual := hex.EncodeToString(sum[:])
	if actual != expectedSHA256 {
		return fmt.Errorf("runtime checksum mismatch: expected %s, got %s", expectedSHA256, actual)
	}
	return nil
}

func (r runtimeArtifactResolver) resolveRuntimePath(runnerKey string) (string, error) {
	if explicitPath := strings.TrimSpace(os.Getenv(envRuntimePath)); explicitPath != "" {
		return explicitPath, nil
	}

	expectedPath := r.expectedRuntimePath()
	if _, err := os.Stat(expectedPath); err == nil {
		return expectedPath, nil
	} else if !errorsIsNotExist(err) {
		return "", fmt.Errorf("inspect cached runtime: %w", err)
	}

	if runtimeDownloadsDisabled() {
		return "", ErrRuntimeDownloadDisabled
	}

	manifest, err := r.fetchRuntimeManifest(runnerKey)
	if err != nil {
		return "", err
	}
	if manifest.Version != r.config.Version {
		return "", RuntimeMismatchError{
			ExpectedVersion: r.config.Version,
			ActualVersion:   manifest.Version,
		}
	}
	if manifest.Protocol != RuntimeProtocolVersion() {
		return "", fmt.Errorf(
			"loadstrike runtime protocol mismatch: expected %d, got %d",
			RuntimeProtocolVersion(),
			manifest.Protocol,
		)
	}

	response, err := runtimeHTTPClient().Get(manifest.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("download runtime artifact: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf("download runtime artifact: unexpected status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read runtime artifact: %w", err)
	}
	if err := verifyRuntimeArtifact(content, manifest.SHA256, manifest.Signature); err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(expectedPath), 0o755); err != nil {
		return "", fmt.Errorf("create runtime cache dir: %w", err)
	}
	if err := os.WriteFile(expectedPath, content, 0o755); err != nil {
		return "", fmt.Errorf("write runtime binary: %w", err)
	}

	return expectedPath, nil
}

func (r runtimeArtifactResolver) fetchRuntimeManifest(runnerKey string) (runtimeArtifactManifest, error) {
	payload := map[string]string{
		"runnerKey": runnerKey,
		"sdk":       "go",
		"version":   r.config.Version,
		"os":        r.config.GOOS,
		"arch":      r.config.GOARCH,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("marshal runtime manifest request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, runtimeResolveEndpoint(), bytes.NewReader(body))
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("build runtime manifest request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := runtimeHTTPClient().Do(request)
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("resolve runtime artifact: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return runtimeArtifactManifest{}, fmt.Errorf(
			"resolve runtime artifact: unexpected status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	var manifest runtimeArtifactManifest
	if err := json.NewDecoder(response.Body).Decode(&manifest); err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("decode runtime artifact manifest: %w", err)
	}

	return manifest, nil
}

func runtimeResolveEndpoint() string {
	baseURL := runtimeBaseURL()
	if strings.HasSuffix(baseURL, "/api/v1/runtime-artifacts/resolve") {
		return baseURL
	}
	return baseURL + "/api/v1/runtime-artifacts/resolve"
}

func runtimeHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
