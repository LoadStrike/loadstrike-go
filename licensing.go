package loadstrike

import (
	"bytes"
	stdcontext "context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	defaultLicenseValidationBaseURL        = "https://licensing.loadstrike.com"
	licenseValidationBaseURLEnv            = "LOADSTRIKE_INTERNAL_BLACKBOX_API_BASE_URL"
	licenseEnforcementDisabledForTestsEnv  = "LOADSTRIKE_DISABLE_LICENSE_ENFORCEMENT_FOR_TESTS"
	defaultLicenseValidationTimeoutSeconds = 10.0
)

var (
	licenseSigningKeyCache   sync.Map
	builtInWorkerPluginNames = map[string]struct{}{
		"loadstrike failed responses": {},
		"loadstrike correlation":      {},
	}
	sinkFeatureByKind = map[string]string{
		"influxdb":      "extensions.reporting_sinks.influxdb",
		"timescaledb":   "extensions.reporting_sinks.timescaledb",
		"grafanaloki":   "extensions.reporting_sinks.grafana_loki",
		"datadog":       "extensions.reporting_sinks.datadog",
		"splunk":        "extensions.reporting_sinks.splunk",
		"otelcollector": "extensions.reporting_sinks.otel_collector",
	}
	trackingFeatureByKind = map[string]string{
		"http":           "endpoint.http",
		"kafka":          "endpoint.kafka",
		"rabbitmq":       "endpoint.rabbitmq",
		"azureeventhubs": "endpoint.azure_event_hubs",
		"pushdiffusion":  "endpoint.push_diffusion",
		"delegatestream": "endpoint.delegate_stream",
		"nats":           "endpoint.nats",
		"redisstreams":   "endpoint.redis_streams",
	}
	ciEnvironmentVariables = []string{
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"TF_BUILD",
		"BUILD_BUILDID",
		"JENKINS_URL",
		"TEAMCITY_VERSION",
		"CIRCLECI",
		"BUILDKITE",
		"TRAVIS",
		"APPVEYOR",
		"BITBUCKET_BUILD_NUMBER",
		"CODEBUILD_BUILD_ID",
		"DRONE",
		"SEMAPHORE",
		"HEROKU_TEST_RUN_ID",
	}
)

type runnerLease struct {
	stop func()
}

type cachedSigningKey struct {
	KeyID        string
	Algorithm    string
	Issuer       string
	Audience     string
	PublicKeyPEM string
	ExpiresAtUTC time.Time
}

type licensingValidationRequest struct {
	RunnerKey                 string   `json:"RunnerKey"`
	RequestedFeatures         []string `json:"RequestedFeatures"`
	SessionID                 string   `json:"SessionId"`
	TestSuite                 string   `json:"TestSuite,omitempty"`
	TestName                  string   `json:"TestName,omitempty"`
	NodeType                  string   `json:"NodeType,omitempty"`
	MachineName               string   `json:"MachineName,omitempty"`
	EnvironmentClassification int      `json:"EnvironmentClassification"`
	DeviceHash                string   `json:"DeviceHash"`
}

type licensingValidationResponse struct {
	IsValid                  bool   `json:"isValid"`
	DenialCode               string `json:"denialCode"`
	Message                  string `json:"message"`
	RunToken                 string `json:"runToken"`
	HeartbeatIntervalSeconds int    `json:"heartbeatIntervalSeconds"`
}

type licensingSigningKeyResponse struct {
	KeyID        string `json:"keyId"`
	Algorithm    string `json:"algorithm"`
	Issuer       string `json:"issuer"`
	Audience     string `json:"audience"`
	PublicKeyPEM string `json:"publicKeyPem"`
}

type jwtClaims struct {
	Issuer     string
	Audience   []string
	Expiration int64
	NotBefore  int64
	RunnerKey  string
	SessionID  string
	DeviceHash string
	Features   []string
	Policy     map[string]any
	RawPayload map[string]any
}

