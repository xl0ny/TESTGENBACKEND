package store

import (
	"sync"
	"time"

	"github.com/testgen/transport/internal/models"
)

type MessageMeta struct {
	Sender  string
	SentAt  string
}

type AssemblyState struct {
	TotalSegments      int
	Chunks             map[int]string
	LastNewSegmentAt   time.Time
	CyclesWithoutNew   int
	Meta               MessageMeta
}

type Store struct {
	mu sync.RWMutex

	metaByMessage map[string]MessageMeta
	assembly      map[string]*AssemblyState
	outbound      []models.OutboundMessage
	waiters       []chan struct{}
}

func New() *Store {
	return &Store{
		metaByMessage: make(map[string]MessageMeta),
		assembly:      make(map[string]*AssemblyState),
		outbound:      make([]models.OutboundMessage, 0),
	}
}

func (s *Store) PutMeta(messageID, sender, sentAt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metaByMessage[messageID] = MessageMeta{Sender: sender, SentAt: sentAt}
}

func (s *Store) Meta(messageID string) (MessageMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.metaByMessage[messageID]
	return m, ok
}

func (s *Store) EnsureAssembly(messageID string, total int, meta MessageMeta) *AssemblyState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.assembly[messageID]
	if !ok {
		st = &AssemblyState{
			TotalSegments:    total,
			Chunks:           make(map[int]string),
			LastNewSegmentAt: time.Now(),
			Meta:             meta,
		}
		s.assembly[messageID] = st
	} else if total > st.TotalSegments {
		st.TotalSegments = total
	}
	if meta.Sender != "" {
		st.Meta = meta
	}
	return st
}

func (s *Store) AddChunk(messageID string, index int, chunk string, total int, meta MessageMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.assembly[messageID]
	if st == nil {
		st = &AssemblyState{
			TotalSegments:    total,
			Chunks:           make(map[int]string),
			LastNewSegmentAt: time.Now(),
			Meta:             meta,
		}
		s.assembly[messageID] = st
	}
	if total > st.TotalSegments {
		st.TotalSegments = total
	}
	st.Chunks[index] = chunk
	st.LastNewSegmentAt = time.Now()
	st.CyclesWithoutNew = 0
	if meta.Sender != "" {
		st.Meta = meta
	}
}

func (s *Store) SnapshotAssembly() map[string]*AssemblyState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*AssemblyState, len(s.assembly))
	for id, st := range s.assembly {
		copyChunks := make(map[int]string, len(st.Chunks))
		for k, v := range st.Chunks {
			copyChunks[k] = v
		}
		cp := *st
		cp.Chunks = copyChunks
		out[id] = &cp
	}
	return out
}

func (s *Store) RemoveAssembly(messageID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.assembly, messageID)
	delete(s.metaByMessage, messageID)
}

func (s *Store) IncrementIdleCycles() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, st := range s.assembly {
		st.CyclesWithoutNew++
	}
}

func (s *Store) EnqueueOutbound(msg models.OutboundMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outbound = append(s.outbound, msg)
	for _, ch := range s.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	s.waiters = nil
}

func (s *Store) DequeueOutbound(sender string) (models.OutboundMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, msg := range s.outbound {
		if sender != "" && msg.Sender != sender {
			continue
		}
		s.outbound = append(s.outbound[:i], s.outbound[i+1:]...)
		return msg, true
	}
	return models.OutboundMessage{}, false
}

func (s *Store) WaitOutbound(timeout time.Duration) {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.waiters = append(s.waiters, ch)
	s.mu.Unlock()

	select {
	case <-ch:
	case <-time.After(timeout):
	}
}

func AssemblePayload(st *AssemblyState) (string, bool) {
	if st.TotalSegments <= 0 {
		return "", false
	}
	for i := 0; i < st.TotalSegments; i++ {
		if _, ok := st.Chunks[i]; !ok {
			return "", false
		}
	}
	var b []byte
	for i := 0; i < st.TotalSegments; i++ {
		b = append(b, st.Chunks[i]...)
	}
	return string(b), true
}

func PartialPayload(st *AssemblyState) string {
	if len(st.Chunks) == 0 {
		return ""
	}
	maxIdx := -1
	for i := range st.Chunks {
		if i > maxIdx {
			maxIdx = i
		}
	}
	var b []byte
	for i := 0; i <= maxIdx; i++ {
		if chunk, ok := st.Chunks[i]; ok {
			b = append(b, chunk...)
		}
	}
	return string(b)
}
