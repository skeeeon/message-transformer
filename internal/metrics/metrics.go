//file: internal/metrics/metrics.go

package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Recorder provides an interface for recording metrics
type Recorder interface {
	// HTTP metrics
	ObserveRequestDuration(path, method string, statusCode int, duration float64)
	ObserveRequestSize(path string, size int64)
	ObserveResponseSize(path string, size int64)

	// MQTT metrics
	SetMQTTConnectionStatus(connected bool)
	IncMQTTPublishAttempts(topic string)
	IncMQTTPublishFailures(topic string)
	ObserveMQTTPublishDuration(topic string, duration float64)
	IncMQTTReconnections()

	// Transformer metrics
	ObserveTransformDuration(ruleID string, duration float64)
	ObserveTransformInputSize(ruleID string, size int64)
	ObserveTransformOutputSize(ruleID string, size int64)
	IncTransformErrors(ruleID string)
	IncTemplateErrors(ruleID string)

	// System metrics
	SetBufferPoolUtilization(utilization float64)
	SetActiveRules(count int)
}

// PrometheusRecorder implements Recorder using Prometheus metrics
type PrometheusRecorder struct {
	// HTTP metrics
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize    *prometheus.HistogramVec
	httpResponseSize   *prometheus.HistogramVec

	// MQTT metrics
	mqttConnectionStatus *prometheus.GaugeVec
	mqttPublishAttempts *prometheus.CounterVec
	mqttPublishFailures *prometheus.CounterVec
	mqttPublishDuration *prometheus.HistogramVec
	mqttReconnections   prometheus.Counter

	// Transformer metrics
	transformDuration    *prometheus.HistogramVec
	transformInputSize   *prometheus.HistogramVec
	transformOutputSize  *prometheus.HistogramVec
	transformErrors      *prometheus.CounterVec
	templateErrors       *prometheus.CounterVec

	// System metrics
	bufferPoolUtilization prometheus.Gauge
	activeRules          prometheus.Gauge
}

// NewPrometheusRecorder creates a new PrometheusRecorder
func NewPrometheusRecorder() *PrometheusRecorder {
	return &PrometheusRecorder{
		// HTTP metrics
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "message_transformer_http_request_duration_seconds",
				Help: "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"path", "method", "status"},
		),
		httpRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "message_transformer_http_request_size_bytes",
				Help: "Size of HTTP requests in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"path"},
		),
		httpResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "message_transformer_http_response_size_bytes",
				Help: "Size of HTTP responses in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"path"},
		),

		// MQTT metrics
		mqttConnectionStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "message_transformer_mqtt_connected",
				Help: "MQTT connection status (1 for connected, 0 for disconnected)",
			},
			[]string{"broker"},
		),
		mqttPublishAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_transformer_mqtt_publish_attempts_total",
				Help: "Total number of MQTT publish attempts",
			},
			[]string{"topic"},
		),
		mqttPublishFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_transformer_mqtt_publish_failures_total",
				Help: "Total number of failed MQTT publish attempts",
			},
			[]string{"topic"},
		),
		mqttPublishDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "message_transformer_mqtt_publish_duration_seconds",
				Help: "Duration of MQTT publish operations in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"topic"},
		),
		mqttReconnections: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "message_transformer_mqtt_reconnections_total",
				Help: "Total number of MQTT reconnection attempts",
			},
		),

		// Transformer metrics
		transformDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "message_transformer_transform_duration_seconds",
				Help: "Duration of message transformations in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"rule_id"},
		),
		transformInputSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "message_transformer_transform_input_size_bytes",
				Help: "Size of input messages in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"rule_id"},
		),
		transformOutputSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "message_transformer_transform_output_size_bytes",
				Help: "Size of transformed messages in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"rule_id"},
		),
		transformErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_transformer_transform_errors_total",
				Help: "Total number of transformation errors",
			},
			[]string{"rule_id"},
		),
		templateErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "message_transformer_template_errors_total",
				Help: "Total number of template execution errors",
			},
			[]string{"rule_id"},
		),

		// System metrics
		bufferPoolUtilization: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "message_transformer_buffer_pool_utilization",
				Help: "Current buffer pool utilization",
			},
		),
		activeRules: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "message_transformer_active_rules",
				Help: "Number of active transformation rules",
			},
		),
	}
}

