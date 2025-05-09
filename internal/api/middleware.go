//file: internal/api/middleware.go

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"message-transformer/internal/metrics"
)

// NewStructuredLogger creates a new structured logger middleware
func NewStructuredLogger(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get buffered writer or create new one if needed
			bw, ok := w.(ResponseWriter)
			if !ok {
				bw = newBufferedResponseWriter(w, make([]byte, 32*1024))
			}

			defer func() {
				duration := time.Since(start)

				logger.Info("HTTP Request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
					zap.String("request_id", middleware.GetReqID(r.Context())),
					zap.Int("status", bw.Status()),
					zap.Int("bytes_written", bw.BytesWritten()),
					zap.Float64("duration_ms", float64(duration.Milliseconds())),
				)

				if buffered, ok := bw.(*bufferedResponseWriter); ok {
					buffered.Flush()
				}
			}()

			next.ServeHTTP(bw, r)
		})
	}
}

// MetricsMiddleware creates middleware for recording request metrics
func MetricsMiddleware(recorder metrics.Recorder) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip metrics for the metrics endpoint itself
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			// Get buffered writer or create new one if needed
			bw, ok := w.(ResponseWriter)
			if !ok {
				bw = newBufferedResponseWriter(w, make([]byte, 32*1024))
			}

			next.ServeHTTP(bw, r)

			// Record request success based on status code
			success := bw.Status() < 500
			recorder.IncRequests(success)
		})
	}
}

// PrometheusMetricsHandler returns the Prometheus metrics HTTP handler
func PrometheusMetricsHandler() http.Handler {
	return promhttp.Handler()
}

// JSONResponse writes a JSON response with the given status code
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SendError sends an error response with the given status code
func SendError(w http.ResponseWriter, status int, message string) {
	JSONResponse(w, status, ErrorResponse{Error: message})
}
