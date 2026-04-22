package loadstrike

// EndpointProduceResult is the public delegate or push callback produce result contract.
type EndpointProduceResult struct {
	IsSuccess    bool            `json:"IsSuccess,omitempty"`
	TrackingID   string          `json:"TrackingId,omitempty"`
	TimestampMS  int64           `json:"TimestampUtc,omitempty"`
	Payload      TrackingPayload `json:"Payload,omitempty"`
	ErrorMessage string          `json:"ErrorMessage,omitempty"`
}

func newEndpointProduceResult(native endpointProduceResult) EndpointProduceResult {
	return EndpointProduceResult{
		IsSuccess:    native.IsSuccess,
		TrackingID:   native.TrackingID,
		TimestampMS:  native.TimestampMS,
		Payload:      newTrackingPayloadWrapper(native.Payload),
		ErrorMessage: native.ErrorMessage,
	}
}

func (r EndpointProduceResult) toNative() endpointProduceResult {
	return endpointProduceResult{
		IsSuccess:    r.IsSuccess,
		TrackingID:   r.TrackingID,
		TimestampMS:  r.TimestampMS,
		Payload:      r.Payload.toNative(),
		ErrorMessage: r.ErrorMessage,
	}
}

// EndpointConsumeEvent is the public delegate or push callback consume event contract.
type EndpointConsumeEvent struct {
	TrackingID  string          `json:"TrackingId,omitempty"`
	TimestampMS int64           `json:"TimestampUtc,omitempty"`
	Payload     TrackingPayload `json:"Payload,omitempty"`
}

func newEndpointConsumeEvent(native endpointConsumeEvent) EndpointConsumeEvent {
	return EndpointConsumeEvent{
		TrackingID:  native.TrackingID,
		TimestampMS: native.TimestampMS,
		Payload:     newTrackingPayloadWrapper(native.Payload),
	}
}

func (e EndpointConsumeEvent) toNative() endpointConsumeEvent {
	return endpointConsumeEvent{
		TrackingID:  e.TrackingID,
		TimestampMS: e.TimestampMS,
		Payload:     e.Payload.toNative(),
	}
}
