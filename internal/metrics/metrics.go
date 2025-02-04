//file: internal/metrics/metrics.go

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Recorder provides an interface for recording essential metrics
type Recorder interface {
	// Counter methods
	IncRequests(success bool)
	IncTransforms(ruleID string, success bool)
	IncPublishes(success bool)

	// Gauge methods
	SetMQTTConnected(connected bool)
	SetActiveRules(count int)
	SetUp(up bool)
}

// PrometheusRecorder implements Recorder using Prometheus metrics
type PrometheusRecorder struct {
	// Counters
	requests   *prometheus.CounterVec
	transforms *prometheus.CounterVec
	publishes  *prometheus.CounterVec

	// Gauges
	mqttConnected *prometheus.GaugeVec
	activeRules   prometheus.Gauge
	up            prometheus.Gauge
}

// NewPrometheusRecorder creates a new PrometheusRecorder
func NewPrometheusRecorder() *PrometheusRecorder {
	return &PrometheusRecorder{
		// Initialize counters
		requests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_transformer_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"status"},
		),
		transforms: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_transformer_transforms_total",
				Help: "Total number of message transformations",
			},
			[]string{"rule_id", "status"},
		),
		publishes: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_transformer_mqtt_publishes_total",
				Help: "Total number of MQTT publish operations",
			},
			[]string{"status"},
		),

		// Initialize gauges
		mqttConnected: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "message_transformer_mqtt_connected",
				Help: "MQTT connection status (1=connected, 0=disconnected)",
			},
			[]string{"broker"},
		),
		activeRules: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "message_transformer_active_rules",
				Help: "Number of active transformation rules",
			},
		),
		up: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "message_transformer_up",
				Help: "Whether the message transformer is up (1) or down (0)",
			},
		),
	}
}

// Counter method implementations
func (r *PrometheusRecorder) IncRequests(success bool) {
	status := statusLabel(success)
	r.requests.WithLabelValues(status).Inc()
}

func (r *PrometheusRecorder) IncTransforms(ruleID string, success bool) {
	status := statusLabel(success)
	r.transforms.WithLabelValues(ruleID, status).Inc()
}

func (r *PrometheusRecorder) IncPublishes(success bool) {
	status := statusLabel(success)
	r.publishes.WithLabelValues(status).Inc()
}

// Gauge method implementations
func (r *PrometheusRecorder) SetMQTTConnected(connected bool) {
	value := 0.0
	if connected {
		value = 1.0
	}
	r.mqttConnected.WithLabelValues("broker").Set(value)
}

func (r *PrometheusRecorder) SetActiveRules(count int) {
	r.activeRules.Set(float64(count))
}

func (r *PrometheusRecorder) SetUp(up bool) {
	value := 0.0
	if up {
		value = 1.0
	}
	r.up.Set(value)
}

// Helper function for status labels
func statusLabel(success bool) string {
	if success {
		return "success"
	}
	return "error"
}

// NoOpRecorder implements Recorder with no-op operations for testing
type NoOpRecorder struct{}

func NewNoOpRecorder() *NoOpRecorder {
	return &NoOpRecorder{}
}

// NoOp implementations
func (r *NoOpRecorder) IncRequests(success bool)                 {}
func (r *NoOpRecorder) IncTransforms(ruleID string, success bool) {}
func (r *NoOpRecorder) IncPublishes(success bool)                {}
func (r *NoOpRecorder) SetMQTTConnected(connected bool)          {}
func (r *NoOpRecorder) SetActiveRules(count int)                 {}
func (r *NoOpRecorder) SetUp(up bool)                            {}
