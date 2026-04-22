package loadstrike

import (
	"encoding/json"
	"fmt"
)

// TrackingPayload is the public cross-platform tracking payload contract.
type TrackingPayload struct {
	Headers            map[string]string `json:"Headers,omitempty"`
	Body               any               `json:"Body,omitempty"`
	ProducedUTC        string            `json:"ProducedUtc,omitempty"`
	ConsumedUTC        string            `json:"ConsumedUtc,omitempty"`
	ContentType        string            `json:"ContentType,omitempty"`
	MessagePayloadType string            `json:"MessagePayloadType,omitempty"`
}

func newTrackingPayloadWrapper(native trackingPayload) TrackingPayload {
	return TrackingPayload{
		Headers:            cloneStringMap(native.Headers),
		Body:               native.Body,
		ProducedUTC:        native.ProducedUTC,
		ConsumedUTC:        native.ConsumedUTC,
		ContentType:        native.ContentType,
		MessagePayloadType: native.MessagePayloadType,
	}
}

func (p TrackingPayload) toNative() trackingPayload {
	return trackingPayload{
		Headers:            cloneStringMap(p.Headers),
		Body:               p.Body,
		ProducedUTC:        p.ProducedUTC,
		ConsumedUTC:        p.ConsumedUTC,
		ContentType:        p.ContentType,
		MessagePayloadType: p.MessagePayloadType,
	}
}

func (p TrackingPayload) GetBodyAsUtf8() string {
	switch typed := p.Body.(type) {
	case nil:
		return ""
	case []byte:
		return string(typed)
	case string:
		return typed
	default:
		body, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(body)
	}
}
