package loadstrike

import (
	stdcontext "context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type trackingFieldSelector struct {
	Kind string
	Path string
}

type gatheredRow struct {
	Total    int `json:"total"`
	Matched  int `json:"matched"`
	TimedOut int `json:"timedOut"`
}

type sourceProduceResult struct {
	TrackingID        string `json:"trackingId"`
	EventID           string `json:"eventId"`
	SourceTimestampMS int64  `json:"sourceTimestampUtc"`
	DuplicateSource   bool   `json:"duplicateSource"`
}

type destinationConsumeResult struct {
	TrackingID             string `json:"trackingId"`
	EventID                string `json:"eventId"`
	SourceTimestampUTCMS   int64  `json:"sourceTimestampUtc"`
	DestinationTimestampMS int64  `json:"destinationTimestampUtc"`
	GatherKey              string `json:"gatherKey"`
	Matched                bool   `json:"matched"`
	DuplicateDestination   bool   `json:"duplicateDestination"`
	LatencyMS              int64  `json:"latencyMs"`
	Message                string `json:"message"`
}

type correlationRuntimeStats struct {
	ProducedCount             int                    `json:"producedCount"`
	ConsumedCount             int                    `json:"consumedCount"`
	MatchedCount              int                    `json:"matchedCount"`
	TimeoutCount              int                    `json:"timeoutCount"`
	DuplicateSourceCount      int                    `json:"duplicateSourceCount"`
	DuplicateDestinationCount int                    `json:"duplicateDestinationCount"`
	InflightCount             int                    `json:"inflightCount"`
	AverageLatencyMS          float64                `json:"averageLatencyMs"`
	Gathered                  map[string]gatheredRow `json:"gathered"`
}

type crossPlatformTrackingRuntimeOptions struct {
	SourceTrackingField             string
	DestinationTrackingField        string
	DestinationGatherByField        string
	TrackingFieldValueCaseSensitive bool
	GatherByFieldValueCaseSensitive bool
	CorrelationTimeoutMS            int64
	TimeoutCountsAsFailure          bool
	Store                           correlationStore
}

