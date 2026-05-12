package natsconn

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

// Connect creates a NATS connection and ensures the FIELDSTONE JetStream stream exists.
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

	if _, err := js.StreamInfo("FIELDSTONE"); err != nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name:      "FIELDSTONE",
			Subjects:  []string{"fieldstone.>"},
			Retention: nats.WorkQueuePolicy,
		}); err != nil {
			nc.Close()
			return nil, nil, fmt.Errorf("create FIELDSTONE stream: %w", err)
		}
	}

	return nc, js, nil
}
