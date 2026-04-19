package loadstrike

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuntimeUsesExactVersionCacheDirectory(t *testing.T) {
	t.Setenv(envRuntimeCacheDir, filepath.Join("C:\\", "loadstrike-cache"))

	resolver := newRuntimeArtifactResolver(runtimeResolverConfig{
		Version: RuntimeArtifactVersion(),
		GOOS:    "windows",
		GOARCH:  "amd64",
	})

	path := resolver.expectedRuntimePath()
	want := filepath.Join("runtime", RuntimeArtifactVersion(), "windows-amd64")
	if !containsNormalizedPath(path, want) {
		t.Fatalf("expectedRuntimePath() = %q, want segment %q", path, want)
	}
}

func TestVerifyDownloadedRuntimeRejectsChecksumMismatch(t *testing.T) {
	err := verifyRuntimeArtifact([]byte("runtime"), "deadbeef", "")
	if err == nil {
		t.Fatal("verifyRuntimeArtifact() unexpectedly returned nil")
	}
}

func containsNormalizedPath(path, segment string) bool {
	normalizedPath := strings.ReplaceAll(path, "\\", "/")
	normalizedSegment := strings.ReplaceAll(segment, "\\", "/")
	return strings.Contains(normalizedPath, normalizedSegment)
}
