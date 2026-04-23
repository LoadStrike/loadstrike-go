package loadstrike

import (
	"encoding/json"
	"fmt"
	"strings"
)

// trackingPayload is the internal transport shape shared by the public wrapper
// adapters and host callback contracts. Execution/correlation engines live in
// the private runtime.
type trackingPayload struct {
	Headers            map[string]string `json:"headers,omitempty"`
	Body               any               `json:"body,omitempty"`
	ProducedUTC        string            `json:"producedUtc,omitempty"`
	ConsumedUTC        string            `json:"consumedUtc,omitempty"`
	ContentType        string            `json:"contentType,omitempty"`
	MessagePayloadType string            `json:"messagePayloadType,omitempty"`
}

type trackingFieldSelector struct {
	Kind string
	Path string
}

func parseTrackingFieldSelector(value string) (trackingFieldSelector, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return trackingFieldSelector{}, fmt.Errorf("tracking field selector must be in kind:path format")
	}

	kind := strings.ToLower(strings.TrimSpace(parts[0]))
	path := strings.TrimSpace(parts[1])
	if path == "" {
		return trackingFieldSelector{}, fmt.Errorf("tracking field selector path is required")
	}

	switch kind {
	case "header", "json":
		return trackingFieldSelector{Kind: kind, Path: path}, nil
	default:
		return trackingFieldSelector{}, fmt.Errorf("tracking field selector kind must be header or json")
	}
}

func normalizeJSONBody(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		if len(typed) == 0 {
			return map[string]any{}
		}
		var decoded any
		if err := json.Unmarshal(typed, &decoded); err == nil {
			return decoded
		}
		return string(typed)
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return map[string]any{}
		}
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return decoded
		}
		return typed
	default:
		return typed
	}
}
