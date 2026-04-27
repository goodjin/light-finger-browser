package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ErrConnectionClosed is returned when the WebSocket connection is closed.
var ErrConnectionClosed = errors.New("websocket: connection closed")

// ErrNotConnected is returned when the WebSocket is not connected.
var ErrNotConnected = errors.New("websocket: not connected")

// WebSocket represents a WebSocket connection interface.
type WebSocket interface {
	WriteJSON(v interface{}) error
	ReadJSON(v interface{}) error
	Close() error
}

// Conn represents a WebSocket connection.
type Conn struct {
	mu     sync.Mutex
	closed bool
}

// WebSocketMessage represents a WebSocket message.
type WebSocketMessage struct {
	Type    int
	Payload []byte
}

// DefaultDialer is the default dialer for WebSocket connections.
var DefaultDialer = &Dialer{
	HandshakeTimeout: 10 * time.Second,
}

// Dialer contains options for connecting to a WebSocket server.
type Dialer struct {
	HandshakeTimeout time.Duration
}

// DialContext connects to a WebSocket server using the context.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (*Conn, *http.Response, error) {
	// For testing purposes, this is a stub implementation
	// In production, this would use the actual gorilla/websocket library
	conn := &Conn{}

	// Try to establish a TCP connection
	dialer := net.Dialer{Timeout: d.HandshakeTimeout}
	connFd, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, nil, err
	}

	// Upgrade to WebSocket would happen here
	// For now, return a stub connection
	_ = connFd

	return conn, nil, nil
}

// ReadJSON reads a JSON message from the WebSocket connection.
func (c *Conn) ReadJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrNotConnected
	}

	return errors.New("stub: ReadJSON not implemented")
}

// WriteJSON writes a JSON message to the WebSocket connection.
func (c *Conn) WriteJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrNotConnected
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	_ = data // Stub: would send data over WebSocket
	return nil
}

// Close closes the WebSocket connection.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	return nil
}

// IsClosed returns whether the connection is closed.
func (c *Conn) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// Upgrader upgrades an HTTP connection to a WebSocket connection.
type Upgrader struct {
	ReadBufferSize  int
	WriteBufferSize int
}

// Upgrade upgrades an HTTP connection to a WebSocket connection.
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	// Stub implementation
	return &Conn{}, nil
}

// Message represents a CDP message for WebSocket transport.
type Message struct {
	ID     int64                  `json:"id,omitempty"`
	Method string                 `json:"method,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// Response represents a CDP response for WebSocket transport.
type Response struct {
	ID     int64           `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Error represents a CDP error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// FormatAddr formats an address for WebSocket connection.
func FormatAddr(host string, port int) string {
	return net.JoinHostPort(host, strings.TrimPrefix(strings.TrimPrefix(fmt.Sprintf("%d", port), ":"), "ws://"))
}