func acquireLicenseLease(runContext *contextState) (*runnerLease, error) {
	if runContext == nil || isLicenseEnforcementDisabledForTests() {
		return nil, nil
	}

	if runContext.NodeType == NodeTypeAgent {
		return nil, nil
	}

	if strings.TrimSpace(runContext.RunnerKey) == "" {
		return nil, errors.New("Runner key is required. Call WithRunnerKey(...) before Run().")
	}

	features := collectRequestedFeatures(*runContext)
	sessionID := strings.TrimSpace(runContext.SessionID)
	if sessionID == "" {
		sessionID = generateSessionID()
		runContext.SessionID = sessionID
	}

	baseURL := resolveLicensingAPIBaseURL()
	deviceHash := computeDeviceHash()
	environmentClassification := detectEnvironmentClassification(runContext.NodeType)
	timeout := licenseValidationTimeout(*runContext)
	requestPayload := licensingValidationRequest{
		RunnerKey:                 strings.TrimSpace(runContext.RunnerKey),
		RequestedFeatures:         features,
		SessionID:                 sessionID,
		TestSuite:                 strings.TrimSpace(runContext.TestSuite),
		TestName:                  strings.TrimSpace(runContext.TestName),
		NodeType:                  nodeTypeName(runContext.NodeType),
		MachineName:               hostName(),
		EnvironmentClassification: environmentClassification,
		DeviceHash:                deviceHash,
	}

	response, err := postLicensingValidation(baseURL, timeout, requestPayload)
	if err != nil {
		return nil, err
	}

	if !response.IsValid {
		details := strings.TrimSpace(response.Message)
		if details == "" {
			details = "No additional details provided."
		}
		return nil, fmt.Errorf(
			"Runner key validation denied for %q. Reason: %s. %s",
			requestPayload.RunnerKey,
			strings.TrimSpace(response.DenialCode),
			details,
		)
	}

	claims, err := validateRunToken(baseURL, timeout, response.RunToken, requestPayload.RunnerKey, sessionID, deviceHash, features)
	if err != nil {
		return nil, err
	}
	if err := enforceRuntimePolicyClaims(claims, *runContext); err != nil {
		return nil, err
	}

	lease := &runnerLease{
		stop: func() {},
	}
	heartbeatIntervalSeconds := response.HeartbeatIntervalSeconds
	if heartbeatIntervalSeconds <= 0 {
		heartbeatIntervalSeconds = 1
	}

	heartbeatContext, cancel := stdcontext.WithCancel(stdcontext.Background())
	ticker := time.NewTicker(time.Duration(heartbeatIntervalSeconds) * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatContext.Done():
				return
			case <-ticker.C:
				_ = postJSON(baseURL+"/api/v1/licenses/heartbeat", timeout, map[string]any{
					"RunToken":                  response.RunToken,
					"SessionId":                 sessionID,
					"DeviceHash":                deviceHash,
					"MachineName":               requestPayload.MachineName,
					"EnvironmentClassification": environmentClassification,
				}, nil)
			}
		}
	}()

	lease.stop = func() {
		cancel()
		_ = postJSON(baseURL+"/api/v1/licenses/stop", timeout, map[string]any{
			"RunToken":   response.RunToken,
			"SessionId":  sessionID,
			"DeviceHash": deviceHash,
		}, nil)
	}

	return lease, nil
}

func licenseValidationTimeout(context contextState) time.Duration {
	seconds := context.LicenseValidationTimeoutSeconds
	if seconds <= 0 {
		seconds = defaultLicenseValidationTimeoutSeconds
	}
	return time.Duration(seconds * float64(time.Second))
}

func postLicensingValidation(baseURL string, timeout time.Duration, payload licensingValidationRequest) (licensingValidationResponse, error) {
	var response licensingValidationResponse
	raw := map[string]any{}
	if err := postJSON(buildURL(baseURL, "/api/v1/licenses/validate"), timeout, payload, &raw); err != nil {
		return licensingValidationResponse{}, err
	}

	response.IsValid = readBool(raw, "isValid", "IsValid")
	response.DenialCode = readString(raw, "denialCode", "DenialCode")
	response.Message = readString(raw, "message", "Message")
	response.RunToken = readString(raw, "runToken", "RunToken", "SignedRunToken", "Token")
	response.HeartbeatIntervalSeconds = readInt(raw, "heartbeatIntervalSeconds", "HeartbeatIntervalSeconds")
	return response, nil
}

