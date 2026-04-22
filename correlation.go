package loadstrike

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type trackingPayload struct {
	Headers            map[string]string `json:"headers,omitempty"`
	Body               any               `json:"body,omitempty"`
	ProducedUTC        string            `json:"producedUtc,omitempty"`
	ConsumedUTC        string            `json:"consumedUtc,omitempty"`
	ContentType        string            `json:"contentType,omitempty"`
	MessagePayloadType string            `json:"messagePayloadType,omitempty"`
}

type correlationEntry struct {
	TrackingID             string          `json:"trackingId"`
	EventID                string          `json:"eventId"`
	SourceTimestampUTCMS   int64           `json:"sourceTimestampUtc,omitempty"`
	DestinationTimestampMS int64           `json:"destinationTimestampUtc,omitempty"`
	ProducedUTCMS          int64           `json:"producedUtc"`
	ExpiresUTCMS           int64           `json:"expiresUtc,omitempty"`
	Payload                trackingPayload `json:"payload"`
}

type correlationStore interface {
	Set(context.Context, correlationEntry) error
	Get(context.Context, string) (*correlationEntry, error)
	Delete(context.Context, string) error
	All(context.Context) ([]correlationEntry, error)
	CollectExpired(context.Context, int64, int) ([]correlationEntry, error)
	IncrementSourceOccurrences(context.Context, string) (int, error)
	IncrementDestinationOccurrences(context.Context, string) (int, error)
	RegisterSource(context.Context, string, int64) (*correlationEntry, error)
	TryMatchDestination(context.Context, string, int64) (*correlationEntry, error)
	TryExpire(context.Context, string) (bool, error)
	Close() error
}

type inMemoryCorrelationStore struct {
	pendingByTrackingID    map[string][]correlationEntry
	pendingByEventID       map[string]correlationEntry
	sourceOccurrences      map[string]int
	destinationOccurrences map[string]int
}

func newInMemoryCorrelationStore() *inMemoryCorrelationStore {
	return &inMemoryCorrelationStore{
		pendingByTrackingID:    map[string][]correlationEntry{},
		pendingByEventID:       map[string]correlationEntry{},
		sourceOccurrences:      map[string]int{},
		destinationOccurrences: map[string]int{},
	}
}

func (s *inMemoryCorrelationStore) Set(_ context.Context, entry correlationEntry) error {
	queue := append([]correlationEntry(nil), s.pendingByTrackingID[entry.TrackingID]...)
	queue = append(queue, entry)
	s.pendingByTrackingID[entry.TrackingID] = queue
	s.pendingByEventID[entry.EventID] = entry
	return nil
}

func (s *inMemoryCorrelationStore) Get(_ context.Context, trackingID string) (*correlationEntry, error) {
	queue := s.pendingByTrackingID[trackingID]
	if len(queue) == 0 {
		return nil, nil
	}
	entry := queue[0]
	return &entry, nil
}

func (s *inMemoryCorrelationStore) Delete(_ context.Context, trackingID string) error {
	queue := s.pendingByTrackingID[trackingID]
	if len(queue) == 0 {
		return nil
	}
	removed := queue[0]
	delete(s.pendingByEventID, removed.EventID)
	if len(queue) == 1 {
		delete(s.pendingByTrackingID, trackingID)
		return nil
	}
	s.pendingByTrackingID[trackingID] = append([]correlationEntry(nil), queue[1:]...)
	return nil
}

func (s *inMemoryCorrelationStore) All(_ context.Context) ([]correlationEntry, error) {
	result := []correlationEntry{}
	for _, queue := range s.pendingByTrackingID {
		result = append(result, queue...)
	}
	return result, nil
}

func (s *inMemoryCorrelationStore) CollectExpired(_ context.Context, nowMS int64, maxCount int) ([]correlationEntry, error) {
	if maxCount <= 0 {
		maxCount = 200
	}
	expired := []correlationEntry{}
	for _, queue := range s.pendingByTrackingID {
		for _, entry := range queue {
			if entry.ExpiresUTCMS > 0 && entry.ExpiresUTCMS <= nowMS {
				expired = append(expired, entry)
				if len(expired) >= maxCount {
					return expired, nil
				}
			}
		}
	}
	return expired, nil
}

