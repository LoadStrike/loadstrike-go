package loadstrike

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	runtimeManifestSignatureVersion = "runner-hmac-sha256-v1"
	runtimeManifestDomain           = "LS-RUNTIME-MANIFEST-V1"
	maxRuntimeManifestBytes         = 64 * 1024
	maxRuntimeArtifactBytes         = 512 * 1024 * 1024
)

func (r runtimeArtifactResolver) expectedRuntimePath() string {
	cacheDir := strings.TrimSpace(r.cacheDir)
	if cacheDir == "" {
		var err error
		cacheDir, err = os.UserCacheDir()
		if err != nil || cacheDir == "" {
			cacheDir = os.TempDir()
		}
	}

	return filepath.Join(
		cacheDir,
		"runtime",
		r.config.Version,
		r.config.GOOS+"-"+r.config.GOARCH,
		r.expectedRuntimeExecutableName(),
	)
}

func (r runtimeArtifactResolver) expectedRuntimeExecutableName() string {
	if r.config.GOOS == "windows" {
		return "loadstrike-runtime.exe"
	}
	return "loadstrike-runtime"
}

func (r runtimeArtifactResolver) expectedRuntimeManifestPath() string {
	return r.expectedRuntimePath() + ".manifest.json"
}

func runtimeManifestCanonicalPayload(manifest runtimeArtifactManifest, config runtimeResolverConfig) string {
	return strings.Join([]string{
		runtimeManifestDomain,
		runtimeManifestSignatureVersion,
		"go",
		config.Version,
		config.GOOS,
		config.GOARCH,
		strconv.Itoa(manifest.Protocol),
		manifest.SHA256,
		manifest.DownloadURL,
		manifest.ExecutableName,
		strconv.FormatInt(manifest.ExpiresUTC.Unix(), 10),
	}, "\n")
}

