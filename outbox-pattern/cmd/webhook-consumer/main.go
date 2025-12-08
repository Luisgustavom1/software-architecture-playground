package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/software-architecture-playground/outbox-pattern/db"
)

type OrderFinishRequest struct {
	OrderID int64  `json:"order_id" binding:"required"`
	Status  string `json:"status" binding:"required"`
}

func main() {
	if err := db.Init(); err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	router := gin.Default()

	// Webhook endpoint to finish orders
	router.POST("/orders/finish", finishOrderHandler(db.DB))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	port := getEnv("PORT", "8082")
	log.Printf("webhook-consumer listening on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func finishOrderHandler(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req OrderFinishRequest

		// Parse request body
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("invalid request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("finishing order id=%d with status=%s", req.OrderID, req.Status)

		// Update order status in database
		result, err := dbConn.Exec(
			"UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2",
			req.Status,
			req.OrderID,
		)
		if err != nil {
			log.Printf("failed to update order: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update order: %v", err)})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("failed to get rows affected: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to verify update: %v", err)})
			return
		}

		if rowsAffected == 0 {
			log.Printf("order id=%d not found", req.OrderID)
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("order id=%d not found", req.OrderID)})
			return
		}

		log.Printf("order id=%d finished successfully", req.OrderID)
		c.JSON(http.StatusOK, gin.H{
			"message":   "order finished successfully",
			"order_id":  req.OrderID,
			"status":    req.Status,
		})
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