func (s *inMemoryCorrelationStore) IncrementSourceOccurrences(_ context.Context, trackingID string) (int, error) {
	s.sourceOccurrences[trackingID]++
	return s.sourceOccurrences[trackingID], nil
}

func (s *inMemoryCorrelationStore) IncrementDestinationOccurrences(_ context.Context, trackingID string) (int, error) {
	s.destinationOccurrences[trackingID]++
	return s.destinationOccurrences[trackingID], nil
}

func (s *inMemoryCorrelationStore) RegisterSource(ctx context.Context, trackingID string, sourceTimestampUTCMS int64) (*correlationEntry, error) {
	entry := &correlationEntry{
		TrackingID:           trackingID,
		EventID:              generateTrackingID(),
		SourceTimestampUTCMS: sourceTimestampUTCMS,
		ProducedUTCMS:        sourceTimestampUTCMS,
		Payload:              trackingPayload{Headers: map[string]string{}},
	}
	return entry, s.Set(ctx, *entry)
}

func (s *inMemoryCorrelationStore) TryMatchDestination(ctx context.Context, trackingID string, destinationTimestampUTCMS int64) (*correlationEntry, error) {
	entry, err := s.Get(ctx, trackingID)
	if err != nil || entry == nil {
		return entry, err
	}
	if err := s.Delete(ctx, trackingID); err != nil {
		return nil, err
	}
	entry.DestinationTimestampMS = destinationTimestampUTCMS
	return entry, nil
}

func (s *inMemoryCorrelationStore) TryExpire(_ context.Context, eventID string) (bool, error) {
	entry, ok := s.pendingByEventID[eventID]
	if !ok {
		return false, nil
	}
	delete(s.pendingByEventID, eventID)
	queue := s.pendingByTrackingID[entry.TrackingID]
	next := make([]correlationEntry, 0, len(queue))
	for _, candidate := range queue {
		if candidate.EventID != eventID {
			next = append(next, candidate)
		}
	}
	if len(next) == 0 {
		delete(s.pendingByTrackingID, entry.TrackingID)
	} else {
		s.pendingByTrackingID[entry.TrackingID] = next
	}
	return true, nil
}

func (s *inMemoryCorrelationStore) Close() error {
	return nil
}

type redisCorrelationStore struct {
	client     redis.UniversalClient
	keyPrefix  string
	entryTTLMS int64
}

func newRedisCorrelationStore(options RedisCorrelationStoreSpec, runNamespace string) (*redisCorrelationStore, error) {
	parsed, err := redis.ParseURL(strings.TrimSpace(options.ConnectionString))
	if err != nil {
		return nil, err
	}
	if options.Database > 0 {
		parsed.DB = options.Database
	}
	client := redis.NewClient(parsed)
	prefix := strings.TrimSpace(options.KeyPrefix)
	if prefix == "" {
		prefix = "loadstrike"
	}
	entryTTLMS := int64(options.EntryTTLSeconds * 1000)
	if entryTTLMS <= 0 {
		entryTTLMS = int64(time.Hour / time.Millisecond)
	}
	return &redisCorrelationStore{
		client:     client,
		keyPrefix:  prefix + ":" + strings.TrimSpace(runNamespace),
		entryTTLMS: entryTTLMS,
	}, nil
}

func (s *redisCorrelationStore) entryKey(eventID string) string {
	return s.keyPrefix + ":entry:" + eventID
}

func (s *redisCorrelationStore) queueKey(trackingID string) string {
	return s.keyPrefix + ":queue:" + trackingID
}

func (s *redisCorrelationStore) sourceOccurrenceKey(trackingID string) string {
	return s.keyPrefix + ":source-count:" + trackingID
}

func (s *redisCorrelationStore) destinationOccurrenceKey(trackingID string) string {
	return s.keyPrefix + ":dest-count:" + trackingID
}

