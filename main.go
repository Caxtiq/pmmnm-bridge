package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logger.WithField("config", *configFile).Info("Starting PMMNM MQTT Bridge")

	// Load configuration
	config, err := LoadConfig(*configFile)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		logger.WithError(err).Fatal("Invalid configuration")
	}

	// Configure logger based on config
	level, err := logrus.ParseLevel(config.Logging.Level)
	if err != nil {
		logger.WithError(err).Warn("Invalid log level, using info")
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	if config.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	logger.WithFields(logrus.Fields{
		"api_endpoint": config.API.Endpoint,
		"mqtt_broker":  config.MQTT.Broker,
		"topics":       len(config.Topics),
		"log_level":    config.Logging.Level,
	}).Info("Configuration loaded successfully")

	// Create API client
	apiClient := NewAPIClient(config.API.Endpoint, config.API.Timeout, logger)

	// Create MQTT bridge
	bridge := NewMQTTBridge(config, apiClient, logger)

	// Connect to MQTT broker
	if err := bridge.Connect(); err != nil {
		logger.WithError(err).Fatal("Failed to connect to MQTT broker")
	}

	logger.Info("Bridge is running. Press Ctrl+C to exit.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down bridge...")
	bridge.Disconnect()
	logger.Info("Bridge stopped")
}