func validateRunToken(baseURL string, timeout time.Duration, token string, runnerKey string, sessionID string, deviceHash string, requestedFeatures []string) (jwtClaims, error) {
	if strings.TrimSpace(token) == "" {
		return jwtClaims{}, errors.New("Runner key validation failed: run token is missing.")
	}

	header, payload, signingInput, signature, err := parseJWT(token)
	if err != nil {
		return jwtClaims{}, err
	}

	algorithm := strings.ToUpper(readString(header, "alg", "Alg"))
	if algorithm == "" {
		algorithm = "RS256"
	}
	if algorithm != "RS256" {
		return jwtClaims{}, errors.New("Runner key validation failed: unsupported run token algorithm.")
	}

	keyID := readString(header, "kid", "Kid")
	if keyID == "" {
		keyID = "default"
	}
	signingKey, err := getSigningKey(baseURL, timeout, keyID)
	if err != nil {
		return jwtClaims{}, err
	}
	if err := verifyRS256Signature(signingInput, signature, signingKey.PublicKeyPEM); err != nil {
		return jwtClaims{}, err
	}

	claims := buildJWTClaims(payload)
	if err := enforceJWTLifetime(claims); err != nil {
		return jwtClaims{}, err
	}
	if strings.TrimSpace(claims.Issuer) != strings.TrimSpace(signingKey.Issuer) {
		return jwtClaims{}, errors.New("Runner key validation failed: run token issuer does not match.")
	}
	if len(claims.Audience) == 0 || !slices.Contains(claims.Audience, signingKey.Audience) {
		return jwtClaims{}, errors.New("Runner key validation failed: run token audience does not match.")
	}
	if claims.RunnerKey != runnerKey {
		return jwtClaims{}, errors.New("Runner key validation failed: run token runner key does not match.")
	}
	if claims.SessionID != sessionID {
		return jwtClaims{}, errors.New("Runner key validation failed: run token session id does not match.")
	}
	if claims.DeviceHash != deviceHash {
		return jwtClaims{}, errors.New("Runner key validation failed: run token device hash does not match.")
	}
	if err := enforceEntitlementClaims(claims, requestedFeatures); err != nil {
		return jwtClaims{}, err
	}
	return claims, nil
}

func buildJWTClaims(payload map[string]any) jwtClaims {
	audience := readStringSlice(payload, "aud", "Aud")
	if len(audience) == 0 {
		if value := readString(payload, "aud", "Aud"); value != "" {
			audience = []string{value}
		}
	}
	return jwtClaims{
		Issuer:     readString(payload, "iss", "Iss"),
		Audience:   audience,
		Expiration: readInt64(payload, "exp", "Exp"),
		NotBefore:  readInt64(payload, "nbf", "Nbf"),
		RunnerKey:  readString(payload, "runner_key", "RunnerKey"),
		SessionID:  readString(payload, "session_id", "SessionId"),
		DeviceHash: readString(payload, "device_hash", "DeviceHash"),
		Features:   collectClaimStrings(payload, "entitlements", "Entitlements", "features", "Features", "feature", "Feature"),
		Policy:     resolveRuntimePolicy(payload),
		RawPayload: payload,
	}
}

func enforceJWTLifetime(claims jwtClaims) error {
	const skew = 30 * time.Second
	now := time.Now().UTC().Unix()
	if claims.Expiration <= 0 {
		return errors.New("Runner key validation failed: run token expiration is missing.")
	}
	if now > claims.Expiration+int64(skew.Seconds()) {
		return errors.New("Runner key validation failed: run token is expired.")
	}
	if claims.NotBefore > 0 && now+int64(skew.Seconds()) < claims.NotBefore {
		return errors.New("Runner key validation failed: run token is not yet valid.")
	}
	return nil
}

