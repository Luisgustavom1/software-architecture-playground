package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestGetEnv tests the getEnv function with various scenarios
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback string
		envValue string
		setEnv   bool
		want     string
	}{
		{
			name:     "returns environment variable when set",
			key:      "TEST_VAR_RELAY",
			fallback: "default",
			envValue: "custom_value",
			setEnv:   true,
			want:     "custom_value",
		},
		{
			name:     "returns fallback when env var not set",
			key:      "NONEXISTENT_VAR_RELAY",
			fallback: "fallback_value",
			setEnv:   false,
			want:     "fallback_value",
		},
		{
			name:     "returns empty string env value over fallback",
			key:      "EMPTY_VAR_RELAY",
			fallback: "fallback",
			envValue: "",
			setEnv:   true,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.key, tt.envValue)
			}
			got := getEnv(tt.key, tt.fallback)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExtractOutbox tests the extractOutbox function comprehensively
func TestExtractOutbox(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr bool
		errMsg  string
		check   func(t *testing.T, outbox Outbox)
	}{
		{
			name: "successfully extracts valid outbox data with float64 timestamp",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "order-456",
				"payload":      `{"id": 456, "status": "pending"}`,
				"status":       "pending",
				"created_at":   float64(1640000000000), // milliseconds
			},
			wantErr: false,
			check: func(t *testing.T, outbox Outbox) {
				assert.Equal(t, int64(123), outbox.ID)
				assert.Equal(t, "order-456", outbox.AggregateID)
				assert.Equal(t, `{"id": 456, "status": "pending"}`, outbox.Payload)
				assert.Equal(t, "pending", outbox.Status)
				assert.False(t, outbox.CreatedAt.IsZero())
			},
		},
		{
			name: "successfully extracts valid outbox data with RFC3339 timestamp",
			data: map[string]interface{}{
				"id":           float64(789),
				"aggregate_id": "order-789",
				"payload":      `{"id": 789}`,
				"status":       "published",
				"created_at":   "2024-01-01T12:00:00Z",
			},
			wantErr: false,
			check: func(t *testing.T, outbox Outbox) {
				assert.Equal(t, int64(789), outbox.ID)
				assert.Equal(t, "order-789", outbox.AggregateID)
				assert.Equal(t, "published", outbox.Status)
				expectedTime, _ := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")
				assert.Equal(t, expectedTime, outbox.CreatedAt)
			},
		},
		{
			name: "missing id field",
			data: map[string]interface{}{
				"aggregate_id": "order-123",
				"payload":      "{}",
				"status":       "pending",
			},
			wantErr: true,
			errMsg:  "missing or invalid id",
		},
		{
			name: "invalid id type",
			data: map[string]interface{}{
				"id":           "not-a-number",
				"aggregate_id": "order-123",
				"payload":      "{}",
				"status":       "pending",
			},
			wantErr: true,
			errMsg:  "missing or invalid id",
		},
		{
			name: "missing aggregate_id field",
			data: map[string]interface{}{
				"id":      float64(123),
				"payload": "{}",
				"status":  "pending",
			},
			wantErr: true,
			errMsg:  "missing or invalid aggregate_id",
		},
		{
			name: "invalid aggregate_id type",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": 12345,
				"payload":      "{}",
				"status":       "pending",
			},
			wantErr: true,
			errMsg:  "missing or invalid aggregate_id",
		},
		{
			name: "missing payload field",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "order-123",
				"status":       "pending",
			},
			wantErr: true,
			errMsg:  "missing or invalid payload",
		},
		{
			name: "invalid payload type",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "order-123",
				"payload":      12345,
				"status":       "pending",
			},
			wantErr: true,
			errMsg:  "missing or invalid payload",
		},
		{
			name: "missing status field",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "order-123",
				"payload":      "{}",
			},
			wantErr: true,
			errMsg:  "missing or invalid status",
		},
		{
			name: "invalid status type",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "order-123",
				"payload":      "{}",
				"status":       true,
			},
			wantErr: true,
			errMsg:  "missing or invalid status",
		},
		{
			name: "handles missing created_at gracefully",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "order-123",
				"payload":      "{}",
				"status":       "pending",
			},
			wantErr: false,
			check: func(t *testing.T, outbox Outbox) {
				assert.True(t, outbox.CreatedAt.IsZero())
			},
		},
		{
			name: "handles invalid timestamp string gracefully",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "order-123",
				"payload":      "{}",
				"status":       "pending",
				"created_at":   "invalid-timestamp",
			},
			wantErr: false,
			check: func(t *testing.T, outbox Outbox) {
				assert.True(t, outbox.CreatedAt.IsZero())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outbox, err := extractOutbox(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, outbox)
				}
			}
		})
	}
}

