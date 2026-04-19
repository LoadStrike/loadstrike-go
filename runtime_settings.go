package loadstrike

import "os"

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