type crossPlatformTrackingRuntime struct {
	sourceSelector                  trackingFieldSelector
	destinationSelector             trackingFieldSelector
	gatherSelector                  *trackingFieldSelector
	trackingFieldValueCaseSensitive bool
	gatherByFieldValueCaseSensitive bool
	timeoutMS                       int64
	timeoutCountsAsFailure          bool
	store                           correlationStore
	trackingIDDisplayByEventID      map[string]string
	gatherByDisplayByComparisonKey  map[string]string
	producedCount                   int
	consumedCount                   int
	matchedCount                    int
	timeoutCount                    int
	duplicateSourceCount            int
	duplicateDestinationCount       int
	totalLatencyMS                  int64
	gathered                        map[string]gatheredRow
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

func mustParseTrackingFieldSelector(value string) trackingFieldSelector {
	selector, err := parseTrackingFieldSelector(value)
	if err != nil {
		panic(err)
	}
	return selector
}

func (s trackingFieldSelector) Extract(payload trackingPayload) string {
	switch s.Kind {
	case "header":
		for key, value := range payload.Headers {
			if strings.EqualFold(key, s.Path) {
				return strings.TrimSpace(value)
			}
		}
		return ""
	case "json":
		body := normalizeJSONBody(payload.Body)
		record, ok := body.(map[string]any)
		if !ok {
			return ""
		}
		path := strings.TrimPrefix(s.Path, "$.")
		current := any(record)
		for _, segment := range strings.Split(path, ".") {
			segment = strings.TrimSpace(segment)
			if segment == "" {
				continue
			}
			next, ok := current.(map[string]any)
			if !ok {
				return ""
			}
			value, exists := next[segment]
			if !exists || value == nil {
				return ""
			}
			current = value
		}
		return strings.TrimSpace(asString(current))
	default:
		return ""
	}
}

func (s trackingFieldSelector) Set(payload *trackingPayload, value string) {
	if payload == nil {
		return
	}
	switch s.Kind {
	case "header":
		if payload.Headers == nil {
			payload.Headers = map[string]string{}
		}
		payload.Headers[s.Path] = value
	case "json":
		path := strings.TrimPrefix(s.Path, "$.")
		record, ok := normalizeJSONBody(payload.Body).(map[string]any)
		if !ok || record == nil {
			record = map[string]any{}
		}
		current := record
		segments := []string{}
		for _, segment := range strings.Split(path, ".") {
			segment = strings.TrimSpace(segment)
			if segment != "" {
				segments = append(segments, segment)
			}
		}
		for index, segment := range segments {
			if index == len(segments)-1 {
				current[segment] = value
				break
			}
			next, ok := current[segment].(map[string]any)
			if !ok || next == nil {
				next = map[string]any{}
				current[segment] = next
			}
			current = next
		}
		payload.Body = record
	}
}

func newCrossPlatformTrackingRuntime(options crossPlatformTrackingRuntimeOptions) *crossPlatformTrackingRuntime {
	sourceSelector := mustParseTrackingFieldSelector(options.SourceTrackingField)
	destinationSelector := sourceSelector
	if strings.TrimSpace(options.DestinationTrackingField) != "" {
		destinationSelector = mustParseTrackingFieldSelector(options.DestinationTrackingField)
	}

	var gatherSelector *trackingFieldSelector
	if strings.TrimSpace(options.DestinationGatherByField) != "" {
		selector := mustParseTrackingFieldSelector(options.DestinationGatherByField)
		gatherSelector = &selector
	}

	timeout := options.CorrelationTimeoutMS
	if timeout <= 0 {
		timeout = 30000
	}

	store := options.Store
	if store == nil {
		store = newInMemoryCorrelationStore()
	}

	return &crossPlatformTrackingRuntime{
		sourceSelector:                  sourceSelector,
		destinationSelector:             destinationSelector,
		gatherSelector:                  gatherSelector,
		trackingFieldValueCaseSensitive: options.TrackingFieldValueCaseSensitive,
		gatherByFieldValueCaseSensitive: options.GatherByFieldValueCaseSensitive,
		timeoutMS:                       timeout,
		timeoutCountsAsFailure:          options.TimeoutCountsAsFailure,
		store:                           store,
		trackingIDDisplayByEventID:      map[string]string{},
		gatherByDisplayByComparisonKey:  map[string]string{},
		gathered:                        map[string]gatheredRow{},
	}
}

func (r *crossPlatformTrackingRuntime) OnSourceProduced(payload trackingPayload, nowMS int64) (sourceProduceResult, bool, error) {
	r.producedCount++
	trackingID := r.sourceSelector.Extract(payload)
	if trackingID == "" {
		return sourceProduceResult{}, false, nil
	}
	comparisonTrackingID := normalizeComparisonValue(trackingID, r.trackingFieldValueCaseSensitive)
	sourceOccurrences, err := r.store.IncrementSourceOccurrences(stdcontext.Background(), comparisonTrackingID)
	if err != nil {
		return sourceProduceResult{}, false, err
	}
	if sourceOccurrences > 1 {
		r.duplicateSourceCount++
	}
	entry, err := r.store.RegisterSource(stdcontext.Background(), comparisonTrackingID, nowMS)
	if err != nil {
		return sourceProduceResult{}, false, err
	}
	entry.ProducedUTCMS = nowMS
	entry.ExpiresUTCMS = nowMS + r.timeoutMS
	entry.Payload = payload
	if entry.EventID != "" {
		r.trackingIDDisplayByEventID[entry.EventID] = trackingID
	}
	if err := r.store.Set(stdcontext.Background(), *entry); err != nil {
		return sourceProduceResult{}, false, err
	}
	return sourceProduceResult{
		TrackingID:        trackingID,
		EventID:           entry.EventID,
		SourceTimestampMS: entry.SourceTimestampUTCMS,
		DuplicateSource:   sourceOccurrences > 1,
	}, true, nil
}

func (r *crossPlatformTrackingRuntime) OnDestinationConsumed(payload trackingPayload, nowMS int64) (destinationConsumeResult, error) {
	return r.onDestinationConsumed(payload, nowMS, true)
}

func (r *crossPlatformTrackingRuntime) RetryDestinationMatch(payload trackingPayload, nowMS int64) (destinationConsumeResult, error) {
	return r.onDestinationConsumed(payload, nowMS, false)
}

func (r *crossPlatformTrackingRuntime) onDestinationConsumed(payload trackingPayload, nowMS int64, recordConsumption bool) (destinationConsumeResult, error) {
	result := destinationConsumeResult{
		GatherKey: "__ungrouped__",
	}

	if recordConsumption {
		r.consumedCount++
	}
	trackingID := r.destinationSelector.Extract(payload)
	if trackingID == "" {
		result.Message = "Destination payload did not include a tracking id."
		return result, nil
	}

	result.TrackingID = trackingID
	comparisonTrackingID := normalizeComparisonValue(trackingID, r.trackingFieldValueCaseSensitive)
	if recordConsumption {
		destinationOccurrences, err := r.store.IncrementDestinationOccurrences(stdcontext.Background(), comparisonTrackingID)
		if err != nil {
			return result, err
		}
		if destinationOccurrences > 1 {
			r.duplicateDestinationCount++
			result.DuplicateDestination = true
		}
	}

	source, err := r.store.TryMatchDestination(stdcontext.Background(), comparisonTrackingID, nowMS)
	if err != nil {
		return result, err
	}
	if source == nil {
		result.Message = "Destination tracking id had no source match."
		return result, nil
	}

	r.matchedCount++
	latencyMS := nowMS - source.ProducedUTCMS
	if latencyMS < 0 {
		latencyMS = 0
	}
	r.totalLatencyMS += latencyMS

	displayTrackingID := r.resolveTrackingIDDisplay(source.EventID, source.TrackingID, true)
	gatherKey := r.resolveGatherKey(payload)
	result.EventID = source.EventID
	result.SourceTimestampUTCMS = source.SourceTimestampUTCMS
	result.DestinationTimestampMS = nowMS
	result.TrackingID = displayTrackingID
	result.GatherKey = gatherKey
	result.Matched = true
	result.LatencyMS = latencyMS

	row := r.gathered[gatherKey]
	row.Total++
	row.Matched++
	r.gathered[gatherKey] = row
	return result, nil
}

func (r *crossPlatformTrackingRuntime) SweepTimeoutEntries(nowMS int64, batchSize int) ([]correlationEntry, error) {
	if batchSize <= 0 {
		batchSize = 200
	}
	expiredEntries, err := r.store.CollectExpired(stdcontext.Background(), nowMS, batchSize)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	result := make([]correlationEntry, 0, len(expiredEntries))
	for _, entry := range expiredEntries {
		expired, err := r.store.TryExpire(stdcontext.Background(), entry.EventID)
		if err != nil {
			return nil, err
		}
		if !expired {
			continue
		}
		key := correlationEntryDedupeKey(entry)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, entry)
	}

	for index := range result {
		entry := &result[index]
		displayTrackingID := r.resolveTrackingIDDisplay(entry.EventID, entry.TrackingID, true)
		entry.TrackingID = displayTrackingID
		r.timeoutCount++
		gatherKey := r.resolveGatherKey(entry.Payload)
		row := r.gathered[gatherKey]
		row.Total++
		row.TimedOut++
		r.gathered[gatherKey] = row
	}

	return result, nil
}