func enforceEntitlementClaims(claims jwtClaims, requestedFeatures []string) error {
	normalized := make(map[string]struct{}, len(claims.Features))
	for _, feature := range claims.Features {
		value := strings.ToLower(strings.TrimSpace(feature))
		if value == "" {
			continue
		}
		normalized[value] = struct{}{}
	}
	for _, feature := range requestedFeatures {
		key := strings.ToLower(strings.TrimSpace(feature))
		if key == "" {
			continue
		}
		if _, ok := normalized[key]; !ok {
			return fmt.Errorf("Runner key validation denied. DenialCode=missing_entitlement, Message=Missing entitlement for %s.", feature)
		}
	}
	return nil
}

func enforceRuntimePolicyClaims(claims jwtClaims, context contextState) error {
	policy := claims.Policy
	if len(policy) == 0 {
		return nil
	}

	scenarioCount := len(context.scenarios)
	if maxScenarios := readInt(policy, "maxScenarios", "MaxScenarios", "maxScenarioCount", "MaxScenarioCount"); maxScenarios > 0 && scenarioCount > maxScenarios {
		return fmt.Errorf("License policy denied run: scenario count %d exceeds max %d.", scenarioCount, maxScenarios)
	}
	agentsCount := maxInt(context.AgentsCount, 1)
	if maxAgents := readInt(policy, "maxAgentsCount", "MaxAgentsCount"); maxAgents > 0 && agentsCount > maxAgents {
		return fmt.Errorf("License policy denied run: AgentsCount %d exceeds max %d.", agentsCount, maxAgents)
	}
	if maxPlugins := readInt(policy, "maxCustomWorkerPlugins", "MaxCustomWorkerPlugins"); maxPlugins > 0 && countCustomWorkerPlugins(context.WorkerPlugins) > maxPlugins {
		return fmt.Errorf("License policy denied run: custom worker plugin count %d exceeds max %d.", countCustomWorkerPlugins(context.WorkerPlugins), maxPlugins)
	}
	if maxSinks := readInt(policy, "maxReportingSinks", "MaxReportingSinks"); maxSinks > 0 && len(context.ReportingSinks) > maxSinks {
		return fmt.Errorf("License policy denied run: reporting sink count %d exceeds max %d.", len(context.ReportingSinks), maxSinks)
	}
	if maxTargets := readInt(policy, "maxTargetScenarioCount", "MaxTargetScenarioCount"); maxTargets > 0 && countTargetScenarios(context) > maxTargets {
		return fmt.Errorf("License policy denied run: targeted scenario count %d exceeds max %d.", countTargetScenarios(context), maxTargets)
	}
	if maxClusterTimeout := readInt(policy, "maxClusterCommandTimeoutSeconds", "MaxClusterCommandTimeoutSeconds"); maxClusterTimeout > 0 && context.ClusterCommandTimeoutSeconds > float64(maxClusterTimeout) {
		return fmt.Errorf("License policy denied run: ClusterCommandTimeout %.3fs exceeds max %ds.", context.ClusterCommandTimeoutSeconds, maxClusterTimeout)
	}
	if maxScenarioTimeout := readInt(policy, "maxScenarioCompletionTimeoutSeconds", "MaxScenarioCompletionTimeoutSeconds"); maxScenarioTimeout > 0 && context.ScenarioCompletionTimeoutSeconds > float64(maxScenarioTimeout) {
		return fmt.Errorf("License policy denied run: ScenarioCompletionTimeout %.3fs exceeds max %ds.", context.ScenarioCompletionTimeoutSeconds, maxScenarioTimeout)
	}
	return nil
}

