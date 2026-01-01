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
	ID          int64      `json:"id"`
	AggregateID string     `json:"aggregate_id"`
	Payload     string     `json:"payload"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
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

		log.Printf("222 %v 111", string(msg.Value))
		var outbox Outbox
		err = json.Unmarshal(msg.Value, &outbox)
		if err != nil {
			log.Printf("failed to unmarshal outbox message: %v", err)
			continue
		}

		log.Printf("publishing message for order %#v", outbox)
		time.Sleep(1 * time.Second)

		published_at := time.Now()
		_, err = db.DB.ExecContext(ctx, `UPDATE outbox SET status = 'published', published_at = ? WHERE id = ?`, published_at, outbox.ID)
		if err != nil {
			log.Printf("failed to update outbox %d: %v", outbox.ID, err)
			continue
		}
	}
}
