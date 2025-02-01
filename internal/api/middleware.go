package api

import (
	"net/http"
	"time"
	"encoding/json"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// NewStructuredLogger creates a new structured logger middleware
func NewStructuredLogger(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info("HTTP Request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
					zap.String("request_id", middleware.GetReqID(r.Context())),
					zap.Int("status", ww.Status()),
					zap.Int("bytes_written", ww.BytesWritten()),
					zap.Float64("duration", time.Since(start).Seconds()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
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