func collectRequestedFeatures(context contextState) []string {
	features := map[string]struct{}{
		"core.runtime":               {},
		"reporting.formats.standard": {},
	}

	if context.LocalDevClusterEnabled {
		features["cluster.local_dev"] = struct{}{}
	}
	if strings.TrimSpace(context.NatsServerURL) != "" {
		features["cluster.distributed.nats"] = struct{}{}
	}
	if len(context.AgentTargetScenarios) > 0 || len(context.CoordinatorTargetScenarios) > 0 || context.ClusterCommandTimeoutSeconds > 120 {
		features["cluster.targeting_and_tuning.advanced"] = struct{}{}
	}
	if countCustomWorkerPlugins(context.WorkerPlugins) > 0 {
		features["extensions.worker_plugins.custom"] = struct{}{}
	}
	for _, sink := range context.ReportingSinks {
		if feature := sinkLicenseFeature(sink); feature != "" {
			features[feature] = struct{}{}
			continue
		}
		features["extensions.reporting_sinks.custom"] = struct{}{}
	}
	for _, scenario := range context.scenarios {
		if len(scenario.Thresholds) > 0 {
			features["policy.report_finalizer_and_threshold_controls"] = struct{}{}
		}
		addTrackingFeatures(features, scenario.Tracking)
	}

	keys := make([]string, 0, len(features))
	for feature := range features {
		keys = append(keys, feature)
	}
	slices.Sort(keys)
	return keys
}

func addTrackingFeatures(features map[string]struct{}, tracking *TrackingConfigurationSpec) {
	if tracking == nil {
		return
	}
	if feature := endpointLicenseFeature(tracking.Source); feature != "" {
		features[feature] = struct{}{}
	}
	if feature := endpointLicenseFeature(tracking.Destination); feature != "" {
		features[feature] = struct{}{}
	} else if tracking.Source != nil {
		features["crossplatform.source_only_reporting"] = struct{}{}
	}
	if tracking.Source != nil || tracking.Destination != nil {
		features["crossplatform.tracking.selectors"] = struct{}{}
		features["reporting.correlation.grouped"] = struct{}{}
	}
	if tracking.CorrelationStore != nil && strings.EqualFold(strings.TrimSpace(tracking.CorrelationStore.Kind), "Redis") {
		features["correlation.store.redis"] = struct{}{}
	}
}

func endpointLicenseFeature(endpoint *EndpointSpec) string {
	if endpoint == nil {
		return ""
	}
	key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(endpoint.Kind), "_", ""))
	key = strings.ReplaceAll(key, "-", "")
	return trackingFeatureByKind[key]
}

func sinkLicenseFeature(sink any) string {
	spec, ok := coerceReportingSinkSpec(sink)
	if !ok {
		return ""
	}
	key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(spec.Kind), "_", ""))
	key = strings.ReplaceAll(key, "-", "")
	if feature, ok := sinkFeatureByKind[key]; ok {
		return feature
	}
	if spec.InfluxDB != nil {
		return sinkFeatureByKind["influxdb"]
	}
	if spec.TimescaleDB != nil {
		return sinkFeatureByKind["timescaledb"]
	}
	if spec.GrafanaLoki != nil {
		return sinkFeatureByKind["grafanaloki"]
	}
	if spec.Datadog != nil {
		return sinkFeatureByKind["datadog"]
	}
	if spec.Splunk != nil {
		return sinkFeatureByKind["splunk"]
	}
	if spec.OTELCollector != nil {
		return sinkFeatureByKind["otelcollector"]
	}
	return ""
}

func coerceReportingSinkSpec(sink any) (reportingSinkSpec, bool) {
	switch typed := sink.(type) {
	case reportingSinkSpec:
		return typed, true
	case *reportingSinkSpec:
		if typed != nil {
			return *typed, true
		}
	case InfluxDbReportingSink:
		return reportingSinkSpec{Kind: "InfluxDb", InfluxDB: &typed.Options}, true
	case *InfluxDbReportingSink:
		if typed != nil {
			return reportingSinkSpec{Kind: "InfluxDb", InfluxDB: &typed.Options}, true
		}
	case TimescaleDbReportingSink:
		return reportingSinkSpec{Kind: "TimescaleDb", TimescaleDB: &typed.Options}, true
	case *TimescaleDbReportingSink:
		if typed != nil {
			return reportingSinkSpec{Kind: "TimescaleDb", TimescaleDB: &typed.Options}, true
		}
	case GrafanaLokiReportingSink:
		return reportingSinkSpec{Kind: "GrafanaLoki", GrafanaLoki: &typed.Options}, true
	case *GrafanaLokiReportingSink:
		if typed != nil {
			return reportingSinkSpec{Kind: "GrafanaLoki", GrafanaLoki: &typed.Options}, true
		}
	case DatadogReportingSink:
		return reportingSinkSpec{Kind: "Datadog", Datadog: &typed.Options}, true
	case *DatadogReportingSink:
		if typed != nil {
			return reportingSinkSpec{Kind: "Datadog", Datadog: &typed.Options}, true
		}
	case SplunkReportingSink:
		return reportingSinkSpec{Kind: "Splunk", Splunk: &typed.Options}, true
	case *SplunkReportingSink:
		if typed != nil {
			return reportingSinkSpec{Kind: "Splunk", Splunk: &typed.Options}, true
		}
	case OtelCollectorReportingSink:
		return reportingSinkSpec{Kind: "OtelCollector", OTELCollector: &typed.Options}, true
	case *OtelCollectorReportingSink:
		if typed != nil {
			return reportingSinkSpec{Kind: "OtelCollector", OTELCollector: &typed.Options}, true
		}
	}
	return reportingSinkSpec{}, false
}

