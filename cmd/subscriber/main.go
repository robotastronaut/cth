package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
)

type config struct {
	SubId        string `env:"PUBSUB_SUBSCRIPTION_ID"`
	ProjectId    string `env:"PUBSUB_PROJECT_ID"`
	DatabaseUser string `env:"POSTGRES_USER"`
	DatabasePass string `env:"POSTGRES_PASSWORD"`
	DatabaseName string `env:"POSTGRES_DB"`
	DatabaseHost string `env:"POSTGRES_HOST"`
	DatabasePort string `env:"POSTGRES_PORT"`
}

// Normally I would spend time abstracting parts of this, but since I'm limited on time and I'm interested in performance,
// I'm going to just keep most of this in main. Given more time, I'd break it out so that each functional element was testable
// with dependency injection.
func main() {
	cfg, err := env.ParseAs[config]()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Just use background context for now
	ctx := context.Background()

	// create the pubsub client
	client, err := pubsub.NewClient(ctx, cfg.ProjectId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating pubsub client: %v\n", err)
		os.Exit(1)
	}
	// defer the close
	defer client.Close()

	// grab subscription
	sub := client.Subscription(cfg.SubId)

	// get the database pool
	pool, err := getDB(cfg.DatabaseUser, cfg.DatabasePass, cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// By default, this runs in streaming pull mode, it maintains 10 connections.
	// The default number of connections in the connection pool is  `min(4,GOMAXPROCS)`, and each connection
	// can handle 100 streams. `ReceiveSettings` can be used to configure this, but considering
	// a StreamingPull connection can handle up to 10 MB/s (reportedly), we're more likely to be
	// bound by IO with the database than the small default number of connections. Increasing the number
	// could make it more likely that we're IO bound in general.
	//
	// Evaluate the performance against the pgx connection pool, which has max connections of `max(4, runtime.NumCPU())`
	err = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {

		// This should only error in true error conditions, like a network error or a message that can't be unmarshalled
		if err := insertMessage(pool, msg.Data); err != nil {
			// Nack this, we can't process it
			msg.Nack()
			return
		}

		// Acks after the first are a no-op, according to the documentation, so there shouldn't be an issue with calling it on a message
		// that was acked by another instance of this subscriber service. While it's generally true that a message is only sent to one
		// client on a subscription, it's not guaranteed, and it's possible that a message could be pulled by multiple StreamingPull clients.
		msg.Ack()
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func insertMessage(pool *pgxpool.Pool, data []byte) error {
	var scan Scan
	err := json.Unmarshal(data, &scan)

	if err != nil {
		return fmt.Errorf("error unmarshalling message: %v", err)
	}

	// This is a pretty straightforward insert. Generally, I'd abstract the storage layer, but interfaces aren't without
	// cost and I'm going for simplicity and performance. Postgres, in its default configuration, has implicit transactions,
	// and the default transaction isolation level is `read committed`, so we need to make sure to use the `on conflict do update`
	// pattern to avoid conditions where we have multiple messages attempting to update the same row at the same time. This makes sure
	// that any previous transactions (including initial inserts) have been committed before we attempt to update. It's not perfect,
	// and it's not universal, but for this exercise, it works.
	_, err = pool.Exec(context.Background(), `
			INSERT INTO 
				scans (ip, port, service, timestamp, data) 
			VALUES 
				($1, $2, $3, $4, $5)
			ON CONFLICT (ip, port, service) DO UPDATE 
				SET timestamp = $4, data = $5
				WHERE scans.timestamp < $4
		`, scan.Ip, scan.Port, scan.Service, scan.Timestamp, scan.Data)

	return err
}

func getDB(user, password, host, port, dbName string) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbName)
	return pgxpool.New(context.Background(), connStr)
}
