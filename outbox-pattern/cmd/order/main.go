package main

import (
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/software-architecture-playground/outbox-pattern/db"
)

type Order struct {
	ID          int64   `json:"id"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"status"`
}

func main() {
	db.Init()
	defer db.Close()

	router := gin.Default()
	router.POST("/orders", createOrderHandler(db.DB))
	router.Run(":8080")
}

func createOrderHandler(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		rndAmount := rand.Float64() * 100
		order := Order{
			TotalAmount: rndAmount,
			Status:      "pending",
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result, err := tx.Exec("INSERT INTO orders (total_amount, status) VALUES (?, ?)", order.TotalAmount, order.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		order.ID, err = result.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		payload, err := json.Marshal(order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = tx.Exec("INSERT INTO outbox (aggregate_id, payload, status) VALUES (?, ?, ?)", order.ID, payload, "pending")
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		err = tx.Commit()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, order)
	}
}