func countCustomWorkerPlugins(plugins []LoadStrikeWorkerPlugin) int {
	count := 0
	for _, plugin := range plugins {
		name := strings.TrimSpace(workerPluginName(plugin))
		if _, ok := builtInWorkerPluginNames[strings.ToLower(name)]; ok {
			continue
		}
		if name != "" {
			count++
		}
	}
	return count
}

func workerPluginName(plugin any) string {
	switch typed := plugin.(type) {
	case nil:
		return ""
	case LoadStrikeWorkerPlugin:
		return typed.PluginName()
	case legacyLoadStrikeWorkerPlugin:
		return typed.PluginName()
	case workerPluginSpec:
		return typed.PluginName()
	case *workerPluginSpec:
		if typed != nil {
			return typed.PluginName()
		}
	case workerPlugin:
		return typed.Name()
	}
	return ""
}

func countTargetScenarios(context contextState) int {
	seen := map[string]struct{}{}
	for _, values := range [][]string{context.TargetScenarios, context.AgentTargetScenarios, context.CoordinatorTargetScenarios} {
		for _, value := range values {
			key := strings.ToLower(strings.TrimSpace(value))
			if key == "" {
				continue
			}
			seen[key] = struct{}{}
		}
	}
	return len(seen)
}

func resolveRuntimePolicy(payload map[string]any) map[string]any {
	policy := readMap(payload, "runtimePolicy", "RuntimePolicy", "policy", "Policy")
	setPositiveInt(policy, payload, "maxAgentsCount", "MaxAgentsCount", "policy.max_agents_count")
	setPositiveInt(policy, payload, "maxScenarios", "MaxScenarios", "maxScenarioCount", "MaxScenarioCount", "policy.max_scenario_count")
	setPositiveInt(policy, payload, "maxCustomWorkerPlugins", "MaxCustomWorkerPlugins", "policy.max_custom_worker_plugins")
	setPositiveInt(policy, payload, "maxReportingSinks", "MaxReportingSinks", "policy.max_reporting_sinks")
	setPositiveInt(policy, payload, "maxTargetScenarioCount", "MaxTargetScenarioCount", "policy.max_target_scenario_count")
	setPositiveInt(policy, payload, "maxClusterCommandTimeoutSeconds", "MaxClusterCommandTimeoutSeconds", "policy.max_cluster_command_timeout_seconds")
	setPositiveInt(policy, payload, "maxScenarioCompletionTimeoutSeconds", "MaxScenarioCompletionTimeoutSeconds", "policy.max_scenario_completion_timeout_seconds")
	return policy
}

func setPositiveInt(target map[string]any, secondary map[string]any, key string, keys ...string) {
	for _, candidate := range append([]string{key}, keys...) {
		value := readInt(secondary, candidate)
		if value > 0 {
			target[key] = value
			return
		}
		value = readInt(target, candidate)
		if value > 0 {
			target[key] = value
			return
		}
	}
}