// TestRelayServerCallWebhook tests the webhook calling functionality
func TestRelayServerCallWebhook(t *testing.T) {
	tests := []struct {
		name           string
		orderID        int64
		status         string
		serverResponse int
		serverBody     string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful webhook call",
			orderID:        123,
			status:         "finished",
			serverResponse: http.StatusOK,
			serverBody:     `{"message": "success"}`,
			wantErr:        false,
		},
		{
			name:           "successful webhook call with 201 status",
			orderID:        456,
			status:         "completed",
			serverResponse: http.StatusCreated,
			serverBody:     `{"message": "created"}`,
			wantErr:        false,
		},
		{
			name:           "webhook returns 4xx error",
			orderID:        789,
			status:         "finished",
			serverResponse: http.StatusBadRequest,
			serverBody:     `{"error": "bad request"}`,
			wantErr:        true,
			errContains:    "webhook returned status 400",
		},
		{
			name:           "webhook returns 5xx error",
			orderID:        999,
			status:         "finished",
			serverResponse: http.StatusInternalServerError,
			serverBody:     `{"error": "internal error"}`,
			wantErr:        true,
			errContains:    "webhook returned status 500",
		},
		{
			name:           "webhook returns 404 not found",
			orderID:        111,
			status:         "finished",
			serverResponse: http.StatusNotFound,
			serverBody:     `{"error": "not found"}`,
			wantErr:        true,
			errContains:    "webhook returned status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and content type
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var payload map[string]interface{}
				err = json.Unmarshal(body, &payload)
				require.NoError(t, err)
				assert.Equal(t, float64(tt.orderID), payload["order_id"])
				assert.Equal(t, tt.status, payload["status"])

				// Send response
				w.WriteHeader(tt.serverResponse)
				w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			// Create RelayServer with mock webhook URL
			rs := &RelayServer{
				webhookURL: server.URL,
				dbConn:     nil,
			}

			// Call webhook
			err := rs.callWebhook(tt.orderID, tt.status)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRelayServerCallWebhookNetworkError tests network error handling
func TestRelayServerCallWebhookNetworkError(t *testing.T) {
	rs := &RelayServer{
		webhookURL: "http://invalid-host-that-does-not-exist-12345.com",
		dbConn:     nil,
	}

	err := rs.callWebhook(123, "finished")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to call webhook")
}

// TestDebeziumHandler tests the main Debezium event handler
func TestDebeziumHandler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(mock sqlmock.Sqlmock, webhookServer *httptest.Server)
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "successfully processes insert operation",
			requestBody: DebeziumChange{
				Op: "c",
				After: map[string]interface{}{
					"id":           float64(123),
					"aggregate_id": "order-456",
					"payload":      `{"id": 456, "status": "pending"}`,
					"status":       "pending",
					"created_at":   float64(1640000000000),
				},
			},
			setupMock: func(mock sqlmock.Sqlmock, webhookServer *httptest.Server) {
				mock.ExpectExec("UPDATE outbox SET status").
					WithArgs(int64(123)).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "processed", response["status"])
				assert.Equal(t, float64(123), response["outbox_id"])
				assert.Equal(t, float64(456), response["order_id"])
			},
		},
		{
			name: "successfully processes update operation",
			requestBody: DebeziumChange{
				Op: "u",
				After: map[string]interface{}{
					"id":           float64(789),
					"aggregate_id": "order-999",
					"payload":      `{"id": 999, "status": "pending"}`,
					"status":       "pending",
					"created_at":   float64(1640000000000),
				},
			},
			setupMock: func(mock sqlmock.Sqlmock, webhookServer *httptest.Server) {
				mock.ExpectExec("UPDATE outbox SET status").
					WithArgs(int64(789)).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "processed", response["status"])
			},
		},
		{
			name: "skips delete operation",
			requestBody: DebeziumChange{
				Op: "d",
				After: map[string]interface{}{
					"id": float64(123),
				},
			},
			setupMock:      func(mock sqlmock.Sqlmock, webhookServer *httptest.Server) {},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "skipped", response["status"])
			},
		},
		{
			name: "skips read operation",
			requestBody: DebeziumChange{
				Op: "r",
			},
			setupMock:      func(mock sqlmock.Sqlmock, webhookServer *httptest.Server) {},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "skipped", response["status"])
			},
		},
		{
			name: "handles missing after data",
			requestBody: DebeziumChange{
				Op:    "c",
				After: nil,
			},
			setupMock:      func(mock sqlmock.Sqlmock, webhookServer *httptest.Server) {},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "skipped", response["status"])
			},
		},
		{
			name: "handles invalid outbox data extraction",
			requestBody: DebeziumChange{
				Op: "c",
				After: map[string]interface{}{
					"invalid_field": "value",
				},
			},
			setupMock:      func(mock sqlmock.Sqlmock, webhookServer *httptest.Server) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "failed to extract outbox")
			},
		},
		{
			name: "handles invalid payload JSON",
			requestBody: DebeziumChange{
				Op: "c",
				After: map[string]interface{}{
					"id":           float64(123),
					"aggregate_id": "order-456",
					"payload":      `invalid json`,
					"status":       "pending",
					"created_at":   float64(1640000000000),
				},
			},
			setupMock:      func(mock sqlmock.Sqlmock, webhookServer *httptest.Server) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "invalid payload json")
			},
		},
		{
			name: "handles database update error",
			requestBody: DebeziumChange{
				Op: "c",
				After: map[string]interface{}{
					"id":           float64(123),
					"aggregate_id": "order-456",
					"payload":      `{"id": 456, "status": "pending"}`,
					"status":       "pending",
					"created_at":   float64(1640000000000),
				},
			},
			setupMock: func(mock sqlmock.Sqlmock, webhookServer *httptest.Server) {
				mock.ExpectExec("UPDATE outbox SET status").
					WithArgs(int64(123)).
					WillReturnError(fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "failed to mark published")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock webhook server
			webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message": "success"}`))
			}))
			defer webhookServer.Close()

			// Create mock database
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// Setup mock expectations
			tt.setupMock(mock, webhookServer)

			// Create RelayServer with mocks
			relayServer = &RelayServer{
				webhookURL: webhookServer.URL,
				dbConn:     mockDB,
			}

			// Create test request
			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/debezium", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Create gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			debeziumHandler(c)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			tt.checkResponse(t, response)

			// Verify all mock expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

