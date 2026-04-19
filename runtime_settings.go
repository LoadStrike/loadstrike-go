package loadstrike

import (
	"os"
	"runtime"
	"strings"
)

const (
	envRuntimePath     = "LOADSTRIKE_RUNTIME_PATH"
	envRuntimeCacheDir = "LOADSTRIKE_RUNTIME_CACHE_DIR"
	envRuntimeBaseURL  = "LOADSTRIKE_RUNTIME_BASE_URL"
	envDisableDownload = "LOADSTRIKE_RUNTIME_NO_DOWNLOAD"
	envAllowLocalStub  = "LOADSTRIKE_DEV_USE_LOCAL_CLIENT"
)

func runtimeDownloadsDisabled() bool {
	return os.Getenv(envDisableDownload) == "1"
}

func runtimeBaseURL() string {
	baseURL := strings.TrimSpace(os.Getenv(envRuntimeBaseURL))
	if baseURL == "" {
		return "https://licensing.loadstrike.com"
	}
	return strings.TrimRight(baseURL, "/")
}

func runtimeGOOS() string {
	return runtime.GOOS
}

func runtimeGOARCH() string {
	return runtime.GOARCH
}
