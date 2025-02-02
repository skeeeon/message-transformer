//internal/config/config.go
package config

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/spf13/viper"
)

// Validation constants
const (
	maxQoSLevel = 2
	minQoSLevel = 0
)

var (
	// Compile regex patterns once
	topicRegex = regexp.MustCompile(`^[^#+]+(/[^#+]+)*$`)
	methodRegex = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE)$`)
)

// AppConfig represents the main application configuration
type AppConfig struct {
	MQTT   MQTTConfig   `json:"mqtt"`
	API    APIConfig    `json:"api"`
	Rules  RulesConfig  `json:"rules"`
	Logger LoggerConfig `json:"logger"`
}

// MQTTConfig holds MQTT connection configuration
type MQTTConfig struct {
	Broker   string `json:"broker"`
	ClientID string `json:"clientId"`
	Username string `json:"username"`
	Password string `json:"password"`
	TLS      struct {
		Enabled bool   `json:"enabled"`
		CACert  string `json:"caCert"`
		Cert    string `json:"cert"`
		Key     string `json:"key"`
	} `json:"tls"`
	Reconnect struct {
		Initial    int `json:"initial"`
		MaxDelay   int `json:"maxDelay"`
		MaxRetries int `json:"maxRetries"`
	} `json:"reconnect"`
}

// APIConfig holds REST API configuration
type APIConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// RulesConfig holds rules directory configuration
type RulesConfig struct {
	Directory string `json:"directory"`
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level      string `json:"level"`
	OutputPath string `json:"outputPath"`
	Encoding   string `json:"encoding"`
}

// LoadConfig loads and validates the application configuration
func LoadConfig(configPath string) (*AppConfig, error) {
	v := viper.New()
	v.SetConfigType("json")
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AppConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure rules directory is absolute path
	if !filepath.IsAbs(config.Rules.Directory) {
		config.Rules.Directory = filepath.Join(filepath.Dir(configPath), config.Rules.Directory)
	}

	return &config, nil
}

// Validate validates the application configuration
func (c *AppConfig) Validate() error {
	// Validate MQTT configuration
	if c.MQTT.Broker == "" {
		return fmt.Errorf("MQTT broker URL is required")
	}
	if c.MQTT.ClientID == "" {
		return fmt.Errorf("MQTT client ID is required")
	}

	// Validate API configuration
	if c.API.Port <= 0 || c.API.Port > 65535 {
		return fmt.Errorf("invalid API port number")
	}

	// Validate TLS configuration if enabled
	if c.MQTT.TLS.Enabled {
		if c.MQTT.TLS.CACert == "" {
			return fmt.Errorf("CA certificate is required when TLS is enabled")
		}
	}

	return nil
}

// ValidateQoS validates a QoS level
func ValidateQoS(qos int) error {
	if qos < minQoSLevel || qos > maxQoSLevel {
		return fmt.Errorf("invalid QoS level: %d, must be between %d and %d", qos, minQoSLevel, maxQoSLevel)
	}
	return nil
}

// ValidateTopic validates an MQTT topic string
func ValidateTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	if !topicRegex.MatchString(topic) {
		return fmt.Errorf("invalid topic format: %s", topic)
	}
	return nil
}

// ValidateHTTPMethod validates an HTTP method
func ValidateHTTPMethod(method string) error {
	if !methodRegex.MatchString(method) {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}
	return nil
}
