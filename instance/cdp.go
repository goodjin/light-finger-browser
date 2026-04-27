package instance

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
)

// CDPClientInterface defines the interface for CDP client operations.
type CDPClientInterface interface {
	Navigate(ctx context.Context, url string) error
	Click(ctx context.Context, selector string) error
	Type(ctx context.Context, selector string, text string) error
	Screenshot(ctx context.Context) ([]byte, error)
	Evaluate(ctx context.Context, script string) (interface{}, error)
	Close() error
}

// CDPClient provides Chrome DevTools Protocol communication.
type CDPClient struct {
	conn WebSocket
	mu   sync.Mutex
	msgID int64
}

// Ensure *CDPClient implements CDPClientInterface
var _ CDPClientInterface = (*CDPClient)(nil)

// CDPMessage represents a CDP protocol message.
type CDPMessage struct {
	ID     int64                  `json:"id"`
	Method string                  `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// CDPResponse represents a CDP protocol response.
type CDPResponse struct {
	ID     int64                   `json:"id"`
	Result json.RawMessage         `json:"result,omitempty"`
	Error  *CDPError               `json:"error,omitempty"`
}

// CDPError represents a CDP protocol error.
type CDPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewCDPClient creates a new CDP client connected to the given endpoint.
func NewCDPClient(conn WebSocket) *CDPClient {
	return &CDPClient{
		conn:  conn,
		msgID: 0,
	}
}

// Navigate opens the specified URL in the browser.
func (c *CDPClient) Navigate(ctx context.Context, url string) error {
	_, err := c.execute(ctx, "Page.navigate", map[string]interface{}{
		"url": url,
	})
	return err
}

// Click simulates a click on the element matching the CSS selector.
func (c *CDPClient) Click(ctx context.Context, selector string) error {
	_, err := c.execute(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression": fmt.Sprintf(`document.querySelector('%s').click()`, selector),
	})
	return err
}

// Type simulates keyboard input on the element matching the CSS selector.
func (c *CDPClient) Type(ctx context.Context, selector string, text string) error {
	_, err := c.execute(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression": fmt.Sprintf(`
			(function() {
				var el = document.querySelector('%s');
				el.focus();
				el.value = '%s';
				el.dispatchEvent(new Event('input', { bubbles: true }));
			})()
		`, selector, text),
	})
	return err
}

// Screenshot captures a screenshot of the current page.
func (c *CDPClient) Screenshot(ctx context.Context) ([]byte, error) {
	result, err := c.execute(ctx, "Page.captureScreenshot", nil)
	if err != nil {
		return nil, err
	}

	data, ok := result["data"].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected screenshot response format")
	}

	return base64.StdEncoding.DecodeString(data)
}

// Evaluate executes a JavaScript expression and returns the result.
func (c *CDPClient) Evaluate(ctx context.Context, script string) (interface{}, error) {
	return c.execute(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression": script,
	})
}

// Close closes the WebSocket connection.
func (c *CDPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// execute sends a CDP command and waits for the response.
func (c *CDPClient) execute(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	c.mu.Lock()
	c.msgID++
	msgID := c.msgID
	c.mu.Unlock()

	if c.conn == nil {
		return nil, ErrNotConnected
	}

	msg := CDPMessage{
		ID:     msgID,
		Method: method,
		Params: params,
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		return nil, fmt.Errorf("failed to send CDP message: %w", err)
	}

	// Wait for response with timeout
	respCh := make(chan CDPResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		var resp CDPResponse
		if err := c.conn.ReadJSON(&resp); err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, fmt.Errorf("failed to read CDP response: %w", err)
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, fmt.Errorf("CDP error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
		}

		if len(resp.Result) == 0 {
			return nil, nil
		}

		var result map[string]interface{}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("failed to parse CDP result: %w", err)
		}

		return result, nil
	}
}

// ConnectCDP establishes a CDP connection to the given endpoint.
func ConnectCDP(ctx context.Context, endpoint string) (*CDPClient, error) {
	conn, _, err := DefaultDialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial CDP endpoint: %w", err)
	}

	return NewCDPClient(conn), nil
}