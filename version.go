package loadstrike

const (
	moduleVersion          = "v0.2.1"
	runtimeProtocolVersion = 2

	runtimeCapabilityCallbackLogsV1                = "callback-logs-v1"
	runtimeCapabilityKafkaPublisherCallbackV1      = "kafka-publisher-callback-v1"
	runtimeCapabilityObservationCancellationV1     = "observation-cancellation-v1"
	runtimeCapabilityScenarioContextV2             = "scenario-context-v2"
	runtimeCapabilityScenarioRequestCancellationV1 = "scenario-request-cancellation-v1"
	runtimeCapabilityThresholdPredicateCallbackV1  = "threshold-predicate-callback-v1"
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

func runtimeBridgeCapabilities() []string {
	return []string{
		runtimeCapabilityCallbackLogsV1,
		runtimeCapabilityKafkaPublisherCallbackV1,
		runtimeCapabilityObservationCancellationV1,
		runtimeCapabilityScenarioContextV2,
		runtimeCapabilityScenarioRequestCancellationV1,
		runtimeCapabilityThresholdPredicateCallbackV1,
	}
}