func (s *redisCorrelationStore) Set(ctx context.Context, entry correlationEntry) error {
	if entry.ExpiresUTCMS == 0 {
		entry.ExpiresUTCMS = time.Now().Add(time.Duration(s.entryTTLMS) * time.Millisecond).UnixMilli()
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, s.entryKey(entry.EventID), encoded, time.Duration(s.entryTTLMS)*time.Millisecond)
	pipe.RPush(ctx, s.queueKey(entry.TrackingID), entry.EventID)
	pipe.Expire(ctx, s.queueKey(entry.TrackingID), time.Duration(s.entryTTLMS)*time.Millisecond)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *redisCorrelationStore) Get(ctx context.Context, trackingID string) (*correlationEntry, error) {
	eventID, err := s.client.LIndex(ctx, s.queueKey(trackingID), 0).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	encoded, err := s.client.Get(ctx, s.entryKey(eventID)).Result()
	if err == redis.Nil {
		_ = s.client.LPop(ctx, s.queueKey(trackingID)).Err()
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entry correlationEntry
	if err := json.Unmarshal([]byte(encoded), &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *redisCorrelationStore) Delete(ctx context.Context, trackingID string) error {
	eventID, err := s.client.LPop(ctx, s.queueKey(trackingID)).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	return s.client.Del(ctx, s.entryKey(eventID)).Err()
}

func (s *redisCorrelationStore) All(ctx context.Context) ([]correlationEntry, error) {
	keys, err := s.client.Keys(ctx, s.keyPrefix+":entry:*").Result()
	if err != nil {
		return nil, err
	}
	entries := make([]correlationEntry, 0, len(keys))
	for _, key := range keys {
		encoded, err := s.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var entry correlationEntry
		if json.Unmarshal([]byte(encoded), &entry) == nil {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (s *redisCorrelationStore) CollectExpired(ctx context.Context, nowMS int64, maxCount int) ([]correlationEntry, error) {
	entries, err := s.All(ctx)
	if err != nil {
		return nil, err
	}
	expired := []correlationEntry{}
	for _, entry := range entries {
		if entry.ExpiresUTCMS > 0 && entry.ExpiresUTCMS <= nowMS {
			expired = append(expired, entry)
			if maxCount > 0 && len(expired) >= maxCount {
				break
			}
		}
	}
	return expired, nil
}

func (s *redisCorrelationStore) IncrementSourceOccurrences(ctx context.Context, trackingID string) (int, error) {
	return s.incrementCounter(ctx, s.sourceOccurrenceKey(trackingID))
}

func (s *redisCorrelationStore) IncrementDestinationOccurrences(ctx context.Context, trackingID string) (int, error) {
	return s.incrementCounter(ctx, s.destinationOccurrenceKey(trackingID))
}

func (s *redisCorrelationStore) incrementCounter(ctx context.Context, key string) (int, error) {
	value, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	_ = s.client.Expire(ctx, key, time.Duration(s.entryTTLMS)*time.Millisecond).Err()
	return int(value), nil
}

func (s *redisCorrelationStore) RegisterSource(ctx context.Context, trackingID string, sourceTimestampUTCMS int64) (*correlationEntry, error) {
	entry := &correlationEntry{
		TrackingID:           trackingID,
		EventID:              generateTrackingID(),
		SourceTimestampUTCMS: sourceTimestampUTCMS,
		ProducedUTCMS:        sourceTimestampUTCMS,
		ExpiresUTCMS:         time.Now().Add(time.Duration(s.entryTTLMS) * time.Millisecond).UnixMilli(),
		Payload:              trackingPayload{Headers: map[string]string{}},
	}
	return entry, s.Set(ctx, *entry)
}

func (s *redisCorrelationStore) TryMatchDestination(ctx context.Context, trackingID string, destinationTimestampUTCMS int64) (*correlationEntry, error) {
	entry, err := s.Get(ctx, trackingID)
	if err != nil || entry == nil {
		return entry, err
	}
	if err := s.Delete(ctx, trackingID); err != nil {
		return nil, err
	}
	entry.DestinationTimestampMS = destinationTimestampUTCMS
	return entry, nil
}

func (s *redisCorrelationStore) TryExpire(ctx context.Context, eventID string) (bool, error) {
	entryKey := s.entryKey(eventID)
	encoded, err := s.client.Get(ctx, entryKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var entry correlationEntry
	if err := json.Unmarshal([]byte(encoded), &entry); err != nil {
		return false, err
	}
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, entryKey)
	pipe.LRem(ctx, s.queueKey(entry.TrackingID), 0, eventID)
	_, err = pipe.Exec(ctx)
	return err == nil, err
}

func (s *redisCorrelationStore) Close() error {
	return s.client.Close()
}
