//file: internal/api/handler.go

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	"message-transformer/internal/config"
	"message-transformer/internal/transformer"
)

const (
	maxRequestSize = 1 << 20 // 1MB
)

// handleHealth returns a handler for health check requests
func (s *Server) handleHealth() http.HandlerFunc {
	type healthResponse struct {
		Status    string `json:"status"`
		MQTTConn bool   `json:"mqtt_connected"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Get or create buffered writer
		var bw ResponseWriter
		if buffered, ok := w.(ResponseWriter); ok {
			bw = buffered
		} else {
			buffer := s.bufferPool.Get().([]byte)
			buffered := newBufferedResponseWriter(w, buffer)
			defer func() {
				buffered.Flush()
				s.bufferPool.Put(buffer)
			}()
			bw = buffered
		}

		resp := healthResponse{
			Status:    "ok",
			MQTTConn: s.mqtt.IsConnected(),
		}
		JSONResponse(bw, http.StatusOK, resp)
	}
}

// handleTransform returns a handler for transformation requests
func (s *Server) handleTransform(rule config.Rule) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get or create buffered writer
		var bw ResponseWriter
		if buffered, ok := w.(ResponseWriter); ok {
			bw = buffered
		} else {
			buffer := s.bufferPool.Get().([]byte)
			buffered := newBufferedResponseWriter(w, buffer)
			defer func() {
				buffered.Flush()
				s.bufferPool.Put(buffer)
			}()
			bw = buffered
		}

		// Read request body with size limit
		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestSize))
		if err != nil {
			s.logger.Error("Failed to read request body",
				zap.Error(err),
				zap.String("rule_id", rule.ID))
			SendError(bw, http.StatusBadRequest, "Failed to read request body")
			return
		}
		defer r.Body.Close()

		// Verify the input is valid JSON before processing
		if !json.Valid(body) {
			s.logger.Error("Invalid JSON in request body",
				zap.String("rule_id", rule.ID))
			SendError(bw, http.StatusBadRequest, "Invalid JSON in request body")
			return
		}

		// Transform message using pre-compiled template
		transformed, err := s.transformer.Transform(rule.ID, body)
		if err != nil {
			var transformErr *transformer.TransformError
			if errors.As(err, &transformErr) {
				s.logger.Error("Transform error",
					zap.Error(transformErr.Err),
					zap.String("message", transformErr.Message),
					zap.String("rule_id", rule.ID))
				SendError(bw, http.StatusUnprocessableEntity,
					fmt.Sprintf("Transform error: %s", transformErr.Message))
				return
			}
			s.logger.Error("Unexpected transform error",
				zap.Error(err),
				zap.String("rule_id", rule.ID))
			SendError(bw, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Publish to MQTT
		if err := s.mqtt.Publish(
			rule.Target.Topic,
			rule.Target.QoS,
			rule.Target.Retain,
			transformed,
		); err != nil {
			s.logger.Error("Failed to publish to MQTT",
				zap.Error(err),
				zap.String("rule_id", rule.ID),
				zap.String("topic", rule.Target.Topic))
			SendError(bw, http.StatusServiceUnavailable, "Failed to publish message")
			return
		}

		// Parse transformed data for response
		var preview interface{}
		if err := json.Unmarshal(transformed, &preview); err != nil {
			s.logger.Error("Failed to parse transformed data for response",
				zap.Error(err),
				zap.String("rule_id", rule.ID))
			SendError(bw, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Return success response with transformed data preview
		JSONResponse(bw, http.StatusOK, struct {
			Status      string      `json:"status"`
			RuleID      string      `json:"rule_id"`
			Transformed interface{} `json:"transformed"`
		}{
			Status:      "published",
			RuleID:      rule.ID,
			Transformed: preview,
		})
	}
}
