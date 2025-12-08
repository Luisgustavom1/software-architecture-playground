package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		want         string
	}{
		{
			name:         "returns environment variable when set",
			key:          "TEST_VAR_WEBHOOK",
			defaultValue: "default",
			envValue:     "from_env",
			setEnv:       true,
			want:         "from_env",
		},
		{
			name:         "returns default when env var not set",
			key:          "NONEXISTENT_VAR_WEBHOOK",
			defaultValue: "default_value",
			setEnv:       false,
			want:         "default_value",
		},
		{
			name:         "returns empty string when env is empty",
			key:          "EMPTY_VAR_WEBHOOK",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			want:         "",
		},
		{
			name:         "handles whitespace in env value",
			key:          "WHITESPACE_VAR",
			defaultValue: "default",
			envValue:     "  value with spaces  ",
			setEnv:       true,
			want:         "  value with spaces  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.key, tt.envValue)
			}
			got := getEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFinishOrderHandler tests the finish order handler with various scenarios
func TestFinishOrderHandler(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(mock sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "successfully finishes order",
			requestBody: OrderFinishRequest{
				OrderID: 123,
				Status:  "finished",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("finished", int64(123)).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "order finished successfully", response["message"])
				assert.Equal(t, float64(123), response["order_id"])
				assert.Equal(t, "finished", response["status"])
			},
		},
		{
			name: "successfully updates order with different status",
			requestBody: OrderFinishRequest{
				OrderID: 456,
				Status:  "completed",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("completed", int64(456)).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "order finished successfully", response["message"])
				assert.Equal(t, float64(456), response["order_id"])
				assert.Equal(t, "completed", response["status"])
			},
		},
		{
			name: "handles order not found",
			requestBody: OrderFinishRequest{
				OrderID: 999,
				Status:  "finished",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("finished", int64(999)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "order id=999 not found")
			},
		},
		{
			name: "handles database update error",
			requestBody: OrderFinishRequest{
				OrderID: 789,
				Status:  "finished",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("finished", int64(789)).
					WillReturnError(fmt.Errorf("database connection lost"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "failed to update order")
			},
		},
		{
			name:           "handles missing order_id in request",
			requestBody:    map[string]interface{}{"status": "finished"},
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "required")
			},
		},
		{
			name:           "handles missing status in request",
			requestBody:    map[string]interface{}{"order_id": 123},
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "required")
			},
		},
		{
			name:           "handles invalid request body type",
			requestBody:    "invalid json string",
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.NotNil(t, response["error"])
			},
		},
		{
			name: "handles negative order ID",
			requestBody: OrderFinishRequest{
				OrderID: -1,
				Status:  "finished",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("finished", int64(-1)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "not found")
			},
		},
		{
			name: "handles zero order ID",
			requestBody: OrderFinishRequest{
				OrderID: 0,
				Status:  "finished",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("finished", int64(0)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response["error"], "not found")
			},
		},
		{
			name: "handles very large order ID",
			requestBody: OrderFinishRequest{
				OrderID: 9223372036854775807, // Max int64
				Status:  "finished",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("finished", int64(9223372036854775807)).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "order finished successfully", response["message"])
			},
		},
		{
			name: "handles empty status string",
			requestBody: OrderFinishRequest{
				OrderID: 123,
				Status:  "",
			},
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.NotNil(t, response["error"])
			},
		},
		{
			name: "handles special characters in status",
			requestBody: OrderFinishRequest{
				OrderID: 123,
				Status:  "finished!@#$%",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("finished!@#$%", int64(123)).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "order finished successfully", response["message"])
				assert.Equal(t, "finished!@#$%", response["status"])
			},
		},
		{
			name: "handles long status string",
			requestBody: OrderFinishRequest{
				OrderID: 123,
				Status:  string(make([]byte, 1000)),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs(string(make([]byte, 1000)), int64(123)).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "order finished successfully", response["message"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// Setup mock expectations
			tt.setupMock(mock)

			// Create test request
			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Create gin context and router
			router := gin.New()
			router.POST("/orders/finish", finishOrderHandler(mockDB))
			router.ServeHTTP(w, req)

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

// TestFinishOrderHandlerInvalidJSON tests invalid JSON handling
func TestFinishOrderHandlerInvalidJSON(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/orders/finish", finishOrderHandler(mockDB))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response["error"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestFinishOrderHandlerEmptyBody tests empty request body
func TestFinishOrderHandlerEmptyBody(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/orders/finish", finishOrderHandler(mockDB))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestFinishOrderHandlerRowsAffectedError tests RowsAffected error handling
func TestFinishOrderHandlerRowsAffectedError(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// Mock RowsAffected to return error
	mock.ExpectExec("UPDATE orders SET status").
		WithArgs("finished", int64(123)).
		WillReturnResult(sqlmock.NewErrorResult(fmt.Errorf("rows affected error")))

	requestBody := OrderFinishRequest{
		OrderID: 123,
		Status:  "finished",
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/orders/finish", finishOrderHandler(mockDB))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "failed to verify update")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestOrderFinishRequestValidation tests request struct validation
func TestOrderFinishRequestValidation(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		shouldError bool
	}{
		{
			name:        "valid request",
			jsonInput:   `{"order_id": 123, "status": "finished"}`,
			shouldError: false,
		},
		{
			name:        "missing order_id",
			jsonInput:   `{"status": "finished"}`,
			shouldError: true,
		},
		{
			name:        "missing status",
			jsonInput:   `{"order_id": 123}`,
			shouldError: true,
		},
		{
			name:        "order_id as string",
			jsonInput:   `{"order_id": "123", "status": "finished"}`,
			shouldError: true,
		},
		{
			name:        "status as number",
			jsonInput:   `{"order_id": 123, "status": 123}`,
			shouldError: true,
		},
		{
			name:        "extra fields ignored",
			jsonInput:   `{"order_id": 123, "status": "finished", "extra": "field"}`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			if !tt.shouldError {
				mock.ExpectExec("UPDATE orders SET status").
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader([]byte(tt.jsonInput)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/orders/finish", finishOrderHandler(mockDB))
			router.ServeHTTP(w, req)

			if tt.shouldError {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}

			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

// TestFinishOrderHandlerConcurrency tests concurrent requests
func TestFinishOrderHandlerConcurrency(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	const numRequests = 10

	// Expect multiple updates
	for i := 0; i < numRequests; i++ {
		mock.ExpectExec("UPDATE orders SET status").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	router := gin.New()
	router.POST("/orders/finish", finishOrderHandler(mockDB))

	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(orderID int64) {
			requestBody := OrderFinishRequest{
				OrderID: orderID,
				Status:  "finished",
			}

			bodyBytes, _ := json.Marshal(requestBody)
			req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}(int64(i))
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestOrderFinishRequestJSONSerialization tests JSON marshaling/unmarshaling
func TestOrderFinishRequestJSONSerialization(t *testing.T) {
	original := OrderFinishRequest{
		OrderID: 12345,
		Status:  "completed",
	}

	jsonBytes, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded OrderFinishRequest
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.OrderID, decoded.OrderID)
	assert.Equal(t, original.Status, decoded.Status)
}

// TestFinishOrderHandlerDifferentContentTypes tests various content types
func TestFinishOrderHandlerDifferentContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		shouldWork  bool
	}{
		{
			name:        "application/json",
			contentType: "application/json",
			shouldWork:  true,
		},
		{
			name:        "application/json with charset",
			contentType: "application/json; charset=utf-8",
			shouldWork:  true,
		},
		{
			name:        "no content type",
			contentType: "",
			shouldWork:  true, // Gin might still parse it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			if tt.shouldWork {
				mock.ExpectExec("UPDATE orders SET status").
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			requestBody := OrderFinishRequest{
				OrderID: 123,
				Status:  "finished",
			}

			bodyBytes, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader(bodyBytes))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/orders/finish", finishOrderHandler(mockDB))
			router.ServeHTTP(w, req)

			if tt.shouldWork {
				assert.Equal(t, http.StatusOK, w.Code)
			}

			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

// TestFinishOrderHandlerNilDatabase tests nil database handling
func TestFinishOrderHandlerNilDatabase(t *testing.T) {
	requestBody := OrderFinishRequest{
		OrderID: 123,
		Status:  "finished",
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/orders/finish", finishOrderHandler(nil))
	router.ServeHTTP(w, req)

	// Should get internal server error when trying to use nil DB
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestFinishOrderHandlerMultipleRowsAffected tests when multiple rows are affected
func TestFinishOrderHandlerMultipleRowsAffected(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// Simulate updating multiple rows (shouldn't happen with WHERE id = $1, but test anyway)
	mock.ExpectExec("UPDATE orders SET status").
		WithArgs("finished", int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 2))

	requestBody := OrderFinishRequest{
		OrderID: 123,
		Status:  "finished",
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/orders/finish", finishOrderHandler(mockDB))
	router.ServeHTTP(w, req)

	// Should still return OK since at least one row was affected
	assert.Equal(t, http.StatusOK, w.Code)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// BenchmarkFinishOrderHandler benchmarks the finish order handler
func BenchmarkFinishOrderHandler(b *testing.B) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer mockDB.Close()

	for i := 0; i < b.N; i++ {
		mock.ExpectExec("UPDATE orders SET status").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	router := gin.New()
	router.POST("/orders/finish", finishOrderHandler(mockDB))

	requestBody := OrderFinishRequest{
		OrderID: 123,
		Status:  "finished",
	}

	bodyBytes, _ := json.Marshal(requestBody)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/orders/finish", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
	}
}