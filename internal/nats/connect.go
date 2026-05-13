package natsconn

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

// Connect creates a NATS connection and ensures the FIELDSTONE JetStream stream exists.
//
// InterestPolicy retains each message until every durable consumer has acknowledged it.
// This allows audit, webhooks, and extension services to each independently receive
// every event — WorkQueuePolicy would have delivered each message to only one subscriber.
func Connect(url string) (*nats.Conn, nats.JetStreamContext, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, nil, fmt.Errorf("connect nats: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("create jetstream context: %w", err)
	}

	info, err := js.StreamInfo("FIELDSTONE")
	if err != nil {
		// Stream does not exist — create it.
		if _, err := js.AddStream(&nats.StreamConfig{
			Name:      "FIELDSTONE",
			Subjects:  []string{"fieldstone.>"},
			Retention: nats.InterestPolicy,
		}); err != nil {
			nc.Close()
			return nil, nil, fmt.Errorf("create FIELDSTONE stream: %w", err)
		}
	} else if info.Config.Retention != nats.InterestPolicy {
		// Existing stream has wrong retention — update it.
		cfg := info.Config
		cfg.Retention = nats.InterestPolicy
		if _, err := js.UpdateStream(&cfg); err != nil {
			nc.Close()
			return nil, nil, fmt.Errorf("update FIELDSTONE stream retention: %w", err)
		}
	}

	return nc, js, nil
}