// Implementation of the Recorder interface for PrometheusRecorder

func (r *PrometheusRecorder) ObserveRequestDuration(path, method string, statusCode int, duration float64) {
	r.httpRequestDuration.WithLabelValues(path, method, fmt.Sprintf("%d", statusCode)).Observe(duration)
}

func (r *PrometheusRecorder) ObserveRequestSize(path string, size int64) {
	r.httpRequestSize.WithLabelValues(path).Observe(float64(size))
}

func (r *PrometheusRecorder) ObserveResponseSize(path string, size int64) {
	r.httpResponseSize.WithLabelValues(path).Observe(float64(size))
}

func (r *PrometheusRecorder) SetMQTTConnectionStatus(connected bool) {
	value := 0.0
	if connected {
		value = 1.0
	}
	r.mqttConnectionStatus.WithLabelValues("broker").Set(value)
}

func (r *PrometheusRecorder) IncMQTTPublishAttempts(topic string) {
	r.mqttPublishAttempts.WithLabelValues(topic).Inc()
}

func (r *PrometheusRecorder) IncMQTTPublishFailures(topic string) {
	r.mqttPublishFailures.WithLabelValues(topic).Inc()
}

func (r *PrometheusRecorder) ObserveMQTTPublishDuration(topic string, duration float64) {
	r.mqttPublishDuration.WithLabelValues(topic).Observe(duration)
}

func (r *PrometheusRecorder) IncMQTTReconnections() {
	r.mqttReconnections.Inc()
}

func (r *PrometheusRecorder) ObserveTransformDuration(ruleID string, duration float64) {
	r.transformDuration.WithLabelValues(ruleID).Observe(duration)
}

func (r *PrometheusRecorder) ObserveTransformInputSize(ruleID string, size int64) {
	r.transformInputSize.WithLabelValues(ruleID).Observe(float64(size))
}

func (r *PrometheusRecorder) ObserveTransformOutputSize(ruleID string, size int64) {
	r.transformOutputSize.WithLabelValues(ruleID).Observe(float64(size))
}

func (r *PrometheusRecorder) IncTransformErrors(ruleID string) {
	r.transformErrors.WithLabelValues(ruleID).Inc()
}

func (r *PrometheusRecorder) IncTemplateErrors(ruleID string) {
	r.templateErrors.WithLabelValues(ruleID).Inc()
}

func (r *PrometheusRecorder) SetBufferPoolUtilization(utilization float64) {
	r.bufferPoolUtilization.Set(utilization)
}

func (r *PrometheusRecorder) SetActiveRules(count int) {
	r.activeRules.Set(float64(count))
}

// NoOpRecorder implements Recorder with no-op operations for testing
type NoOpRecorder struct{}

func NewNoOpRecorder() *NoOpRecorder {
	return &NoOpRecorder{}
}

// Implement all Recorder methods as no-ops
func (r *NoOpRecorder) ObserveRequestDuration(path, method string, statusCode int, duration float64) {}
func (r *NoOpRecorder) ObserveRequestSize(path string, size int64)                                   {}
func (r *NoOpRecorder) ObserveResponseSize(path string, size int64)                                  {}
func (r *NoOpRecorder) SetMQTTConnectionStatus(connected bool)                                       {}
func (r *NoOpRecorder) IncMQTTPublishAttempts(topic string)                                         {}
func (r *NoOpRecorder) IncMQTTPublishFailures(topic string)                                         {}
func (r *NoOpRecorder) ObserveMQTTPublishDuration(topic string, duration float64)                   {}
func (r *NoOpRecorder) IncMQTTReconnections()                                                        {}
func (r *NoOpRecorder) ObserveTransformDuration(ruleID string, duration float64)                    {}
func (r *NoOpRecorder) ObserveTransformInputSize(ruleID string, size int64)                         {}
func (r *NoOpRecorder) ObserveTransformOutputSize(ruleID string, size int64)                        {}
func (r *NoOpRecorder) IncTransformErrors(ruleID string)                                            {}
func (r *NoOpRecorder) IncTemplateErrors(ruleID string)                                             {}
func (r *NoOpRecorder) SetBufferPoolUtilization(utilization float64)                                {}
func (r *NoOpRecorder) SetActiveRules(count int)                                                    {}
