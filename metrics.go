package loadstrike

import (
	"math"
	"sync/atomic"
)

// IMetric mirrors the .NET public metric contract.
type IMetric interface {
	MetricName() string
	UnitOfMeasure() string
}

// ICounter mirrors the .NET public counter metric contract.
type ICounter interface {
	IMetric
	Add(int64)
	Value() int64
}

// IGauge mirrors the .NET public gauge metric contract.
type IGauge interface {
	IMetric
	Set(float64)
	Value() float64
}

type counterMetric struct {
	metricName    string
	unitOfMeasure string
	value         atomic.Int64
}

// Exposes the public MetricName operation.
// Use this when the surrounding wrapper type makes this operation the clearest way to express your intent.
func (m *counterMetric) MetricName() string    { return m.metricName }
// Exposes the public UnitOfMeasure operation.
// Use this when the surrounding wrapper type makes this operation the clearest way to express your intent.
func (m *counterMetric) UnitOfMeasure() string { return m.unitOfMeasure }
// Adds a value to the current metric.
// Use this when a counter or gauge should be incremented by an explicit delta.
func (m *counterMetric) Add(value int64)       { m.value.Add(value) }
// Returns the current metric value.
// Use this when custom metric state needs to be inspected inside user code.
func (m *counterMetric) Value() int64          { return m.value.Load() }

type gaugeMetric struct {
	metricName    string
	unitOfMeasure string
	valueBits     atomic.Uint64
}

// Exposes the public MetricName operation.
// Use this when the surrounding wrapper type makes this operation the clearest way to express your intent.
func (m *gaugeMetric) MetricName() string    { return m.metricName }
// Exposes the public UnitOfMeasure operation.
// Use this when the surrounding wrapper type makes this operation the clearest way to express your intent.
func (m *gaugeMetric) UnitOfMeasure() string { return m.unitOfMeasure }
// Sets the current gauge value.
// Use this when the latest observed value should replace the previous one.
func (m *gaugeMetric) Set(value float64)     { m.valueBits.Store(math.Float64bits(value)) }
// Returns the current metric value.
// Use this when custom metric state needs to be inspected inside user code.
func (m *gaugeMetric) Value() float64        { return math.Float64frombits(m.valueBits.Load()) }

type metricNamespace struct{}

// Metric mirrors the .NET public metric factory namespace.
var Metric metricNamespace

// Creates a counter metric.
// Use this when a scenario should accumulate a monotonically increasing custom metric.
func (metricNamespace) CreateCounter(metricName string, unitOfMeasure string) ICounter {
	if metricName == "" {
		panic("metric name must be provided")
	}
	return &counterMetric{metricName: metricName, unitOfMeasure: unitOfMeasure}
}

// Creates a gauge metric.
// Use this when a scenario should track the latest value of a custom metric.
func (metricNamespace) CreateGauge(metricName string, unitOfMeasure string) IGauge {
	if metricName == "" {
		panic("metric name must be provided")
	}
	return &gaugeMetric{metricName: metricName, unitOfMeasure: unitOfMeasure}
}

type metricRegistry struct {
	metrics []IMetric
}

func (r *metricRegistry) register(metric IMetric) {
	if r == nil || metric == nil {
		return
	}
	r.metrics = append(r.metrics, metric)
}

func (r *metricRegistry) snapshot(scenarioName string) (metricStats, []metricResult) {
	if r == nil || len(r.metrics) == 0 {
		return metricStats{}, nil
	}

	stats := metricStats{
		Counters: make([]counterStats, 0, len(r.metrics)),
		Gauges:   make([]gaugeStats, 0, len(r.metrics)),
	}
	flattened := make([]metricResult, 0, len(r.metrics))
	for _, metric := range r.metrics {
		switch typed := metric.(type) {
		case ICounter:
			value := typed.Value()
			stats.Counters = append(stats.Counters, counterStats{
				MetricName:    typed.MetricName(),
				ScenarioName:  scenarioName,
				UnitOfMeasure: typed.UnitOfMeasure(),
				Value:         value,
			})
			flattened = append(flattened, metricResult{
				Name:          typed.MetricName(),
				ScenarioName:  scenarioName,
				Type:          "counter",
				Value:         float64(value),
				Unit:          typed.UnitOfMeasure(),
				UnitOfMeasure: typed.UnitOfMeasure(),
			})
		case IGauge:
			value := typed.Value()
			stats.Gauges = append(stats.Gauges, gaugeStats{
				MetricName:    typed.MetricName(),
				ScenarioName:  scenarioName,
				UnitOfMeasure: typed.UnitOfMeasure(),
				Value:         value,
			})
			flattened = append(flattened, metricResult{
				Name:          typed.MetricName(),
				ScenarioName:  scenarioName,
				Type:          "gauge",
				Value:         value,
				Unit:          typed.UnitOfMeasure(),
				UnitOfMeasure: typed.UnitOfMeasure(),
			})
		}
	}
	return stats, flattened
}