func getSigningKey(baseURL string, timeout time.Duration, keyID string) (cachedSigningKey, error) {
	normalizedKeyID := firstNonBlank(strings.TrimSpace(keyID), "default")
	cacheKey := strings.ToLower(strings.TrimRight(baseURL, "/")) + "|" + normalizedKeyID
	if value, ok := licenseSigningKeyCache.Load(cacheKey); ok {
		cached := value.(cachedSigningKey)
		if time.Now().UTC().Before(cached.ExpiresAtUTC) {
			return cached, nil
		}
	}

	raw := map[string]any{}
	escapedKeyID := strings.ReplaceAll(url.QueryEscape(normalizedKeyID), "+", "%20")
	if err := getJSON(buildURL(baseURL, "/api/v1/licenses/signing-public-key?kid="+escapedKeyID), timeout, &raw); err != nil {
		return cachedSigningKey{}, err
	}
	signingKey := cachedSigningKey{
		KeyID:        firstNonBlank(readString(raw, "keyId", "KeyId"), normalizedKeyID),
		Algorithm:    firstNonBlank(readString(raw, "algorithm", "Algorithm"), "RS256"),
		Issuer:       readString(raw, "issuer", "Issuer"),
		Audience:     readString(raw, "audience", "Audience"),
		PublicKeyPEM: readString(raw, "publicKeyPem", "PublicKeyPem", "pem", "Pem"),
		ExpiresAtUTC: time.Now().UTC().Add(15 * time.Minute),
	}
	if strings.TrimSpace(signingKey.KeyID) != "" && !strings.EqualFold(strings.TrimSpace(signingKey.KeyID), normalizedKeyID) {
		return cachedSigningKey{}, fmt.Errorf("Runner key validation failed: signing key id mismatch. Requested %q but received %q.", normalizedKeyID, signingKey.KeyID)
	}
	if strings.TrimSpace(signingKey.PublicKeyPEM) == "" {
		return cachedSigningKey{}, errors.New("Runner key validation failed: signing key response is empty.")
	}
	licenseSigningKeyCache.Store(cacheKey, signingKey)
	return signingKey, nil
}

func verifyRS256Signature(signingInput string, signature []byte, publicKeyPEM string) error {
	block, _ := pem.Decode([]byte(strings.TrimSpace(publicKeyPEM)))
	if block == nil {
		return errors.New("Runner key validation failed: signing key response is empty.")
	}

	publicKeyValue, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	publicKey, ok := publicKeyValue.(*rsa.PublicKey)
	if !ok {
		return errors.New("Runner key validation failed: signing key format is invalid.")
	}

	sum := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, sum[:], signature); err != nil {
		return errors.New("Runner key validation failed: invalid run token signature.")
	}
	return nil
}

func parseJWT(token string) (map[string]any, map[string]any, string, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, nil, "", nil, errors.New("Runner key validation failed: run token is malformed.")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, "", nil, err
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, "", nil, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, nil, "", nil, err
	}

	var header map[string]any
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, nil, "", nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, nil, "", nil, err
	}
	return header, payload, parts[0] + "." + parts[1], signature, nil
}

func postJSON(url string, timeout time.Duration, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	return doJSONRequest(request, timeout, target)
}

func getJSON(url string, timeout time.Duration, target any) error {
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: timeout}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("license signing key request failed with status %d", response.StatusCode)
	}

	if target == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil && !errors.Is(err, stdcontext.Canceled) {
		return err
	}
	return nil
}

func doJSONRequest(request *http.Request, timeout time.Duration, target any) error {
	client := &http.Client{Timeout: timeout}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if target == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil && !errors.Is(err, stdcontext.Canceled) {
		return err
	}
	return nil
}

func resolveLicensingAPIBaseURL() string {
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv(licenseValidationBaseURLEnv)), "/"); value != "" {
		return value
	}
	return defaultLicenseValidationBaseURL
}

func buildURL(baseURL string, relative string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(relative, "/")
}

func isLicenseEnforcementDisabledForTests() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(licenseEnforcementDisabledForTestsEnv)))
	return value == "1" || value == "true" || value == "yes"
}

func generateSessionID() string {
	return strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), "-", "")
}

func nodeTypeName(nodeType NodeType) string {
	switch nodeType {
	case NodeTypeCoordinator:
		return "Coordinator"
	case NodeTypeAgent:
		return "Agent"
	default:
		return "SingleNode"
	}
}

