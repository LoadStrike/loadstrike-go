package loadstrike

const (
	// LoadStrikeTraceIDHeader is the message header LoadStrike can generate for tracking.
	LoadStrikeTraceIDHeader = "loadstrike-trace-id"

	// LoadStrikeTraceIDTrackingField is the tracking selector for LoadStrikeTraceIDHeader.
	LoadStrikeTraceIDTrackingField = "header:" + LoadStrikeTraceIDHeader
)
