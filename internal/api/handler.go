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

// handleHealth returns a handler for health check requests
func (s *Server) handleHealth() http.HandlerFunc {
	type healthResponse struct {
		Status    string `json:"status"`
		MQTTConn bool   `json:"mqtt_connected"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		resp := healthResponse{
			Status:    "ok",
			MQTTConn: s.mqtt.IsConnected(),
		}
		JSONResponse(w, http.StatusOK, resp)
	}
}

// handleTransform returns a handler for transformation requests
func (s *Server) handleTransform(rule config.Rule) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.logger.Error("Failed to read request body",
				zap.Error(err),
				zap.String("rule_id", rule.ID))
			SendError(w, http.StatusBadRequest, "Failed to read request body")
			return
		}

		// Verify the input is valid JSON
		if !json.Valid(body) {
			s.logger.Error("Invalid JSON in request body",
				zap.String("rule_id", rule.ID))
			SendError(w, http.StatusBadRequest, "Invalid JSON in request body")
			return
		}

		// Transform message
		transformed, err := s.transformer.Transform(rule.Transform.Template, body)
		if err != nil {
			var transformErr *transformer.TransformError
			if errors.As(err, &transformErr) {
				s.logger.Error("Transform error",
					zap.Error(transformErr.Err),
					zap.String("message", transformErr.Message),
					zap.String("rule_id", rule.ID))
				SendError(w, http.StatusUnprocessableEntity, 
					fmt.Sprintf("Transform error: %s", transformErr.Message))
				return
			}
			s.logger.Error("Unexpected transform error",
				zap.Error(err),
				zap.String("rule_id", rule.ID))
			SendError(w, http.StatusInternalServerError, "Internal server error")
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
			SendError(w, http.StatusServiceUnavailable, "Failed to publish message")
			return
		}

		// Return success response with transformed data preview
		var preview interface{}
		_ = json.Unmarshal(transformed, &preview) // Error already checked in transformer

		JSONResponse(w, http.StatusOK, struct {
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
