package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

// SensorPayload represents the expected message format from ESP32
type SensorPayload struct {
	SensorID  string  `json:"sensorId,omitempty"` // Optional: for payload-based mode
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp,omitempty"`
}

// MQTTBridge handles MQTT connections and message routing
type MQTTBridge struct {
	config    *Config
	client    mqtt.Client
	apiClient *APIClient
	logger    *logrus.Logger
	topicMap  map[string]TopicMap
}

// NewMQTTBridge creates a new MQTT bridge instance
func NewMQTTBridge(config *Config, apiClient *APIClient, logger *logrus.Logger) *MQTTBridge {
	topicMap := make(map[string]TopicMap)
	for _, tm := range config.Topics {
		topicMap[tm.MQTTTopic] = tm
	}

	return &MQTTBridge{
		config:    config,
		apiClient: apiClient,
		logger:    logger,
		topicMap:  topicMap,
	}
}

// Connect establishes connection to MQTT broker and subscribes to topics
func (b *MQTTBridge) Connect() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(b.config.MQTT.Broker)
	opts.SetClientID(b.config.MQTT.ClientID)
	opts.SetCleanSession(b.config.MQTT.CleanSession)

	if b.config.MQTT.Username != "" {
		opts.SetUsername(b.config.MQTT.Username)
		opts.SetPassword(b.config.MQTT.Password)
	}

	opts.SetDefaultPublishHandler(b.messageHandler)
	opts.SetConnectionLostHandler(b.connectionLostHandler)
	opts.SetOnConnectHandler(b.onConnectHandler)

	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(1 * time.Minute)

	b.client = mqtt.NewClient(opts)

	b.logger.WithField("broker", b.config.MQTT.Broker).Info("Connecting to MQTT broker...")

	if token := b.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	b.logger.Info("Successfully connected to MQTT broker")
	return nil
}

// onConnectHandler is called when connection is established
func (b *MQTTBridge) onConnectHandler(client mqtt.Client) {
	b.logger.Info("Connected to MQTT broker, subscribing to topics...")

	for _, topic := range b.config.Topics {
		b.logger.WithFields(logrus.Fields{
			"topic":       topic.MQTTTopic,
			"description": topic.Description,
		}).Info("Subscribing to topic")

		token := client.Subscribe(topic.MQTTTopic, b.config.MQTT.QoS, nil)
		if token.Wait() && token.Error() != nil {
			b.logger.WithError(token.Error()).WithField("topic", topic.MQTTTopic).Error("Failed to subscribe")
		}
	}
}

// connectionLostHandler is called when connection is lost
func (b *MQTTBridge) connectionLostHandler(client mqtt.Client, err error) {
	b.logger.WithError(err).Warn("Connection to MQTT broker lost, will attempt to reconnect...")
}

// messageHandler processes incoming MQTT messages
func (b *MQTTBridge) messageHandler(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()

	logger := b.logger.WithFields(logrus.Fields{
		"topic": topic,
	})

	logger.Debug("Received MQTT message")

	var payload SensorPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		logger.WithError(err).WithField("payload", string(msg.Payload())).Error("Failed to parse JSON payload")
		return
	}

	sensorID, err := b.getSensorID(topic, &payload)
	if err != nil {
		logger.WithError(err).Error("Failed to determine sensor ID")
		return
	}

	logger = logger.WithField("sensor_id", sensorID)

	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().UnixMilli()
	}

	logger.WithFields(logrus.Fields{
		"value":     payload.Value,
		"timestamp": payload.Timestamp,
	}).Info("Processing sensor data")

	// Send to API
	if err := b.apiClient.SendSensorData(sensorID, payload.Value, payload.Timestamp); err != nil {
		logger.WithError(err).Error("Failed to send data to API")
		return
	}

	logger.Info("Successfully forwarded sensor data to API")
}

// getSensorID determines the sensor ID from payload
func (b *MQTTBridge) getSensorID(topic string, payload *SensorPayload) (string, error) {
		if tm, ok := b.topicMap[topic]; ok {
		if tm.SensorIDFromPayload {
			if payload.SensorID == "" {
				return "", fmt.Errorf("sensorId not found in payload")
			}
			return payload.SensorID, nil
		}
		return "", fmt.Errorf("topic matched but sensor_id_from_payload not enabled")
	}

	// Try wildcard matching
	for _, tm := range b.config.Topics {
		if b.matchTopic(tm.MQTTTopic, topic) {
			if tm.SensorIDFromPayload {
				if payload.SensorID == "" {
					return "", fmt.Errorf("sensorId not found in payload")
				}
				return payload.SensorID, nil
			}
			return "", fmt.Errorf("topic matched but sensor_id_from_payload not enabled")
		}
	}

	return "", fmt.Errorf("no mapping found for topic: %s", topic)
}

// matchTopic checks if a topic matches a pattern with wildcards
func (b *MQTTBridge) matchTopic(pattern, topic string) bool {
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")

	for i := range patternParts {
		if patternParts[i] == "#" {
			return true
		}
		if i >= len(topicParts) {
			return false
		}
		if patternParts[i] == "+" {
			continue
		}
		if patternParts[i] != topicParts[i] {
			return false
		}
	}

	return len(patternParts) == len(topicParts)
}

func (b *MQTTBridge) Disconnect() {
	if b.client != nil && b.client.IsConnected() {
		b.client.Disconnect(250)
		b.logger.Info("Disconnected from MQTT broker")
	}
}
