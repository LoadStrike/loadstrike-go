package loadstrike

const (
	moduleVersion          = "v0.1.19501"
	runtimeProtocolVersion = 1
)

// Version returns the public SDK version embedded in this package.
func Version() string {
	return moduleVersion
}

// RuntimeArtifactVersion returns the exact runtime version this package will resolve.
func RuntimeArtifactVersion() string {
	return moduleVersion
}

// RuntimeProtocolVersion returns the host/runtime RPC protocol version.
func RuntimeProtocolVersion() int {
	return runtimeProtocolVersion
}
