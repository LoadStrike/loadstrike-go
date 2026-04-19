package loadstrike

// runtimeArtifactManifest describes the exact runtime artifact the public SDK can download.
type runtimeArtifactManifest struct {
	Version        string `json:"version"`
	Protocol       int    `json:"protocolVersion"`
	DownloadURL    string `json:"downloadUrl"`
	SHA256         string `json:"sha256"`
	Signature      string `json:"signature"`
	ExecutableName string `json:"executableName"`
}

type runtimeResolverConfig struct {
	Version string
	GOOS    string
	GOARCH  string
}

type runtimeArtifactResolver struct {
	config runtimeResolverConfig
}

func newRuntimeArtifactResolver(config runtimeResolverConfig) runtimeArtifactResolver {
	return runtimeArtifactResolver{config: config}
}
