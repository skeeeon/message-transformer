//file: internal/mqtt/client.go

package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"message-transformer/internal/metrics"
)

// Client wraps the MQTT client functionality
type Client struct {
	client   paho.Client
	logger   *zap.Logger
	metrics  metrics.Recorder
	broker   string
}

// Config holds the MQTT client configuration
type Config struct {
	Broker     string
	ClientID   string
	Username   string
	Password   string
	TLS        TLSConfig
	Reconnect  ReconnectConfig
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled bool
	CACert  string
	Cert    string
	Key     string
}

// ReconnectConfig holds reconnection configuration
type ReconnectConfig struct {
	Initial    int
	MaxDelay   int
	MaxRetries int
}

// New creates a new MQTT client with metrics recording
func New(cfg Config, logger *zap.Logger, metricsRecorder metrics.Recorder) (*Client, error) {
	if metricsRecorder == nil {
		metricsRecorder = metrics.NewNoOpRecorder()
	}

	client := &Client{
		logger:  logger,
		metrics: metricsRecorder,
		broker:  cfg.Broker,
	}

	opts := paho.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetOrderMatters(false).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(time.Duration(cfg.Reconnect.Initial) * time.Second).
		SetMaxReconnectInterval(time.Duration(cfg.Reconnect.MaxDelay) * time.Second)

	// Configure TLS if enabled
	if cfg.TLS.Enabled {
		tlsConfig, err := createTLSConfig(cfg.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// Configure connection callbacks with metrics
	opts.SetConnectionLostHandler(func(c paho.Client, err error) {
		logger.Warn("MQTT connection lost", zap.Error(err))
		client.metrics.SetMQTTConnectionStatus(false)
	})

	opts.SetOnConnectHandler(func(c paho.Client) {
		logger.Info("MQTT connected successfully")
		client.metrics.SetMQTTConnectionStatus(true)
	})

	opts.SetReconnectingHandler(func(c paho.Client, opts *paho.ClientOptions) {
		logger.Info("MQTT attempting reconnection")
		client.metrics.IncMQTTReconnections()
	})

	mqttClient := paho.NewClient(opts)
	client.client = mqttClient

	// Initial connection with retry and metrics
	retries := 0
	for {
		token := mqttClient.Connect()
		if token.WaitTimeout(time.Duration(cfg.Reconnect.Initial) * time.Second) {
			if token.Error() != nil {
				if retries >= cfg.Reconnect.MaxRetries {
					return nil, fmt.Errorf("failed to connect after %d retries: %w", retries, token.Error())
				}
				logger.Warn("Failed to connect, retrying...",
					zap.Error(token.Error()),
					zap.Int("retry", retries+1),
					zap.Int("maxRetries", cfg.Reconnect.MaxRetries))
				client.metrics.SetMQTTConnectionStatus(false)
				retries++
				time.Sleep(time.Duration(cfg.Reconnect.Initial) * time.Second)
				continue
			}
			break
		}
		return nil, fmt.Errorf("connection timeout")
	}

	// Set initial connection status metric
	client.metrics.SetMQTTConnectionStatus(true)

	return client, nil
}

// Publish publishes a message to the specified topic with metrics
func (c *Client) Publish(topic string, qos int, retain bool, payload []byte) error {
	start := time.Now()
	c.metrics.IncMQTTPublishAttempts(topic)

	token := c.client.Publish(topic, byte(qos), retain, payload)
	if !token.WaitTimeout(10 * time.Second) {
		c.metrics.IncMQTTPublishFailures(topic)
		return fmt.Errorf("publish timeout")
	}

	if err := token.Error(); err != nil {
		c.metrics.IncMQTTPublishFailures(topic)
		return err
	}

	c.metrics.ObserveMQTTPublishDuration(topic, time.Since(start).Seconds())
	return nil
}

// Subscribe subscribes to the specified topic
func (c *Client) Subscribe(topic string, qos int, callback func([]byte)) error {
	token := c.client.Subscribe(topic, byte(qos), func(client paho.Client, msg paho.Message) {
		callback(msg.Payload())
	})
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("subscribe timeout")
	}
	return token.Error()
}

// Close disconnects the client and updates metrics
func (c *Client) Close() {
	if c.client.IsConnected() {
		c.client.Disconnect(250)
		c.metrics.SetMQTTConnectionStatus(false)
	}
}

// createTLSConfig creates a TLS configuration for the MQTT client
func createTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	// Load client cert/key if specified
	var certificates []tls.Certificate
	if cfg.Cert != "" && cfg.Key != "" {
		cert, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		certificates = append(certificates, cert)
	}

	// Load CA cert if specified
	var caCertPool *x509.CertPool
	if cfg.CACert != "" {
		caCertPool = x509.NewCertPool()
		caCert, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
	}

	return &tls.Config{
		Certificates:       certificates,
		RootCAs:           caCertPool,
		MinVersion:        tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}, nil
}

// IsConnected returns the connection status and updates metrics
func (c *Client) IsConnected() bool {
	connected := c.client != nil && c.client.IsConnected()
	c.metrics.SetMQTTConnectionStatus(connected)
	return connected
}
