package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// APIClient handles HTTP requests to the sensor data API
type APIClient struct {
	endpoint   string
	httpClient *http.Client
	logger     *logrus.Logger
}

// SensorDataRequest represents the API request body
type SensorDataRequest struct {
	SensorID  string  `json:"sensorId"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp,omitempty"`
}

// SensorDataResponse represents the API success response
type SensorDataResponse struct {
	Success           bool   `json:"success"`
	DataID            string `json:"dataId"`
	ThresholdExceeded bool   `json:"thresholdExceeded"`
	Automation        struct {
		RulesChecked   int      `json:"rulesChecked"`
		RulesTriggered int      `json:"rulesTriggered"`
		ZonesCreated   []string `json:"zonesCreated"`
		Message        string   `json:"message"`
	} `json:"automation"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Warning string `json:"warning,omitempty"`
}

// NewAPIClient creates a new API client
func NewAPIClient(endpoint string, timeout time.Duration, logger *logrus.Logger) *APIClient {
	return &APIClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// SendSensorData sends sensor data to the API endpoint
func (c *APIClient) SendSensorData(sensorID string, value float64, timestamp int64) error {
	req := SensorDataRequest{
		SensorID:  sensorID,
		Value:     value,
		Timestamp: timestamp,
	}

	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	c.logger.WithFields(logrus.Fields{
		"endpoint":  c.endpoint,
		"sensor_id": sensorID,
		"value":     value,
	}).Debug("Sending request to API")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Handle different status codes
	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		var apiResp SensorDataResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			c.logger.WithError(err).Warn("Failed to parse success response")
		} else {
			c.logger.WithFields(logrus.Fields{
				"data_id":            apiResp.DataID,
				"threshold_exceeded": apiResp.ThresholdExceeded,
				"rules_triggered":    apiResp.Automation.RulesTriggered,
				"zones_created":      len(apiResp.Automation.ZonesCreated),
			}).Info("Sensor data processed successfully")

			if apiResp.ThresholdExceeded {
				c.logger.WithFields(logrus.Fields{
					"sensor_id":     sensorID,
					"zones_created": apiResp.Automation.ZonesCreated,
				}).Warn("Threshold exceeded - automation triggered")
			}
		}
		return nil

	case http.StatusAccepted:
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			c.logger.WithField("warning", errResp.Warning).Warn("API warning")
		}
		return nil

	case http.StatusBadRequest:
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return fmt.Errorf("bad request: %s", errResp.Error)
		}
		return fmt.Errorf("bad request: %s", string(respBody))

	default:
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}
}
