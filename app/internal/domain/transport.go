package domain

import "context"

// TransportClient talks to the transport service (:8081).
type TransportClient interface {
	Send(ctx context.Context, payload map[string]any) (SendAccepted, error)
	Receive(ctx context.Context, query ReceiveQuery) (ReceiveResult, error)
	Health(ctx context.Context) (bool, error)
}

type SendAccepted struct {
	MessageID string `json:"message_id"`
}

type ReceiveQuery struct {
	Sender string `json:"sender,omitempty"`
	WaitMs int    `json:"wait_ms,omitempty"`
}

type ReceiveResult struct {
	Sender  string `json:"sender"`
	SentAt  string `json:"sent_at"`
	Error   bool   `json:"error"`
	Payload string `json:"payload"`
}

type GenerateRequest struct {
	Sender      string
	SentAt      string
	JSONSchema  map[string]any
	SampleCount int
	Constraints string
}
