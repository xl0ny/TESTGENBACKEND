package assembly

import (
	"log"
	"time"

	"github.com/testgen/transport/internal/config"
	"github.com/testgen/transport/internal/models"
	"github.com/testgen/transport/internal/store"
)

type Runner struct {
	cfg   config.Config
	store *store.Store
}

func NewRunner(cfg config.Config, st *store.Store) *Runner {
	return &Runner{cfg: cfg, store: st}
}

func (r *Runner) Start(stop <-chan struct{}) {
	ticker := time.NewTicker(r.cfg.AssemblyInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			r.tick()
		}
	}
}

func (r *Runner) tick() {
	r.store.IncrementIdleCycles()
	snapshot := r.store.SnapshotAssembly()

	for messageID, st := range snapshot {
		meta := st.Meta
		if meta.Sender == "" {
			if m, ok := r.store.Meta(messageID); ok {
				meta = m
			}
		}

		if payload, ok := store.AssemblePayload(st); ok {
			r.store.EnqueueOutbound(models.OutboundMessage{
				MessageID: messageID,
				Sender:    meta.Sender,
				SentAt:    meta.SentAt,
				Error:     false,
				Payload:   payload,
				ReadyAt:   time.Now(),
			})
			r.store.RemoveAssembly(messageID)
			log.Printf("assembled message %s (%d segments)", messageID, st.TotalSegments)
			continue
		}

		if partial := store.PartialPayload(st); partial != "" {
			r.store.EnqueueOutbound(models.OutboundMessage{
				MessageID: messageID,
				Sender:    meta.Sender,
				SentAt:    meta.SentAt,
				Error:     false,
				Payload:   partial,
				ReadyAt:   time.Now(),
			})
		}

		if st.CyclesWithoutNew >= r.cfg.TimeoutCycles {
			payload := store.PartialPayload(st)
			r.store.EnqueueOutbound(models.OutboundMessage{
				MessageID: messageID,
				Sender:    meta.Sender,
				SentAt:    meta.SentAt,
				Error:     true,
				Payload:   payload,
				ReadyAt:   time.Now(),
			})
			r.store.RemoveAssembly(messageID)
			log.Printf("assembly timeout message %s (cycles=%d)", messageID, st.CyclesWithoutNew)
		}
	}
}
