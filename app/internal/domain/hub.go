package domain

// Conn is a single WebSocket (or other) client connection.
type Conn interface {
	SendJSON(v any) error
	IsOpen() bool
}

// ClientHub tracks connected clients per username.
type ClientHub interface {
	Add(username string, conn Conn)
	Remove(username string, conn Conn)
	PushToUser(username string, message any)
	BroadcastExcept(sender string, message any)
}
