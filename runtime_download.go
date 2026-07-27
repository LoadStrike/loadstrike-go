package loadstrike

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func (r runtimeArtifactResolver) expectedRuntimePath() string {
	cacheDir, err := selectRuntimeCacheRoot(r.cacheDir, os.UserCacheDir)
	if err != nil {
		return ""
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

func verifyRuntimeManifest(
	manifest runtimeArtifactManifest,
	config runtimeResolverConfig,
	now time.Time,
) error {
	keys, err := configuredRuntimePublisherKeyring()
	if err != nil {
		return err
	}
	if len(manifest.canonicalJSON) == 0 {
		return errors.New(
			"runtime manifest canonical descriptor bytes are unavailable",
		)
	}
	if _, err := verifyRuntimeManifestV2(
		manifest.canonicalJSON,
		config,
		keys,
		now,
	); err != nil {
		return err
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
	if !isLowerHex64(expectedSHA256) {
		return errors.New("runtime checksum must be exactly 64 lowercase hexadecimal characters")
	}
	object, err := openStableRuntimeObject(
		path,
		false,
		true,
		true,
		false,
	)
	if err != nil {
		return err
	}
	if err := verifyRuntimeCacheFileMode(
		path,
		object.info,
		runtimeCacheRuntimePermissions,
	); err != nil {
		_ = object.Close()
		return err
	}
	if err := protectOpenedRuntimeObject(
		object.file,
		false,
		runtimeCacheRuntimePermissions,
	); err != nil {
		_ = object.Close()
		return err
	}
	if err := object.Close(); err != nil {
		return err
	}
	object, err = openStableRuntimeObject(
		path,
		false,
		true,
		true,
		true,
	)
	if err != nil {
		return err
	}
	defer object.Close()
	actual, err := hashOpenedRuntimeObject(object)
	if err != nil {
		return fmt.Errorf("hash cached runtime: %w", err)
	}
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
	if _, err := configuredRuntimePublisherKeyring(); err != nil {
		return "", fmt.Errorf("validate runtime publisher trust: %w", err)
	}

	cacheRoot, err := selectRuntimeCacheRoot(r.cacheDir, os.UserCacheDir)
	if err != nil {
		return "", err
	}
	expectedPath := r.expectedRuntimePath()
	if expectedPath == "" {
		return "", errors.New("resolve per-user runtime cache path")
	}
	if err := ensureRuntimeCacheDirectory(
		cacheRoot,
		filepath.Dir(expectedPath),
	); err != nil {
		return "", err
	}
	manifestPath := r.expectedRuntimeManifestPath()
	now := r.currentTime()
	transportPolicy, err := r.transportPolicy()
	if err != nil {
		return "", err
	}

	if manifest, err := readRuntimeManifestSidecar(manifestPath); err == nil {
		if _, err := transportPolicy.validateArtifactURL(
			manifest.Claims.DownloadURL,
		); err == nil {
			if err := verifyRuntimeManifest(manifest, r.config, now); err == nil {
				if err := verifyRuntimeArtifactFile(
					expectedPath,
					manifest.Claims.SHA256,
				); err == nil {
					return expectedPath, nil
				}
				return r.downloadAndInstallRuntime(manifest)
			}
		}
	}

	manifest, err := r.fetchRuntimeManifest(runnerKey)
	if err != nil {
		return "", err
	}
	if err := verifyRuntimeManifest(manifest, r.config, now); err != nil {
		return "", fmt.Errorf("authenticate runtime manifest: %w", err)
	}
	if _, err := transportPolicy.validateArtifactURL(
		manifest.Claims.DownloadURL,
	); err != nil {
		return "", err
	}

	if err := verifyRuntimeArtifactFile(
		expectedPath,
		manifest.Claims.SHA256,
	); err == nil {
		if err := writeRuntimeManifestSidecar(
			cacheRoot,
			manifestPath,
			manifest,
		); err != nil {
			return "", err
		}
		if err := verifyRuntimeArtifactFile(
			expectedPath,
			manifest.Claims.SHA256,
		); err != nil {
			return "", fmt.Errorf("revalidate cached runtime after manifest migration: %w", err)
		}
		return expectedPath, nil
	}

	return r.downloadAndInstallRuntime(manifest)
}

func (r runtimeArtifactResolver) resolveRuntimeExecution(
	runnerKey string,
) (*resolvedRuntimeExecution, error) {
	runtimePath, err := r.resolveRuntimePath(runnerKey)
	if err != nil {
		return nil, err
	}
	manifest, err := readRuntimeManifestSidecar(r.expectedRuntimeManifestPath())
	if err != nil {
		return nil, fmt.Errorf("read authenticated runtime manifest before execution: %w", err)
	}
	if err := verifyRuntimeManifest(
		manifest,
		r.config,
		r.currentTime(),
	); err != nil {
		return nil, fmt.Errorf("authenticate runtime manifest before execution: %w", err)
	}
	return prepareRuntimeExecutionCopy(runtimePath, manifest.Claims.SHA256, nil)
}

func (r runtimeArtifactResolver) downloadAndInstallRuntime(manifest runtimeArtifactManifest) (string, error) {
	transportPolicy, err := r.transportPolicy()
	if err != nil {
		return "", err
	}
	artifactURL, err := transportPolicy.validateArtifactURL(
		manifest.Claims.DownloadURL,
	)
	if err != nil {
		return "", err
	}
	response, err := r.client().Get(artifactURL.String())
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return "", fmt.Errorf(
			"download runtime artifact: %s",
			sanitizeRuntimeDiagnostic(err.Error()),
		)
	}
	defer response.Body.Close()
	if err := validateRuntimeResponseURL(
		response,
		artifactURL,
		transportPolicy,
		"runtime artifact download",
	); err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		message := sanitizeRuntimeDiagnostic(string(body))
		clearAutopilotRunnerKeyBytes(body)
		return "", fmt.Errorf(
			"download runtime artifact: unexpected status %d: %s",
			response.StatusCode,
			message,
		)
	}

	expectedPath := r.expectedRuntimePath()
	directory := filepath.Dir(expectedPath)
	cacheRoot, err := selectRuntimeCacheRoot(r.cacheDir, os.UserCacheDir)
	if err != nil {
		return "", err
	}
	if err := ensureRuntimeCacheDirectory(cacheRoot, directory); err != nil {
		return "", err
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

	if err := runtimeTemp.Chmod(runtimeCacheSidecarPermissions); err != nil {
		return "", fmt.Errorf("secure runtime download temp file: %w", err)
	}
	if manifest.Claims.ByteLength < 1 ||
		manifest.Claims.ByteLength > maxRuntimeArtifactBytes {
		return "", errors.New(
			"runtime artifact signed byte length is outside the safety limit",
		)
	}
	expectedLength := int64(manifest.Claims.ByteLength)
	if response.ContentLength >= 0 &&
		response.ContentLength != expectedLength {
		return "", fmt.Errorf(
			"runtime artifact length mismatch: expected %d bytes, got %d",
			expectedLength,
			response.ContentLength,
		)
	}
	hasher := sha256.New()
	written, err := io.Copy(
		io.MultiWriter(runtimeTemp, hasher),
		io.LimitReader(response.Body, expectedLength+1),
	)
	if err != nil {
		return "", fmt.Errorf(
			"read runtime artifact: %s",
			sanitizeRuntimeDiagnostic(err.Error()),
		)
	}
	if written != expectedLength {
		return "", fmt.Errorf(
			"runtime artifact length mismatch: expected %d bytes, got %d",
			expectedLength,
			written,
		)
	}
	actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA256 != manifest.Claims.SHA256 {
		return "", fmt.Errorf(
			"runtime checksum mismatch: expected %s, got %s",
			manifest.Claims.SHA256,
			actualSHA256,
		)
	}
	if err := runtimeTemp.Sync(); err != nil {
		return "", fmt.Errorf("sync runtime download: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := runtimeTemp.Chmod(runtimeCacheRuntimePermissions); err != nil {
			return "", fmt.Errorf("mark runtime artifact executable: %w", err)
		}
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
	if err := protectRuntimeObject(
		expectedPath,
		false,
		runtimeCacheRuntimePermissions,
	); err != nil {
		return "", fmt.Errorf("protect installed runtime binary: %w", err)
	}
	if err := atomicReplaceFile(manifestTempPath, r.expectedRuntimeManifestPath()); err != nil {
		return "", fmt.Errorf("install runtime manifest sidecar: %w", err)
	}
	if err := protectRuntimeObject(
		r.expectedRuntimeManifestPath(),
		false,
		runtimeCacheSidecarPermissions,
	); err != nil {
		return "", fmt.Errorf("protect installed runtime manifest sidecar: %w", err)
	}
	if err := syncRuntimeCacheDirectory(directory); err != nil {
		return "", fmt.Errorf("sync runtime cache directory: %w", err)
	}
	if err := verifyRuntimeArtifactFile(
		expectedPath,
		manifest.Claims.SHA256,
	); err != nil {
		return "", fmt.Errorf("revalidate installed runtime: %w", err)
	}

	return expectedPath, nil
}

func readRuntimeManifestSidecar(path string) (runtimeArtifactManifest, error) {
	object, err := openStableRuntimeObject(
		path,
		false,
		true,
		true,
		false,
	)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if object.info.Size() > maxRuntimeManifestBytes {
		_ = object.Close()
		return runtimeArtifactManifest{}, errors.New("runtime manifest sidecar exceeds the safety limit")
	}
	if err := verifyRuntimeCacheFileMode(
		path,
		object.info,
		runtimeCacheSidecarPermissions,
	); err != nil {
		_ = object.Close()
		return runtimeArtifactManifest{}, err
	}
	if err := protectOpenedRuntimeObject(
		object.file,
		false,
		runtimeCacheSidecarPermissions,
	); err != nil {
		_ = object.Close()
		return runtimeArtifactManifest{}, err
	}
	if err := object.Close(); err != nil {
		return runtimeArtifactManifest{}, err
	}
	object, err = openStableRuntimeObject(
		path,
		false,
		true,
		true,
		true,
	)
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	defer object.Close()
	if object.info.Size() > maxRuntimeManifestBytes {
		return runtimeArtifactManifest{}, errors.New("runtime manifest sidecar exceeds the safety limit")
	}
	manifest, err := decodeRuntimeManifest(object.file, "runtime manifest sidecar")
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if err := object.ensureStable(); err != nil {
		return runtimeArtifactManifest{}, err
	}
	return manifest, nil
}

func decodeRuntimeManifest(reader io.Reader, source string) (runtimeArtifactManifest, error) {
	content, err := io.ReadAll(
		io.LimitReader(reader, maxRuntimeManifestBytes+1),
	)
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("read %s: %w", source, err)
	}
	if len(content) > maxRuntimeManifestBytes {
		return runtimeArtifactManifest{}, fmt.Errorf(
			"%s exceeds the %d byte safety limit",
			source,
			maxRuntimeManifestBytes,
		)
	}
	manifest, err := parseRuntimeManifestV2(content)
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("decode %s: %w", source, err)
	}
	return manifest, nil
}

func prepareRuntimeManifestSidecar(directory string, manifest runtimeArtifactManifest) (string, error) {
	if len(manifest.canonicalJSON) < 1 ||
		len(manifest.canonicalJSON) > maxRuntimeManifestBytes {
		return "", errors.New(
			"runtime manifest canonical descriptor bytes are unavailable",
		)
	}
	content := append([]byte(nil), manifest.canonicalJSON...)

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

func writeRuntimeManifestSidecar(
	cacheRoot string,
	path string,
	manifest runtimeArtifactManifest,
) error {
	directory := filepath.Dir(path)
	cacheRoot, err := selectRuntimeCacheRoot(cacheRoot, os.UserCacheDir)
	if err != nil {
		return err
	}
	if err := ensureRuntimeCacheDirectory(cacheRoot, directory); err != nil {
		return err
	}
	tempPath, err := prepareRuntimeManifestSidecar(directory, manifest)
	if err != nil {
		return err
	}
	defer os.Remove(tempPath)
	if err := atomicReplaceFile(tempPath, path); err != nil {
		return fmt.Errorf("install runtime manifest sidecar: %w", err)
	}
	if err := protectRuntimeObject(path, false, runtimeCacheSidecarPermissions); err != nil {
		return fmt.Errorf("protect runtime manifest sidecar: %w", err)
	}
	if err := syncRuntimeCacheDirectory(directory); err != nil {
		return fmt.Errorf("sync runtime cache directory: %w", err)
	}
	return nil
}

func (r runtimeArtifactResolver) fetchRuntimeManifest(runnerKey string) (runtimeArtifactManifest, error) {
	transportPolicy, err := r.transportPolicy()
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	resolveURL, err := transportPolicy.validateResolveURL(r.endpoint())
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
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
	if len(body) > 4096 {
		clearAutopilotRunnerKeyBytes(body)
		return runtimeArtifactManifest{}, errors.New(
			"runtime manifest request exceeds the 4096 byte safety limit",
		)
	}
	payload["runnerKey"] = ""
	defer clearAutopilotRunnerKeyBytes(body)

	request, err := http.NewRequest(http.MethodPost, resolveURL.String(), bytes.NewReader(body))
	if err != nil {
		return runtimeArtifactManifest{}, fmt.Errorf("build runtime manifest request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := r.client().Do(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		message := sanitizeRuntimeDiagnostic(err.Error(), runnerKey)
		return runtimeArtifactManifest{}, fmt.Errorf("resolve runtime artifact: %s", message)
	}
	defer response.Body.Close()
	if err := validateRuntimeResponseURL(
		response,
		resolveURL,
		transportPolicy,
		"runtime artifact resolve",
	); err != nil {
		return runtimeArtifactManifest{}, err
	}

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		message := sanitizeRuntimeDiagnostic(string(responseBody), runnerKey)
		clearAutopilotRunnerKeyBytes(responseBody)
		return runtimeArtifactManifest{}, fmt.Errorf(
			"resolve runtime artifact: unexpected status %d: %s",
			response.StatusCode,
			message,
		)
	}
	mediaType, parameters, err := mime.ParseMediaType(
		response.Header.Get("Content-Type"),
	)
	if err != nil ||
		mediaType != "application/json" ||
		len(parameters) != 0 {
		return runtimeArtifactManifest{}, errors.New(
			"runtime artifact manifest response must use Content-Type application/json",
		)
	}

	manifest, err := decodeRuntimeManifest(response.Body, "runtime artifact manifest")
	if err != nil {
		return runtimeArtifactManifest{}, err
	}
	if _, err := transportPolicy.validateArtifactURL(
		manifest.Claims.DownloadURL,
	); err != nil {
		return runtimeArtifactManifest{}, err
	}
	return manifest, nil
}

func (r runtimeArtifactResolver) endpoint() string {
	if strings.TrimSpace(r.resolveEndpoint) != "" {
		return r.resolveEndpoint
	}
	return buildURL(defaultLicenseValidationBaseURL, "/api/v1/runtime-artifacts/resolve")
}

func (r runtimeArtifactResolver) client() *http.Client {
	var client http.Client
	if r.httpClient != nil {
		client = *r.httpClient
	} else {
		client = *runtimeHTTPClient()
	}
	client.CheckRedirect = rejectRuntimeArtifactRedirect
	return &client
}

func (r runtimeArtifactResolver) transportPolicy() (runtimeTransportPolicy, error) {
	hasEndpoint := strings.TrimSpace(r.resolveEndpoint) != ""
	hasClient := r.httpClient != nil
	if hasEndpoint || hasClient {
		if !hasEndpoint || !hasClient || !isGoTestBinary() {
			return runtimeTransportPolicy{}, errors.New(
				"explicit runtime artifact transport is allowed only for internal tests with both an endpoint and client",
			)
		}
		return newRuntimeTransportPolicy(r.resolveEndpoint, true)
	}
	return newRuntimeTransportPolicy(defaultLicenseValidationBaseURL, false)
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
	return &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: rejectRuntimeArtifactRedirect,
	}
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