func verifyRuntimeManifest(
	manifest runtimeArtifactManifest,
	config runtimeResolverConfig,
	runnerKey string,
	now time.Time,
) error {
	runnerKey = strings.TrimSpace(runnerKey)
	if runnerKey == "" {
		return errors.New("runner key is required to authenticate the runtime manifest")
	}
	if manifest.SignatureVersion != runtimeManifestSignatureVersion {
		return fmt.Errorf(
			"runtime manifest signature version mismatch: expected %s, got %q",
			runtimeManifestSignatureVersion,
			manifest.SignatureVersion,
		)
	}
	if manifest.Version != config.Version {
		return RuntimeMismatchError{ExpectedVersion: config.Version, ActualVersion: manifest.Version}
	}
	if manifest.Protocol != RuntimeProtocolVersion() {
		return fmt.Errorf(
			"loadstrike runtime protocol mismatch: expected %d, got %d",
			RuntimeProtocolVersion(),
			manifest.Protocol,
		)
	}
	expectedExecutableName := "loadstrike-runtime"
	if config.GOOS == "windows" {
		expectedExecutableName += ".exe"
	}
	if manifest.ExecutableName != expectedExecutableName {
		return fmt.Errorf(
			"runtime manifest executable name mismatch: expected %q, got %q",
			expectedExecutableName,
			manifest.ExecutableName,
		)
	}
	if !isLowerHex64(manifest.SHA256) {
		return errors.New("runtime manifest checksum must be exactly 64 lowercase hexadecimal characters")
	}
	if manifest.ExpiresUTC.IsZero() || !manifest.ExpiresUTC.After(now.UTC()) {
		return errors.New("runtime manifest has expired")
	}
	parsedURL, err := url.Parse(manifest.DownloadURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.User != nil || parsedURL.Fragment != "" {
		return errors.New("runtime manifest download URL must be an absolute HTTP or HTTPS URL without credentials or a fragment")
	}
	if !isLowerHex64(manifest.Signature) {
		return errors.New("runtime manifest signature encoding must be exactly 64 lowercase hexadecimal characters")
	}

	providedSignature, err := hex.DecodeString(manifest.Signature)
	if err != nil {
		return errors.New("runtime manifest signature encoding is invalid")
	}
	mac := hmac.New(sha256.New, []byte(runnerKey))
	_, _ = mac.Write([]byte(runtimeManifestCanonicalPayload(manifest, config)))
	if !hmac.Equal(providedSignature, mac.Sum(nil)) {
		return errors.New("runtime manifest signature verification failed")
	}
	return nil
}

func verifyRuntimeArtifact(content []byte, expectedSHA256 string) error {
	sum := sha256.Sum256(content)
	actual := hex.EncodeToString(sum[:])
	if actual != expectedSHA256 {
		return fmt.Errorf("runtime checksum mismatch: expected %s, got %s", expectedSHA256, actual)
	}
	return nil
}

func verifyRuntimeArtifactFile(path string, expectedSHA256 string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("cached runtime is not a regular file: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("hash cached runtime: %w", err)
	}
	actual := hex.EncodeToString(hasher.Sum(nil))
	if actual != expectedSHA256 {
		return fmt.Errorf("runtime checksum mismatch: expected %s, got %s", expectedSHA256, actual)
	}
	return nil
}

func (r runtimeArtifactResolver) resolveRuntimePath(runnerKey string) (string, error) {
	runnerKey = strings.TrimSpace(runnerKey)
	if runnerKey == "" {
		return "", errors.New("runner key is required to resolve the compatible execution component")
	}

	expectedPath := r.expectedRuntimePath()
	manifestPath := r.expectedRuntimeManifestPath()
	now := r.currentTime()

	if manifest, err := readRuntimeManifestSidecar(manifestPath); err == nil {
		if err := verifyRuntimeManifest(manifest, r.config, runnerKey, now); err == nil {
			if err := verifyRuntimeArtifactFile(expectedPath, manifest.SHA256); err == nil {
				return expectedPath, nil
			}
			return r.downloadAndInstallRuntime(manifest)
		}
	}

	manifest, err := r.fetchRuntimeManifest(runnerKey)
	if err != nil {
		return "", err
	}
	if err := verifyRuntimeManifest(manifest, r.config, runnerKey, now); err != nil {
		return "", fmt.Errorf("authenticate runtime manifest: %w", err)
	}

	if err := verifyRuntimeArtifactFile(expectedPath, manifest.SHA256); err == nil {
		if err := writeRuntimeManifestSidecar(manifestPath, manifest); err != nil {
			return "", err
		}
		if err := verifyRuntimeArtifactFile(expectedPath, manifest.SHA256); err != nil {
			return "", fmt.Errorf("revalidate cached runtime after manifest migration: %w", err)
		}
		return expectedPath, nil
	}

	return r.downloadAndInstallRuntime(manifest)
}

func (r runtimeArtifactResolver) downloadAndInstallRuntime(manifest runtimeArtifactManifest) (string, error) {
	response, err := r.client().Get(manifest.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("download runtime artifact: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf(
			"download runtime artifact: unexpected status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	expectedPath := r.expectedRuntimePath()
	directory := filepath.Dir(expectedPath)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("create runtime cache dir: %w", err)
	}

	runtimeTemp, err := os.CreateTemp(directory, ".loadstrike-runtime-download-*")
	if err != nil {
		return "", fmt.Errorf("create runtime download temp file: %w", err)
	}
	runtimeTempPath := runtimeTemp.Name()
	defer os.Remove(runtimeTempPath)
	closed := false
	defer func() {
		if !closed {
			_ = runtimeTemp.Close()
		}
	}()

	if err := runtimeTemp.Chmod(0o700); err != nil {
		return "", fmt.Errorf("secure runtime download temp file: %w", err)
	}
	hasher := sha256.New()
	written, err := io.Copy(
		io.MultiWriter(runtimeTemp, hasher),
		io.LimitReader(response.Body, maxRuntimeArtifactBytes+1),
	)
	if err != nil {
		return "", fmt.Errorf("read runtime artifact: %w", err)
	}
	if written > maxRuntimeArtifactBytes {
		return "", fmt.Errorf("runtime artifact exceeds the %d byte safety limit", maxRuntimeArtifactBytes)
	}
	actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA256 != manifest.SHA256 {
		return "", fmt.Errorf(
			"runtime checksum mismatch: expected %s, got %s",
			manifest.SHA256,
			actualSHA256,
		)
	}
	if err := runtimeTemp.Sync(); err != nil {
		return "", fmt.Errorf("sync runtime download: %w", err)
	}
	if err := runtimeTemp.Chmod(0o755); err != nil {
		return "", fmt.Errorf("mark runtime artifact executable: %w", err)
	}
	if err := runtimeTemp.Close(); err != nil {
		return "", fmt.Errorf("close runtime download: %w", err)
	}
	closed = true

	manifestTempPath, err := prepareRuntimeManifestSidecar(directory, manifest)
	if err != nil {
		return "", err
	}
	defer os.Remove(manifestTempPath)

	if err := atomicReplaceFile(runtimeTempPath, expectedPath); err != nil {
		return "", fmt.Errorf("install runtime binary: %w", err)
	}
	if err := atomicReplaceFile(manifestTempPath, r.expectedRuntimeManifestPath()); err != nil {
		return "", fmt.Errorf("install runtime manifest sidecar: %w", err)
	}
	if err := syncRuntimeCacheDirectory(directory); err != nil {
		return "", fmt.Errorf("sync runtime cache directory: %w", err)
	}
	if err := verifyRuntimeArtifactFile(expectedPath, manifest.SHA256); err != nil {
		return "", fmt.Errorf("revalidate installed runtime: %w", err)
	}

	return expectedPath, nil
}

func readRuntimeManifestSidecar(path string) (runtimeArtifactManifest, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if !info.Mode().IsRegular() {
		return runtimeArtifactManifest{}, fmt.Errorf("runtime manifest sidecar is not a regular file: %s", path)
	}
	if info.Size() > maxRuntimeManifestBytes {
		return runtimeArtifactManifest{}, errors.New("runtime manifest sidecar exceeds the safety limit")
	}

	file, err := os.Open(path)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	defer file.Close()

	return decodeRuntimeManifest(file, "runtime manifest sidecar")
}

func decodeRuntimeManifest(reader io.Reader, source string) (runtimeArtifactManifest, error) {
	content, err := io.ReadAll(io.LimitReader(reader, maxRuntimeManifestBytes+1))
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("read %s: %w", source, err)
	}
	if len(content) > maxRuntimeManifestBytes {
		return runtimeArtifactManifest{}, fmt.Errorf("%s exceeds the %d byte safety limit", source, maxRuntimeManifestBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	var manifest runtimeArtifactManifest
	if err := decoder.Decode(&manifest); err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("decode %s: %w", source, err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return runtimeArtifactManifest{}, fmt.Errorf("decode %s: trailing JSON content is not allowed", source)
		}
		return runtimeArtifactManifest{}, fmt.Errorf("decode %s: invalid trailing content: %w", source, err)
	}

	return manifest, nil
}

func prepareRuntimeManifestSidecar(directory string, manifest runtimeArtifactManifest) (string, error) {
	content, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("encode runtime manifest sidecar: %w", err)
	}
	content = append(content, '\n')

	temp, err := os.CreateTemp(directory, ".loadstrike-runtime-manifest-*")
	if err != nil {
		return "", fmt.Errorf("create runtime manifest temp file: %w", err)
	}
	tempPath := temp.Name()
	remove := true
	defer func() {
		_ = temp.Close()
		if remove {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		return "", fmt.Errorf("secure runtime manifest temp file: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		return "", fmt.Errorf("write runtime manifest temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return "", fmt.Errorf("sync runtime manifest temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close runtime manifest temp file: %w", err)
	}
	remove = false
	return tempPath, nil
}

func writeRuntimeManifestSidecar(path string, manifest runtimeArtifactManifest) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create runtime cache dir: %w", err)
	}
	tempPath, err := prepareRuntimeManifestSidecar(directory, manifest)
	if err != nil {
		return err
	}
	defer os.Remove(tempPath)
	if err := atomicReplaceFile(tempPath, path); err != nil {
		return fmt.Errorf("install runtime manifest sidecar: %w", err)
	}
	if err := syncRuntimeCacheDirectory(directory); err != nil {
		return fmt.Errorf("sync runtime cache directory: %w", err)
	}
	return nil
}

func (r runtimeArtifactResolver) fetchRuntimeManifest(runnerKey string) (runtimeArtifactManifest, error) {
	payload := map[string]string{
		"runnerKey":                strings.TrimSpace(runnerKey),
		"sdk":                      "go",
		"version":                  r.config.Version,
		"os":                       r.config.GOOS,
		"arch":                     r.config.GOARCH,
		"manifestSignatureVersion": runtimeManifestSignatureVersion,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("marshal runtime manifest request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, r.endpoint(), bytes.NewReader(body))
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("build runtime manifest request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := r.client().Do(request)
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

	return decodeRuntimeManifest(response.Body, "runtime artifact manifest")
}

func (r runtimeArtifactResolver) endpoint() string {
	if strings.TrimSpace(r.resolveEndpoint) != "" {
		return r.resolveEndpoint
	}
	return runtimeResolveEndpoint()
}

func (r runtimeArtifactResolver) client() *http.Client {
	if r.httpClient != nil {
		return r.httpClient
	}
	return runtimeHTTPClient()
}

func (r runtimeArtifactResolver) currentTime() time.Time {
	if r.now != nil {
		return r.now().UTC()
	}
	return time.Now().UTC()
}

func runtimeResolveEndpoint() string {
	return buildURL(resolveLicensingAPIBaseURL(), "/api/v1/runtime-artifacts/resolve")
}

func runtimeHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func isLowerHex64(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}