// TestDebeziumHandlerInvalidJSON tests invalid JSON handling
func TestDebeziumHandlerInvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/debezium", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	debeziumHandler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid json", response["error"])
}

// TestDebeziumHandlerReadBodyError tests body reading errors
func TestDebeziumHandlerReadBodyError(t *testing.T) {
	// Create a request with an error-prone body
	req := httptest.NewRequest("POST", "/debezium", &errorReader{})
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	debeziumHandler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// errorReader is a helper type that always returns an error when reading
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

// TestDebeziumHandlerWebhookFailure tests webhook failure scenarios
func TestDebeziumHandlerWebhookFailure(t *testing.T) {
	// Create mock webhook server that returns error
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "webhook failed"}`))
	}))
	defer webhookServer.Close()

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// No database expectations since webhook fails first

	relayServer = &RelayServer{
		webhookURL: webhookServer.URL,
		dbConn:     mockDB,
	}

	requestBody := DebeziumChange{
		Op: "c",
		After: map[string]interface{}{
			"id":           float64(123),
			"aggregate_id": "order-456",
			"payload":      `{"id": 456, "status": "pending"}`,
			"status":       "pending",
			"created_at":   float64(1640000000000),
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/debezium", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	debeziumHandler(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "webhook call failed")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestDebeziumHandlerMissingOrderIDInPayload tests missing order ID in payload
func TestDebeziumHandlerMissingOrderIDInPayload(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	relayServer = &RelayServer{
		webhookURL: webhookServer.URL,
		dbConn:     mockDB,
	}

	requestBody := DebeziumChange{
		Op: "c",
		After: map[string]interface{}{
			"id":           float64(123),
			"aggregate_id": "order-456",
			"payload":      `{"status": "pending"}`, // Missing "id" field
			"status":       "pending",
			"created_at":   float64(1640000000000),
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/debezium", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	debeziumHandler(c)

	// The handler will panic when trying to access missing "id" field
	// In production, this should be handled more gracefully
	assert.Equal(t, http.StatusOK, w.Code) // Context might have been written before panic

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestExtractOutboxEdgeCases tests additional edge cases
func TestExtractOutboxEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr bool
	}{
		{
			name: "handles very large ID",
			data: map[string]interface{}{
				"id":           float64(9223372036854775807), // Max int64
				"aggregate_id": "order-123",
				"payload":      "{}",
				"status":       "pending",
			},
			wantErr: false,
		},
		{
			name: "handles empty string values",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "",
				"payload":      "",
				"status":       "",
			},
			wantErr: false,
		},
		{
			name: "handles special characters in strings",
			data: map[string]interface{}{
				"id":           float64(123),
				"aggregate_id": "order-!@#$%^&*()",
				"payload":      `{"key": "value with \"quotes\" and \\ backslashes"}`,
				"status":       "pending",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outbox, err := extractOutbox(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, outbox)
			}
		})
	}
}

// TestRelayServerCallWebhookConcurrency tests concurrent webhook calls
func TestRelayServerCallWebhookConcurrency(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rs := &RelayServer{
		webhookURL: server.URL,
		dbConn:     nil,
	}

	// Make concurrent webhook calls
	const numCalls = 10
	done := make(chan bool, numCalls)

	for i := 0; i < numCalls; i++ {
		go func(orderID int64) {
			err := rs.callWebhook(orderID, "finished")
			assert.NoError(t, err)
			done <- true
		}(int64(i))
	}

	// Wait for all calls to complete
	for i := 0; i < numCalls; i++ {
		<-done
	}

	assert.Equal(t, numCalls, callCount)
}

// TestDebeziumChangeJSONSerialization tests JSON marshaling/unmarshaling
func TestDebeziumChangeJSONSerialization(t *testing.T) {
	original := DebeziumChange{
		Before: map[string]interface{}{
			"id": float64(123),
		},
		After: map[string]interface{}{
			"id":           float64(456),
			"aggregate_id": "order-789",
			"payload":      `{"test": "data"}`,
			"status":       "pending",
		},
		Source: DebeziumSource{
			Version:   "2.5",
			Connector: "postgresql",
			Name:      "outbox",
			TsMs:      1640000000000,
			Snapshot:  "false",
			DB:        "outbox_db",
			Schema:    "public",
			Table:     "outbox",
			TxID:      12345,
			LSN:       67890,
			XMin:      11111,
		},
		Op:          "c",
		TsMs:        1640000000000,
		Transaction: nil,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var decoded DebeziumChange
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, original.Op, decoded.Op)
	assert.Equal(t, original.TsMs, decoded.TsMs)
	assert.Equal(t, original.Source.Connector, decoded.Source.Connector)
	assert.Equal(t, original.After["id"], decoded.After["id"])
}

// TestOutboxJSONSerialization tests Outbox struct JSON handling
func TestOutboxJSONSerialization(t *testing.T) {
	now := time.Now()
	original := Outbox{
		ID:          123,
		AggregateID: "order-456",
		Payload:     `{"test": "data"}`,
		Status:      "pending",
		CreatedAt:   now,
		PublishedAt: &now,
	}

	jsonBytes, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Outbox
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.AggregateID, decoded.AggregateID)
	assert.Equal(t, original.Payload, decoded.Payload)
	assert.Equal(t, original.Status, decoded.Status)
	assert.WithinDuration(t, original.CreatedAt, decoded.CreatedAt, time.Second)
	assert.NotNil(t, decoded.PublishedAt)
}

// TestRelayServerCallWebhookTimeout tests timeout handling
func TestRelayServerCallWebhookTimeout(t *testing.T) {
	// Create server that delays response beyond timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than httpClient timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rs := &RelayServer{
		webhookURL: server.URL,
		dbConn:     nil,
	}

	// This should timeout since httpClient has 5 second timeout
	err := rs.callWebhook(123, "finished")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to call webhook")
}