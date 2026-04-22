package loadstrike

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CorrelationStoreConfiguration mirrors the .NET-style correlation store wrapper.
type CorrelationStoreConfiguration struct {
	Kind  string                        `json:"Kind,omitempty"`
	Redis *RedisCorrelationStoreOptions `json:"Redis,omitempty"`
}

// RedisCorrelationStoreOptions mirrors the Redis-backed correlation store options wrapper.
type RedisCorrelationStoreOptions struct {
	ConnectionString string  `json:"ConnectionString,omitempty"`
	Database         int     `json:"Database,omitempty"`
	KeyPrefix        string  `json:"KeyPrefix,omitempty"`
	EntryTTLSeconds  float64 `json:"EntryTtlSeconds,omitempty"`
}

func (CorrelationStoreConfiguration) InMemory() CorrelationStoreConfiguration {
	return CorrelationStoreConfiguration{Kind: "InMemory"}
}

func (CorrelationStoreConfiguration) RedisStore(options RedisCorrelationStoreOptions) CorrelationStoreConfiguration {
	value := options
	return CorrelationStoreConfiguration{
		Kind:  "Redis",
		Redis: &value,
	}
}

// TrackingFieldSelector mirrors the public selector helper surface.
type TrackingFieldSelector struct {
	Location string `json:"Location,omitempty"`
	Path     string `json:"Path,omitempty"`
}

func (TrackingFieldSelector) Parse(value string) TrackingFieldSelector {
	selector, err := parseTrackingFieldSelector(value)
	if err != nil {
		panic(err)
	}
	return newTrackingFieldSelector(selector)
}

func (TrackingFieldSelector) TryParse(value string) (bool, TrackingFieldSelector) {
	selector, err := parseTrackingFieldSelector(value)
	if err != nil {
		return false, TrackingFieldSelector{}
	}
	return true, newTrackingFieldSelector(selector)
}

func (s TrackingFieldSelector) Extract(payload TrackingPayload) string {
	switch normalizeTrackingFieldSelectorLocation(s.Location) {
	case "header":
		for key, value := range payload.Headers {
			if key == s.Path {
				return strings.TrimSpace(value)
			}
		}
		return ""
	case "json":
		return extractTrackingJSONValue(normalizeJSONBody(payload.Body), s.Path)
	default:
		return ""
	}
}

// TrackingPayloadBuilder mirrors the public payload-builder helper surface.
type TrackingPayloadBuilder struct {
	Headers            map[string]string
	ContentType        string
	MessagePayloadType string
	body               []byte
}

func (b *TrackingPayloadBuilder) SetBody(body any) {
	if b == nil {
		return
	}
	switch typed := body.(type) {
	case nil:
		b.body = nil
	case []byte:
		b.body = append([]byte(nil), typed...)
	case string:
		b.body = []byte(typed)
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			b.body = []byte(fmt.Sprint(typed))
			return
		}
		b.body = encoded
	}
}

func (b TrackingPayloadBuilder) Build() TrackingPayload {
	headers := cloneStringMap(b.Headers)
	if headers == nil {
		headers = map[string]string{}
	}
	return TrackingPayload{
		Headers:            headers,
		Body:               append([]byte(nil), b.body...),
		ContentType:        b.ContentType,
		MessagePayloadType: b.MessagePayloadType,
	}
}

func newTrackingFieldSelector(native trackingFieldSelector) TrackingFieldSelector {
	location := "Json"
	if native.Kind == "header" {
		location = "Header"
	}
	return TrackingFieldSelector{
		Location: location,
		Path:     native.Path,
	}
}

func normalizeTrackingFieldSelectorLocation(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "header":
		return "header"
	case "json":
		return "json"
	default:
		return ""
	}
}

var trackingArrayAccessPattern = regexp.MustCompile(`^([^\[\]]+)?((?:\[\d+\])*)$`)
var trackingArrayIndexPattern = regexp.MustCompile(`\[(\d+)\]`)

func extractTrackingJSONValue(body any, path string) string {
	normalized := strings.TrimSpace(path)
	if strings.HasPrefix(normalized, "$.") {
		normalized = normalized[2:]
	} else if strings.HasPrefix(normalized, "$") {
		normalized = normalized[1:]
	}
	if normalized == "" {
		return ""
	}

	current := body
	for _, rawSegment := range strings.Split(normalized, ".") {
		segment := strings.TrimSpace(rawSegment)
		if segment == "" {
			continue
		}
		next, ok := readTrackingPathSegment(current, segment)
		if !ok || next == nil {
			return ""
		}
		current = next
	}

	switch typed := current.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func readTrackingPathSegment(current any, segment string) (any, bool) {
	if index, err := strconv.Atoi(segment); err == nil {
		return readTrackingArrayIndex(current, index)
	}

	match := trackingArrayAccessPattern.FindStringSubmatch(segment)
	if match == nil {
		return nil, false
	}

	value := current
	base := strings.TrimSpace(match[1])
	if base != "" {
		record, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		next, exists := record[base]
		if !exists {
			return nil, false
		}
		value = next
	}

	for _, indexMatch := range trackingArrayIndexPattern.FindAllStringSubmatch(match[2], -1) {
		index, err := strconv.Atoi(indexMatch[1])
		if err != nil {
			return nil, false
		}
		next, ok := readTrackingArrayIndex(value, index)
		if !ok {
			return nil, false
		}
		value = next
	}

	if base == "" && match[2] == "" {
		return nil, false
	}

	return value, true
}

func readTrackingArrayIndex(current any, index int) (any, bool) {
	rows, ok := current.([]any)
	if !ok || index < 0 || index >= len(rows) {
		return nil, false
	}
	return rows[index], true
}
