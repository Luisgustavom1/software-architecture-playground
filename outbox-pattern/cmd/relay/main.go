package main

import (
	"context"
	"log"
	"time"

	"github.com/software-architecture-playground/outbox-pattern/db"
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
	db.Init()
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()

	for {
		log.Printf("getting outbox records to process")
		rows, err := db.DB.QueryContext(ctx, "SELECT * FROM outbox WHERE status = 'pending' ORDER BY created_at ASC")
		if err != nil {
			log.Printf("failed to query outbox: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		var outboxes []Outbox

		for rows.Next() {
			var outbox Outbox
			err = rows.Scan(&outbox.ID, &outbox.AggregateID, &outbox.Payload, &outbox.Status, &outbox.CreatedAt, &outbox.PublishedAt)
			if err != nil {
				log.Printf("failed to scan row: %v", err)
				continue
			}
			outboxes = append(outboxes, outbox)
		}

		rows.Close()

		for i := range outboxes {
			outbox := outboxes[i]

			// publishing message
			log.Printf("publishing message for order %#v", outbox)
			time.Sleep(1 * time.Second)

			published_at := time.Now()
			_, err = db.DB.ExecContext(ctx, `UPDATE outbox SET status = 'published', published_at = ? WHERE id = ?`, published_at, outbox.ID)
			if err != nil {
				log.Printf("failed to update outbox %d: %v", outbox.ID, err)
				continue
			}
		}

		time.Sleep(1 * time.Second)
	}
}
