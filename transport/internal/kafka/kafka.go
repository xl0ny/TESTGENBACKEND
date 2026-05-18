package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/testgen/transport/internal/config"
	"github.com/testgen/transport/internal/models"
	"github.com/testgen/transport/internal/store"
)

type Client struct {
	writer *kafka.Writer
	reader *kafka.Reader
	store  *store.Store
}

func NewClient(cfg config.Config, st *store.Store) *Client {
	return &Client{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(cfg.KafkaBrokers...),
			Topic:        cfg.KafkaTopicSegments,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.KafkaBrokers,
			Topic:          cfg.KafkaTopicSegments,
			GroupID:        "transport-assembler",
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		}),
		store: st,
	}
}

func (c *Client) ProduceSegment(ctx context.Context, seg models.Segment) error {
	body, err := json.Marshal(seg)
	if err != nil {
		return err
	}
	return c.writer.WriteMessages(ctx, kafka.Message{Value: body})
}

func (c *Client) ConsumeLoop(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("kafka read: %v", err)
			time.Sleep(time.Second)
			continue
		}
		var seg models.Segment
		if err := json.Unmarshal(msg.Value, &seg); err != nil {
			log.Printf("invalid segment json: %v", err)
			continue
		}
		meta, _ := c.store.Meta(seg.MessageID)
		if meta.Sender == "" {
			meta = store.MessageMeta{SentAt: seg.SentAt}
		}
		c.store.AddChunk(seg.MessageID, seg.SegmentIndex, seg.PayloadChunk, seg.TotalSegments, meta)
	}
}

func (c *Client) Close() error {
	_ = c.reader.Close()
	return c.writer.Close()
}
