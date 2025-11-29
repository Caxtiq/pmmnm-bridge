package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration structure
type Config struct {
	API     APIConfig     `yaml:"api"`
	MQTT    MQTTConfig    `yaml:"mqtt"`
	Topics  []TopicMap    `yaml:"topics"`
	Logging LoggingConfig `yaml:"logging"`
}

// APIConfig contains API endpoint settings
type APIConfig struct {
	Endpoint string        `yaml:"endpoint"`
	Timeout  time.Duration `yaml:"timeout"`
}

// MQTTConfig contains MQTT broker settings
type MQTTConfig struct {
	Broker       string `yaml:"broker"`
	ClientID     string `yaml:"client_id"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	QoS          byte   `yaml:"qos"`
	CleanSession bool   `yaml:"clean_session"`
}

// TopicMap maps MQTT topics to sensor IDs (payload-based only)
type TopicMap struct {
	MQTTTopic           string `yaml:"mqtt_topic"`
	SensorIDFromPayload bool   `yaml:"sensor_id_from_payload"`
	Description         string `yaml:"description"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// LoadConfig reads and parses the YAML configuration file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.API.Timeout == 0 {
		config.API.Timeout = 10 * time.Second
	}
	if config.MQTT.ClientID == "" {
		config.MQTT.ClientID = "pmmnm-bridge"
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "text"
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.API.Endpoint == "" {
		return fmt.Errorf("api.endpoint is required")
	}
	if c.MQTT.Broker == "" {
		return fmt.Errorf("mqtt.broker is required")
	}
	if len(c.Topics) == 0 {
		return fmt.Errorf("at least one topic mapping is required")
	}
	return nil
}
