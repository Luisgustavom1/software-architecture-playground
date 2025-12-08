package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/software-architecture-playground/outbox-pattern/db"
)

type DebeziumChange struct {
	Before       map[string]interface{} `json:"before"`
	After        map[string]interface{} `json:"after"`
	Source       DebeziumSource         `json:"source"`
	Op           string                 `json:"op"`
	TsMs         int64                  `json:"ts_ms"`
	Transaction  interface{}            `json:"transaction"`
}

type DebeziumSource struct {
	Version     string `json:"version"`
	Connector   string `json:"connector"`
	Name        string `json:"name"`
	TsMs        int64  `json:"ts_ms"`
	Snapshot    string `json:"snapshot"`
	DB          string `json:"db"`
	Schema      string `json:"schema"`
	Table       string `json:"table"`
	TxID        int64  `json:"txId"`
	LSN         int64  `json:"lsn"`
	XMin        int64  `json:"xmin"`
}

type Outbox struct {
	ID          int64      `json:"id"`
	AggregateID string     `json:"aggregate_id"`
	Payload     string     `json:"payload"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type RelayServer struct {
	webhookURL string
	dbConn     *sql.DB
	mu         sync.Mutex
}

var (
	relayServer *RelayServer
	httpClient  = &http.Client{Timeout: 5 * time.Second}
)

func main() {
	if err := db.Init(); err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	webhookURL := getEnv("WEBHOOK_URL", "http://webhook-consumer:8082/orders/finish")

	relayServer = &RelayServer{
		webhookURL: webhookURL,
		dbConn:     db.DB,
		mu:         sync.Mutex{},
	}

	router := gin.Default()

	// Endpoint to receive Debezium CDC events
	router.POST("/debezium", debeziumHandler)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	port := getEnv("PORT", "8081")
	log.Printf("relay server listening on port %s", port)
	log.Printf("webhook URL configured as: %s", webhookURL)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func debeziumHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	log.Printf("received debezium event: %s", string(body))

	var change DebeziumChange
	if err := json.Unmarshal(body, &change); err != nil {
		log.Printf("failed to unmarshal debezium change: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	// Only process insert operations on outbox table
	if change.Op != "c" && change.Op != "u" {
		log.Printf("skipping non-insert/update operation: %s", change.Op)
		c.JSON(http.StatusOK, gin.H{"status": "skipped"})
		return
	}

	if change.After == nil {
		log.Printf("no after data in change, skipping")
		c.JSON(http.StatusOK, gin.H{"status": "skipped"})
		return
	}

	// Extract outbox data from change.After
	outbox, err := extractOutbox(change.After)
	if err != nil {
		log.Printf("failed to extract outbox: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to extract outbox: %v", err)})
		return
	}

	log.Printf("[cdc] processing outbox event id=%d aggregate=%s", outbox.ID, outbox.AggregateID)

	// Parse payload to get order ID
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(outbox.Payload), &payload); err != nil {
		log.Printf("failed to parse payload json: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload json"})
		return
	}

	orderID := int64(payload["id"].(float64))

	// Call webhook to finish order
	if err := relayServer.callWebhook(orderID, "finished"); err != nil {
		log.Printf("webhook call failed for outbox %d: %v", outbox.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("webhook call failed: %v", err)})
		return
	}

	// Mark outbox as published
	ctx := context.Background()
	_, err = relayServer.dbConn.ExecContext(ctx,
		`UPDATE outbox SET status = 'published', published_at = NOW() WHERE id = $1`,
		outbox.ID,
	)
	if err != nil {
		log.Printf("failed to mark outbox %d published: %v", outbox.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to mark published: %v", err)})
		return
	}

	log.Printf("outbox %d marked as published", outbox.ID)
	c.JSON(http.StatusOK, gin.H{
		"status":    "processed",
		"outbox_id": outbox.ID,
		"order_id":  orderID,
	})
}

func (rs *RelayServer) callWebhook(orderID int64, status string) error {
	payload := map[string]interface{}{
		"order_id": orderID,
		"status":   status,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", rs.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("webhook call successful for order %d", orderID)
	return nil
}

func extractOutbox(data map[string]interface{}) (Outbox, error) {
	outbox := Outbox{}

	// Extract ID
	if id, ok := data["id"].(float64); ok {
		outbox.ID = int64(id)
	} else {
		return outbox, fmt.Errorf("missing or invalid id")
	}

	// Extract aggregate_id
	if aggID, ok := data["aggregate_id"].(string); ok {
		outbox.AggregateID = aggID
	} else {
		return outbox, fmt.Errorf("missing or invalid aggregate_id")
	}

	// Extract payload
	if payload, ok := data["payload"].(string); ok {
		outbox.Payload = payload
	} else {
		return outbox, fmt.Errorf("missing or invalid payload")
	}

	// Extract status
	if status, ok := data["status"].(string); ok {
		outbox.Status = status
	} else {
		return outbox, fmt.Errorf("missing or invalid status")
	}

	// Extract created_at (Debezium sends as epoch milliseconds or string)
	if createdAt, ok := data["created_at"].(float64); ok {
		outbox.CreatedAt = time.Unix(0, int64(createdAt)*int64(time.Millisecond))
	} else if createdAtStr, ok := data["created_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			outbox.CreatedAt = parsed
		}
	}

	return outbox, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

