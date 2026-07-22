package loadstrike

import (
	"net/http"
	"time"
)

// runtimeArtifactManifest describes the exact execution component compatible with this SDK.
type runtimeArtifactManifest struct {
	Version          string    `json:"version"`
	Protocol         int       `json:"protocolVersion"`
	DownloadURL      string    `json:"downloadUrl"`
	SHA256           string    `json:"sha256"`
	Signature        string    `json:"signature"`
	SignatureVersion string    `json:"signatureVersion"`
	ExpiresUTC       time.Time `json:"expiresUtc"`
	ExecutableName   string    `json:"executableName"`
}

type runtimeResolverConfig struct {
	Version string
	GOOS    string
	GOARCH  string
}

type runtimeArtifactResolver struct {
	config          runtimeResolverConfig
	cacheDir        string
	resolveEndpoint string
	httpClient      *http.Client
	now             func() time.Time
}

func newRuntimeArtifactResolver(config runtimeResolverConfig) runtimeArtifactResolver {
	return runtimeArtifactResolver{config: config}
}