func detectEnvironmentClassification(nodeType NodeType) int {
	if isCIHost() {
		return 1
	}
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" || os.Getenv("KUBERNETES_PORT") != "" {
		return 2
	}
	if isLocalHost() {
		return 0
	}
	if nodeType == NodeTypeCoordinator {
		return 3
	}
	return 1
}

func isCIHost() bool {
	if isTruthyEnv("CI") {
		return true
	}
	for _, name := range ciEnvironmentVariables {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			return true
		}
	}
	return false
}

func isLocalHost() bool {
	if runtime.GOOS == "windows" {
		return !strings.Contains(strings.ToLower(readWindowsProductName()), "server")
	}
	if runtime.GOOS == "darwin" {
		return true
	}
	if runtime.GOOS == "linux" {
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	}
	return false
}

func computeDeviceHash() string {
	parts := []string{
		strings.TrimSpace(runtime.GOOS + "-" + runtime.GOARCH),
		hostName(),
	}
	switch runtime.GOOS {
	case "windows":
		if value := readWindowsMachineGUID(); value != "" {
			parts = append(parts, value)
		}
	case "linux":
		if value := readFirstExistingFile("/etc/machine-id", "/var/lib/dbus/machine-id"); value != "" {
			parts = append(parts, value)
		}
	case "darwin":
		if value := runCommandOutput("ioreg", "-rd1", "-c", "IOPlatformExpertDevice"); value != "" {
			parts = append(parts, value)
		}
	}
	sum := sha256.Sum256([]byte(strings.ToLower(strings.Join(parts, "|"))))
	return fmt.Sprintf("%x", sum[:])
}

func hostName() string {
	value, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return value
}

func readWindowsMachineGUID() string {
	return parseRegistryQuery("HKLM\\SOFTWARE\\Microsoft\\Cryptography", "MachineGuid")
}

func readWindowsProductName() string {
	return parseRegistryQuery("HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion", "ProductName")
}

func parseRegistryQuery(path string, valueName string) string {
	output := runCommandOutput("reg", "query", path, "/v", valueName)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(strings.ToLower(line), strings.ToLower(valueName)) {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

func runCommandOutput(name string, args ...string) string {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func readFirstExistingFile(paths ...string) string {
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(body))
		}
	}
	return ""
}

func isTruthyEnv(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return value != "" && value != "0" && value != "false" && value != "no"
}

func collectClaimStrings(payload map[string]any, keys ...string) []string {
	values := []string{}
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case []any:
			for _, entry := range typed {
				if text := strings.TrimSpace(fmt.Sprint(entry)); text != "" {
					values = append(values, text)
				}
			}
		default:
			if text := strings.TrimSpace(fmt.Sprint(typed)); text != "" {
				values = append(values, text)
			}
		}
	}
	return values
}

func readMap(payload map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		if record, ok := value.(map[string]any); ok {
			return mapsClone(record)
		}
	}
	return map[string]any{}
}

func readStringSlice(payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case []any:
			result := make([]string, 0, len(typed))
			for _, entry := range typed {
				result = append(result, fmt.Sprint(entry))
			}
			return result
		case []string:
			return append([]string(nil), typed...)
		}
	}
	return nil
}

func readString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if ok && value != nil {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func readBool(payload map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		default:
			text := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
			return text == "1" || text == "true" || text == "yes"
		}
	}
	return false
}

func readInt(payload map[string]any, keys ...string) int {
	return int(readInt64(payload, keys...))
}

func readInt64(payload map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int64(typed)
		case int:
			return int64(typed)
		case int64:
			return typed
		case json.Number:
			parsed, _ := typed.Int64()
			return parsed
		default:
			parsed, err := time.ParseDuration(strings.TrimSpace(fmt.Sprint(value)))
			if err == nil {
				return int64(parsed.Seconds())
			}
			var asFloat float64
			if _, err := fmt.Sscan(fmt.Sprint(value), &asFloat); err == nil {
				return int64(asFloat)
			}
		}
	}
	return 0
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mapsClone(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
