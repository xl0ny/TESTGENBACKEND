package models

import "time"

type Segment struct {
	MessageID     string `json:"message_id"`
	SentAt        string `json:"sent_at"`
	TotalSegments int    `json:"total_segments"`
	SegmentIndex  int    `json:"segment_index"`
	PayloadChunk  string `json:"payload_chunk"`
}

type SendPayload struct {
	Sender       string                 `json:"sender"`
	SentAt       string                 `json:"sent_at"`
	JSONSchema   map[string]interface{} `json:"json_schema"`
	SampleCount  int                    `json:"sample_count"`
	Constraints  string                 `json:"constraints,omitempty"`
}

type SendAccepted struct {
	MessageID string `json:"message_id"`
}

type ReceiveQuery struct {
	Sender string `json:"sender,omitempty"`
	WaitMs int    `json:"wait_ms,omitempty"`
}

type ReceivePayload struct {
	Sender  string `json:"sender"`
	SentAt  string `json:"sent_at"`
	Error   bool   `json:"error"`
	Payload string `json:"payload"`
}

type ErrorBody struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type OutboundMessage struct {
	MessageID string
	Sender    string
	SentAt    string
	Error     bool
	Payload   string
	ReadyAt   time.Time
}

type GenerationTaskRequest struct {
	MessageID        string                 `json:"message_id"`
	Sender           string                 `json:"sender"`
	SentAt           string                 `json:"sent_at"`
	JSONSchema       map[string]interface{} `json:"json_schema"`
	SampleCount      int                    `json:"sample_count"`
	Constraints      string                 `json:"constraints,omitempty"`
	TransportBaseURL string                 `json:"transport_base_url,omitempty"`
	SegmentMaxBytes  int                    `json:"segment_max_bytes,omitempty"`
}
