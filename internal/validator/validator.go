//file: internal/validator/validator.go

package validator

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"go.uber.org/zap"

	"message-transformer/internal/config"
)

// Common validation errors
var (
	ErrInvalidMQTTTopic   = fmt.Errorf("invalid MQTT topic format")
	ErrInvalidHTTPMethod  = fmt.Errorf("invalid HTTP method")
	ErrInvalidTemplate    = fmt.Errorf("invalid template syntax")
	ErrInvalidJSONSchema  = fmt.Errorf("invalid JSON schema")
	ErrEmptyConfiguration = fmt.Errorf("empty configuration")
)

// Validator handles validation of rules and messages
type Validator struct {
	logger *zap.Logger
	// Pre-compiled regular expressions
	topicRegex *regexp.Regexp
	// Valid HTTP methods
	validMethods map[string]bool
}

// New creates a new validator instance
func New(logger *zap.Logger) *Validator {
	return &Validator{
		logger: logger,
		topicRegex: regexp.MustCompile(`^[^#+]+(/[^#+]+)*$`),
		validMethods: map[string]bool{
			"GET":     true,
			"POST":    true,
			"PUT":     true,
			"PATCH":   true,
			"DELETE":  true,
		},
	}
}

// ValidateRule performs comprehensive validation of a rule configuration
func (v *Validator) ValidateRule(rule config.Rule) error {
	// Validate required fields
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	// Validate HTTP method
	if err := v.ValidateHTTPMethod(rule.API.Method); err != nil {
		return fmt.Errorf("invalid HTTP method for rule %s: %w", rule.ID, err)
	}

	// Validate API path
	if err := v.ValidateAPIPath(rule.API.Path); err != nil {
		return fmt.Errorf("invalid API path for rule %s: %w", rule.ID, err)
	}

	// Validate transformation template
	if err := v.ValidateTemplate(rule.Transform.Template); err != nil {
		return fmt.Errorf("invalid template for rule %s: %w", rule.ID, err)
	}

	// Validate MQTT topic
	if err := v.ValidateMQTTTopic(rule.Target.Topic); err != nil {
		return fmt.Errorf("invalid MQTT topic for rule %s: %w", rule.ID, err)
	}

	// Validate QoS level
	if err := v.ValidateQoS(rule.Target.QoS); err != nil {
		return fmt.Errorf("invalid QoS for rule %s: %w", rule.ID, err)
	}

	return nil
}

// ValidateHTTPMethod validates the HTTP method
func (v *Validator) ValidateHTTPMethod(method string) error {
	if !v.validMethods[strings.ToUpper(method)] {
		return fmt.Errorf("%w: %s", ErrInvalidHTTPMethod, method)
	}
	return nil
}

// ValidateAPIPath validates the API path format
func (v *Validator) ValidateAPIPath(path string) error {
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("API path must start with /")
	}
	return nil
}

// ValidateTemplate validates the transformation template syntax
func (v *Validator) ValidateTemplate(templateStr string) error {
	if templateStr == "" {
		return fmt.Errorf("%w: template is empty", ErrInvalidTemplate)
	}

	// Create template with all supported functions for validation
	tmpl := template.New("validator").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string { return "" },
		"fromJSON": func(s string) interface{} { return nil },
		"now": func() string { return "" },
		"uuid7": func() string { return "00000000-0000-7000-0000-000000000000" },
		"num": func(v interface{}) string { return "0" },
		"bool": func(v interface{}) string { return "false" },
	})

	if _, err := tmpl.Parse(templateStr); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTemplate, err)
	}

	return nil
}

// ValidateMQTTTopic validates the MQTT topic format
func (v *Validator) ValidateMQTTTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("%w: topic is empty", ErrInvalidMQTTTopic)
	}

	if !v.topicRegex.MatchString(topic) {
		return fmt.Errorf("%w: topic contains invalid characters", ErrInvalidMQTTTopic)
	}

	return nil
}

// ValidateQoS validates the MQTT QoS level
func (v *Validator) ValidateQoS(qos int) error {
	if qos < 0 || qos > 2 {
		return fmt.Errorf("invalid QoS level: %d, must be 0, 1, or 2", qos)
	}
	return nil
}

// ValidatePayload validates a message payload against a rule's requirements
func (v *Validator) ValidatePayload(payload []byte, rule config.Rule) error {
	if len(payload) == 0 {
		return fmt.Errorf("empty payload")
	}
	return nil
}
