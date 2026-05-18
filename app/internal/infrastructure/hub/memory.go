package hub

import (
	"sync"

	"github.com/testgen/app/internal/domain"
)

type MemoryHub struct {
	mu      sync.RWMutex
	clients map[string]map[domain.Conn]struct{}
}

func NewMemoryHub() *MemoryHub {
	return &MemoryHub{
		clients: make(map[string]map[domain.Conn]struct{}),
	}
}

func (h *MemoryHub) Add(username string, conn domain.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[username] == nil {
		h.clients[username] = make(map[domain.Conn]struct{})
	}
	h.clients[username][conn] = struct{}{}
}

func (h *MemoryHub) Remove(username string, conn domain.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.clients[username]
	if set == nil {
		return
	}
	delete(set, conn)
	if len(set) == 0 {
		delete(h.clients, username)
	}
}

func (h *MemoryHub) PushToUser(username string, message any) {
	h.mu.RLock()
	set := h.clients[username]
	conns := make([]domain.Conn, 0, len(set))
	for c := range set {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	for _, c := range conns {
		if c.IsOpen() {
			_ = c.SendJSON(message)
		}
	}
}

func (h *MemoryHub) BroadcastExcept(sender string, message any) {
	h.mu.RLock()
	var targets []domain.Conn
	for user, set := range h.clients {
		if user == sender {
			continue
		}
		for c := range set {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range targets {
		if c.IsOpen() {
			_ = c.SendJSON(message)
		}
	}
}
