package loadstrike

import "testing"

func TestCallbackRegistryRejectsUnknownCallbackID(t *testing.T) {
	registry := newRuntimeCallbackRegistry()
	if _, ok := registry.lookup("missing"); ok {
		t.Fatal("lookup returned ok for missing callback")
	}
}
