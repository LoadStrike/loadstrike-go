package loadstrike

// RuntimeMismatchError is returned when the resolved runtime does not match the SDK version.
type RuntimeMismatchError struct {
	ExpectedVersion string
	ActualVersion   string
}

// Error returns the current error text. Use this when you need the readable failure message.
func (e RuntimeMismatchError) Error() string {
	return "loadstrike runtime version mismatch: expected " + e.ExpectedVersion + ", got " + e.ActualVersion
}
