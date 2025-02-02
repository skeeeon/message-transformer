package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"message-transformer/internal/api"
	"message-transformer/internal/config"
	"message-transformer/internal/mqtt"
	"message-transformer/internal/transformer"
	"message-transformer/pkg/logger"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config/app.json", "path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(logger.Config{
		Level:      cfg.Logger.Level,
		OutputPath: cfg.Logger.OutputPath,
		Encoding:   cfg.Logger.Encoding,
	})
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// Load rules
	rules, err := config.LoadRules(cfg.Rules.Directory, log)
	if err != nil {
		log.Fatal("Failed to load rules", zap.Error(err))
	}
	log.Info("Rules loaded successfully", zap.Int("count", len(rules)))

	// Initialize transformer with pre-compiled templates
	transform, err := transformer.New(log, rules)
	if err != nil {
		log.Fatal("Failed to initialize transformer", zap.Error(err))
	}

	// Initialize MQTT client
	mqttClient, err := mqtt.New(mqtt.Config{
		Broker:   cfg.MQTT.Broker,
		ClientID: cfg.MQTT.ClientID,
		Username: cfg.MQTT.Username,
		Password: cfg.MQTT.Password,
		TLS: mqtt.TLSConfig{
			Enabled: cfg.MQTT.TLS.Enabled,
			CACert:  cfg.MQTT.TLS.CACert,
			Cert:    cfg.MQTT.TLS.Cert,
			Key:     cfg.MQTT.TLS.Key,
		},
		Reconnect: mqtt.ReconnectConfig{
			Initial:    cfg.MQTT.Reconnect.Initial,
			MaxDelay:   cfg.MQTT.Reconnect.MaxDelay,
			MaxRetries: cfg.MQTT.Reconnect.MaxRetries,
		},
	}, log)
	if err != nil {
		log.Fatal("Failed to initialize MQTT client", zap.Error(err))
	}
	defer mqttClient.Close()

	// Initialize HTTP server
	server := api.NewServer(api.ServerConfig{
		Logger:      log,
		Rules:       rules,
		Transformer: transform,
		MQTT:        mqttClient,
	})

	httpServer := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler:        server,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Info("Starting HTTP server",
			zap.String("host", cfg.API.Host),
			zap.Int("port", cfg.API.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigChan
	log.Info("Received signal, initiating shutdown", zap.String("signal", sig.String()))

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown failed", zap.Error(err))
	}

	log.Info("Shutdown complete")
}
