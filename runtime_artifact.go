package loadstrike

import (
	"net/http"
	"time"
)

// runtimeArtifactManifest is the strictly decoded, publisher-authenticated
// descriptor for the exact execution component compatible with this SDK.
type runtimeArtifactManifest struct {
	SchemaVersion uint32                     `json:"schemaVersion"`
	Claims        runtimeArtifactClaims      `json:"claims"`
	Signatures    []runtimeArtifactSignature `json:"signatures"`
	canonicalJSON []byte
}

type runtimeArtifactClaims struct {
	Arch                 string `json:"arch"`
	AttestationDigest    string `json:"attestationDigest"`
	AttestationMediaType string `json:"attestationMediaType"`
	BuildFlags           string `json:"buildFlags"`
	ByteLength           uint64 `json:"byteLength"`
	DownloadURL          string `json:"downloadUrl"`
	Executable           string `json:"executable"`
	IssuedAt             uint64 `json:"issuedAt"`
	ManifestVersion      uint32 `json:"manifestVersion"`
	NotBefore            uint64 `json:"notBefore"`
	OS                   string `json:"os"`
	Protocol             uint32 `json:"protocol"`
	Publisher            string `json:"publisher"`
	ReleaseID            string `json:"releaseId"`
	Repository           string `json:"repository"`
	RunAttempt           uint32 `json:"runAttempt"`
	RunID                uint64 `json:"runId"`
	RuntimeVersion       string `json:"runtimeVersion"`
	SDK                  string `json:"sdk"`
	SHA256               string `json:"sha256"`
	SourceRef            string `json:"sourceRef"`
	SourceSHA            string `json:"sourceSha"`
	Toolchain            string `json:"toolchain"`
	ValidUntil           uint64 `json:"validUntil"`
	WorkflowIdentity     string `json:"workflowIdentity"`
	WrapperCommit        string `json:"wrapperCommit"`
	WrapperGoModSum      string `json:"wrapperGoModSum"`
	WrapperModule        string `json:"wrapperModule"`
	WrapperSum           string `json:"wrapperSum"`
	WrapperTagObject     string `json:"wrapperTagObject"`
	WrapperTree          string `json:"wrapperTree"`
	WrapperVersion       string `json:"wrapperVersion"`
}

type runtimeArtifactSignature struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
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
