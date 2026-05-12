package events

import (
	"encoding/json"
	"time"
)

type EventEnvelope struct {
	ID            string          `json:"id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	SourceService string          `json:"source_service"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	Actor         *ActorContext   `json:"actor,omitempty"`
	SchemaVersion int             `json:"schema_version"`
}

type ActorContext struct {
	UserID    *string `json:"user_id,omitempty"`
	Email     *string `json:"email,omitempty"`
	IPAddress *string `json:"ip_address,omitempty"`
	RequestID string  `json:"request_id"`
}
