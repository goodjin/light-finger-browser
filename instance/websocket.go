package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

// Conn represents a WebSocket connection wrapping gorilla/websocket.Conn.
type Conn struct {
	mu     sync.Mutex
	closed bool
	conn   *websocket.Conn
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
	// Parse the address - if it's not a ws:// URL, add the scheme
	wsURL := addr
	if !strings.HasPrefix(addr, "ws://") && !strings.HasPrefix(addr, "wss://") {
		wsURL = "ws://" + addr
	}

	if _, err := url.Parse(wsURL); err != nil {
		return nil, nil, err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: d.HandshakeTimeout,
	}

	log.Printf("[WebSocket DialContext] Connecting to: %s", wsURL)
	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		log.Printf("[WebSocket DialContext] Error: %v, resp: %v", err, resp)
		return nil, resp, err
	}
	log.Printf("[WebSocket DialContext] Connected successfully!")

	return &Conn{conn: conn}, resp, nil
}

// ReadJSON reads a JSON message from the WebSocket connection.
func (c *Conn) ReadJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return ErrNotConnected
	}

	return c.conn.ReadJSON(v)
}

// WriteJSON writes a JSON message to the WebSocket connection.
func (c *Conn) WriteJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return ErrNotConnected
	}

	return c.conn.WriteJSON(v)
}

// Close closes the WebSocket connection.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
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
	upgrader := websocket.Upgrader{
		ReadBufferSize:  u.ReadBufferSize,
		WriteBufferSize: u.WriteBufferSize,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &Conn{conn: conn}, nil
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
	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}
