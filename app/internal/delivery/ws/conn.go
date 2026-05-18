package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Conn struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func NewConn(ws *websocket.Conn) *Conn {
	return &Conn{ws: ws}
}

func (c *Conn) SendJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws.WriteJSON(v)
}

func (c *Conn) IsOpen() bool {
	return c.ws != nil
}
