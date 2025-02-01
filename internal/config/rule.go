package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"go.uber.org/zap"
)

// Rule represents a single message transformation rule
type Rule struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	API         RuleAPI    `json:"api"`
	Transform   Transform  `json:"transform"`
	Target      TargetMQTT `json:"target"`
}

// RuleAPI holds the API configuration for a rule
type RuleAPI struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Transform holds the message transformation configuration
type Transform struct {
	Template string `json:"template"`
}

// TargetMQTT holds the target MQTT configuration for transformed messages
type TargetMQTT struct {
	Topic   string `json:"topic"`
	QoS     int    `json:"qos"`
	Retain  bool   `json:"retain"`
}

// LoadRules loads and validates all rules from the specified directory
func LoadRules(rulesDir string, logger *zap.Logger) ([]Rule, error) {
	files, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules directory: %w", err)
	}

	var rules []Rule
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Read the JSON file
		data, err := os.ReadFile(filepath.Join(rulesDir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read rule file %s: %w", file.Name(), err)
		}

		var rule Rule
		if err := json.Unmarshal(data, &rule); err != nil {
			return nil, fmt.Errorf("failed to parse rule file %s: %w", file.Name(), err)
		}

		// Validate the rule
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("invalid rule in file %s: %w", file.Name(), err)
		}

		logger.Info("Loaded rule",
			zap.String("id", rule.ID),
			zap.String("file", file.Name()))

		rules = append(rules, rule)
	}

	return rules, nil
}

// Validate validates a rule configuration
func (r *Rule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	// Validate API configuration
	if err := ValidateHTTPMethod(r.API.Method); err != nil {
		return fmt.Errorf("invalid API configuration: %w", err)
	}
	if r.API.Path == "" || r.API.Path[0] != '/' {
		return fmt.Errorf("API path must start with /")
	}

	// Validate template
	if r.Transform.Template == "" {
		return fmt.Errorf("transformation template is required")
	}

	// Create template with all supported functions for validation
	tmpl := template.New("validator").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			b, err := json.Marshal(v)
			if err != nil {
				return ""
			}
			return string(b)
		},
		"fromJSON": func(s string) interface{} {
			var v interface{}
			if err := json.Unmarshal([]byte(s), &v); err != nil {
				return nil
			}
			return v
		},
		"now": func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
		"num": func(v interface{}) string {
			switch n := v.(type) {
			case float64:
				return strconv.FormatFloat(n, 'f', -1, 64)
			case float32:
				return strconv.FormatFloat(float64(n), 'f', -1, 32)
			case int:
				return strconv.Itoa(n)
			case int64:
				return strconv.FormatInt(n, 10)
			case int32:
				return strconv.FormatInt(int64(n), 10)
			case string:
				if _, err := strconv.ParseFloat(n, 64); err == nil {
					return n
				}
				return "0"
			default:
				return "0"
			}
		},
		"bool": func(v interface{}) string {
			switch b := v.(type) {
			case bool:
				return strconv.FormatBool(b)
			case string:
				if b == "true" || b == "false" {
					return b
				}
				return "false"
			case int, int64, float64:
				return "true"
			case nil:
				return "false"
			default:
				return "false"
			}
		},
	})

	if _, err := tmpl.Parse(r.Transform.Template); err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}

	// Validate MQTT configuration
	if err := ValidateTopic(r.Target.Topic); err != nil {
		return fmt.Errorf("invalid target configuration: %w", err)
	}
	if err := ValidateQoS(r.Target.QoS); err != nil {
		return fmt.Errorf("invalid target configuration: %w", err)
	}

	return nil
}
