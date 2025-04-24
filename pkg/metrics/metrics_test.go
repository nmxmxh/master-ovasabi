package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()
	assert.NotNil(t, collector)
	assert.Empty(t, collector.metrics)
}

func TestMetricsCollector_Record(t *testing.T) {
	collector := NewMetricsCollector()
	metric := Metric{
		Name:      "test_metric",
		Type:      Counter,
		Value:     42.0,
		Labels:    map[string]string{"label1": "value1"},
		Timestamp: time.Now(),
	}

	collector.Record(metric)
	metrics := collector.GetMetrics()

	assert.Len(t, metrics, 1)
	assert.Equal(t, metric.Name, metrics[0].Name)
	assert.Equal(t, metric.Type, metrics[0].Type)
	assert.Equal(t, metric.Value, metrics[0].Value)
	assert.Equal(t, metric.Labels, metrics[0].Labels)
}

func TestCounterMetric(t *testing.T) {
	collector := NewMetricsCollector()
	labels := map[string]string{"service": "test"}
	counter := NewCounterMetric("requests_total", 1.0, labels)

	counter.Record(collector)
	metrics := collector.GetMetrics()

	assert.Len(t, metrics, 1)
	assert.Equal(t, "requests_total", metrics[0].Name)
	assert.Equal(t, Counter, metrics[0].Type)
	assert.Equal(t, 1.0, metrics[0].Value)
	assert.Equal(t, labels, metrics[0].Labels)
}

func TestGaugeMetric(t *testing.T) {
	collector := NewMetricsCollector()
	labels := map[string]string{"service": "test"}
	gauge := NewGaugeMetric("memory_usage", 1024.0, labels)

	gauge.Record(collector)
	metrics := collector.GetMetrics()

	assert.Len(t, metrics, 1)
	assert.Equal(t, "memory_usage", metrics[0].Name)
	assert.Equal(t, Gauge, metrics[0].Type)
	assert.Equal(t, 1024.0, metrics[0].Value)
	assert.Equal(t, labels, metrics[0].Labels)
}

func TestHistogramMetric(t *testing.T) {
	collector := NewMetricsCollector()
	labels := map[string]string{"service": "test"}
	histogram := NewHistogramMetric("response_time", 0.42, labels)

	histogram.Record(collector)
	metrics := collector.GetMetrics()

	assert.Len(t, metrics, 1)
	assert.Equal(t, "response_time", metrics[0].Name)
	assert.Equal(t, Histogram, metrics[0].Type)
	assert.Equal(t, 0.42, metrics[0].Value)
	assert.Equal(t, labels, metrics[0].Labels)
}

func TestConcurrentMetricRecording(t *testing.T) {
	collector := NewMetricsCollector()
	done := make(chan bool)

	// Record metrics concurrently
	for i := 0; i < 10; i++ {
		go func(i int) {
			counter := NewCounterMetric(
				"concurrent_test",
				float64(i),
				map[string]string{"goroutine": string(rune('A' + i))},
			)
			counter.Record(collector)
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := collector.GetMetrics()
	assert.Len(t, metrics, 10)
}

func TestPrometheusExporter(t *testing.T) {
	exporter := NewPrometheusExporter()
	assert.NotNil(t, exporter)

	metrics := []Metric{
		{
			Name:      "test_counter",
			Type:      Counter,
			Value:     1.0,
			Labels:    map[string]string{"test": "true"},
			Timestamp: time.Now(),
		},
	}

	err := exporter.Export(context.Background(), metrics)
	assert.NoError(t, err)
}

func TestStatsDExporter(t *testing.T) {
	exporter := NewStatsDExporter()
	assert.NotNil(t, exporter)

	metrics := []Metric{
		{
			Name:      "test_gauge",
			Type:      Gauge,
			Value:     42.0,
			Labels:    map[string]string{"test": "true"},
			Timestamp: time.Now(),
		},
	}

	err := exporter.Export(context.Background(), metrics)
	assert.NoError(t, err)
}

func TestMetricTimestamps(t *testing.T) {
	collector := NewMetricsCollector()
	start := time.Now()

	// Record metrics with some delay
	counter := NewCounterMetric("test_counter", 1.0, nil)
	counter.Record(collector)

	time.Sleep(time.Millisecond)

	gauge := NewGaugeMetric("test_gauge", 2.0, nil)
	gauge.Record(collector)

	metrics := collector.GetMetrics()
	assert.Len(t, metrics, 2)

	// Verify timestamps are after start time and in correct order
	assert.True(t, metrics[0].Timestamp.After(start) || metrics[0].Timestamp.Equal(start))
	assert.True(t, metrics[1].Timestamp.After(metrics[0].Timestamp))
}

func TestMetricLabels(t *testing.T) {
	collector := NewMetricsCollector()
	labels := map[string]string{
		"service":    "test",
		"endpoint":   "/api",
		"method":     "GET",
		"status":     "200",
		"datacenter": "us-west",
	}

	metrics := []struct {
		name  string
		value float64
		typ   MetricType
	}{
		{"requests_total", 1.0, Counter},
		{"response_time", 0.2, Histogram},
		{"memory_usage", 1024.0, Gauge},
	}

	for _, m := range metrics {
		switch m.typ {
		case Counter:
			NewCounterMetric(m.name, m.value, labels).Record(collector)
		case Gauge:
			NewGaugeMetric(m.name, m.value, labels).Record(collector)
		case Histogram:
			NewHistogramMetric(m.name, m.value, labels).Record(collector)
		}
	}

	recorded := collector.GetMetrics()
	assert.Len(t, recorded, len(metrics))

	for i, m := range recorded {
		assert.Equal(t, metrics[i].name, m.Name)
		assert.Equal(t, metrics[i].value, m.Value)
		assert.Equal(t, metrics[i].typ, m.Type)
		assert.Equal(t, labels, m.Labels)
	}
}