func (r *crossPlatformTrackingRuntime) GetStats() (correlationRuntimeStats, error) {
	inflight, err := r.store.All(stdcontext.Background())
	if err != nil {
		return correlationRuntimeStats{}, err
	}
	gathered := map[string]gatheredRow{}
	for key, value := range r.gathered {
		gathered[key] = value
	}
	averageLatency := 0.0
	if r.matchedCount > 0 {
		averageLatency = float64(r.totalLatencyMS) / float64(r.matchedCount)
	}
	return correlationRuntimeStats{
		ProducedCount:             r.producedCount,
		ConsumedCount:             r.consumedCount,
		MatchedCount:              r.matchedCount,
		TimeoutCount:              r.timeoutCount,
		DuplicateSourceCount:      r.duplicateSourceCount,
		DuplicateDestinationCount: r.duplicateDestinationCount,
		InflightCount:             len(inflight),
		AverageLatencyMS:          averageLatency,
		Gathered:                  gathered,
	}, nil
}

func (r *crossPlatformTrackingRuntime) resolveGatherKey(payload trackingPayload) string {
	if r.gatherSelector == nil {
		return "__ungrouped__"
	}
	gatherKey := r.gatherSelector.Extract(payload)
	if gatherKey == "" {
		return "__unknown__"
	}
	if r.gatherByFieldValueCaseSensitive {
		return gatherKey
	}
	comparisonKey := normalizeComparisonValue(gatherKey, false)
	if existing, ok := r.gatherByDisplayByComparisonKey[comparisonKey]; ok {
		return existing
	}
	r.gatherByDisplayByComparisonKey[comparisonKey] = gatherKey
	return gatherKey
}

func (r *crossPlatformTrackingRuntime) resolveTrackingIDDisplay(eventID string, fallbackTrackingID string, remove bool) string {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return fallbackTrackingID
	}
	if remove {
		if displayTrackingID, ok := r.trackingIDDisplayByEventID[eventID]; ok {
			delete(r.trackingIDDisplayByEventID, eventID)
			return displayTrackingID
		}
		return fallbackTrackingID
	}
	if displayTrackingID, ok := r.trackingIDDisplayByEventID[eventID]; ok {
		return displayTrackingID
	}
	return fallbackTrackingID
}

func normalizeComparisonValue(value string, caseSensitive bool) string {
	value = strings.TrimSpace(value)
	if caseSensitive {
		return value
	}
	return strings.ToLower(value)
}

func correlationEntryDedupeKey(entry correlationEntry) string {
	if strings.TrimSpace(entry.EventID) != "" {
		return strings.TrimSpace(entry.EventID)
	}
	return fmt.Sprintf("%s:%d", entry.TrackingID, entry.SourceTimestampUTCMS)
}

func generateTrackingID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return generateSessionID()
	}
	return hex.EncodeToString(buffer)
}

func normalizeJSONBody(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case map[string]any:
		return typed
	case []byte:
		var decoded any
		if json.Unmarshal(typed, &decoded) == nil {
			return decoded
		}
		return string(typed)
	case string:
		var decoded any
		if json.Unmarshal([]byte(typed), &decoded) == nil {
			return decoded
		}
		return typed
	default:
		body, err := json.Marshal(typed)
		if err != nil {
			return typed
		}
		var decoded any
		if err := json.Unmarshal(body, &decoded); err != nil {
			return typed
		}
		return decoded
	}
}
