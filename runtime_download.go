package loadstrike

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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
