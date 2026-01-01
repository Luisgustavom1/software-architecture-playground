package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/software-architecture-playground/outbox-pattern/db"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Outbox struct {
	ID          int64   `json:"id"`
	AggregateID string  `json:"aggregate_id"`
	Payload     string  `json:"payload"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	PublishedAt *string `json:"published_at"`
}

type DebeziumSource struct {
	Version   string  `json:"version"`
	Connector string  `json:"connector"`
	Name      string  `json:"name"`
	TsMs      int64   `json:"ts_ms"`
	Snapshot  string  `json:"snapshot"`
	Db        string  `json:"db"`
	Sequence  *string `json:"sequence"`
	Table     string  `json:"table"`
	ServerID  int64   `json:"server_id"`
	Gtid      *string `json:"gtid"`
	File      string  `json:"file"`
	Pos       int64   `json:"pos"`
	Row       int32   `json:"row"`
	Thread    *int64  `json:"thread"`
	Query     *string `json:"query"`
}

type DebeziumTransaction struct {
	ID                  string `json:"id"`
	TotalOrder          int64  `json:"total_order"`
	DataCollectionOrder int64  `json:"data_collection_order"`
}

type DebeziumPayload struct {
	Before      *Outbox              `json:"before"`
	After       *Outbox              `json:"after"`
	Source      DebeziumSource       `json:"source"`
	Op          string               `json:"op"`
	TsMs        *int64               `json:"ts_ms"`
	Transaction *DebeziumTransaction `json:"transaction"`
}

type DebeziumCDCMessage struct {
	Schema  interface{}     `json:"schema"`
	Payload DebeziumPayload `json:"payload"`
}

func main() {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "outbox-relay-group",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		panic(err)
	}

	err = c.SubscribeTopics([]string{"cdc.outbox_db.outbox"}, nil)

	if err != nil {
		panic(err)
	}

	db.Init()
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		c.Close()
	}()

	for {
		log.Printf("getting outbox records to process")
		msg, err := c.ReadMessage(time.Second)
		if err != nil && !err.(kafka.Error).IsTimeout() {
			// The client will automatically try to recover from all errors.
			// Timeout is not considered an error because it is raised by
			// ReadMessage in absence of messages.
			log.Printf("Consumer error: %v (%v)\n", err, msg)
			continue
		}

		if msg == nil {
			continue
		}

		var cdcMessage DebeziumCDCMessage
		err = json.Unmarshal(msg.Value, &cdcMessage)
		if err != nil {
			log.Printf("failed to unmarshal CDC message: %v", err)
			log.Printf("raw message: %s", string(msg.Value))
			continue
		}

		var outbox *Outbox
		if cdcMessage.Payload.Op == "c" {
			outbox = cdcMessage.Payload.After
		}

		if outbox == nil {
			log.Printf("no outbox data found in CDC message for operation: %s", cdcMessage.Payload.Op)
			continue
		}

		log.Printf("CDC Operation: %s, Order ID: %d, Aggregate: %s",
			cdcMessage.Payload.Op, outbox.ID, outbox.AggregateID)
		log.Printf("Outbox payload (business event): %s", outbox.Payload)

		time.Sleep(1 * time.Second)

		published_at := time.Now().Format(time.RFC3339)
		_, err = db.DB.ExecContext(ctx, `UPDATE outbox SET status = 'published', published_at = ? WHERE id = ?`, published_at, outbox.ID)
		if err != nil {
			log.Printf("failed to update outbox %d: %v", outbox.ID, err)
			continue
		}
	}
}
