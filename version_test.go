package loadstrike

import "testing"

func TestRuntimeMetadataExposesExactSDKAndProtocolVersions(t *testing.T) {
	if Version() == "" || Version() == "dev" {
		t.Fatalf("Version() = %q, want tagged SDK version metadata", Version())
	}
	if RuntimeProtocolVersion() <= 0 {
		t.Fatalf("RuntimeProtocolVersion() = %d, want positive protocol version", RuntimeProtocolVersion())
	}
	if RuntimeArtifactVersion() != Version() {
		t.Fatalf("RuntimeArtifactVersion() = %q, want %q", RuntimeArtifactVersion(), Version())
	}
}
