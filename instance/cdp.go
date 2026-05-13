package instance

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// CDPClientInterface defines the interface for CDP client operations.
type CDPClientInterface interface {
	Navigate(ctx context.Context, url string) error
	Click(ctx context.Context, selector string) error
	Type(ctx context.Context, selector string, text string) error
	Screenshot(ctx context.Context) ([]byte, error)
	Evaluate(ctx context.Context, script string) (interface{}, error)
	// CreateTarget creates a new target (tab) and returns its URL
	CreateTarget(ctx context.Context, url string) (string, error)
	// CloseTarget closes a browser target (tab) by its targetId
	CloseTarget(ctx context.Context, targetId string) error
	// BrowserContext management
	CreateBrowserContext(ctx context.Context) (string, error)
	CloseBrowserContext(ctx context.Context, contextId string) error
	CreateTargetWithContext(ctx context.Context, url string, contextId string) (string, error)
	GetTargets(ctx context.Context) ([]*CDPTarget, error)
	// IsConnected checks if the CDP connection is still alive by sending a simple command
	IsConnected(ctx context.Context) bool
	Close() error
}

// CDPClient provides Chrome DevTools Protocol communication.
type CDPClient struct {
	conn  WebSocket
	mu    sync.Mutex
	msgID int64
}

// Ensure *CDPClient implements CDPClientInterface
var _ CDPClientInterface = (*CDPClient)(nil)

// CDPMessage represents a CDP protocol message.
type CDPMessage struct {
	ID     int64                  `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// CDPResponse represents a CDP protocol response.
type CDPResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *CDPError       `json:"error,omitempty"`
}

// CDPError represents a CDP protocol error.
type CDPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CDPTarget represents a CDP browser target.
type CDPTarget struct {
	Type             string `json:"type"`
	ID               string `json:"id"`
	Title            string `json:"title"`
	URL              string `json:"url"`
	Attached         bool   `json:"attached"`
	BrowserContextID string `json:"browserContextId,omitempty"`
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

// CreateTarget creates a new browser target (tab) and returns its targetId.
func (c *CDPClient) CreateTarget(ctx context.Context, url string) (string, error) {
	result, err := c.execute(ctx, "Target.createTarget", map[string]interface{}{
		"url": url,
	})
	if err != nil {
		return "", err
	}

	targetID, ok := result["targetId"].(string)
	if !ok {
		return "", fmt.Errorf("CreateTarget response missing targetId")
	}
	return targetID, nil
}

// CloseTarget closes a browser target (tab) by its targetId.
func (c *CDPClient) CloseTarget(ctx context.Context, targetId string) error {
	_, err := c.execute(ctx, "Target.closeTarget", map[string]interface{}{
		"targetId": targetId,
	})
	return err
}

// Close closes the WebSocket connection.
func (c *CDPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// CreateBrowserContext creates a new browser context and returns its ID.
func (c *CDPClient) CreateBrowserContext(ctx context.Context) (string, error) {
	result, err := c.execute(ctx, "Target.createBrowserContext", nil)
	if err != nil {
		return "", err
	}
	browserContextId, ok := result["browserContextId"].(string)
	if !ok {
		return "", fmt.Errorf("CreateBrowserContext response missing browserContextId")
	}
	return browserContextId, nil
}

// CloseBrowserContext closes a browser context by its ID.
func (c *CDPClient) CloseBrowserContext(ctx context.Context, contextId string) error {
	_, err := c.execute(ctx, "Target.closeBrowserContext", map[string]interface{}{
		"browserContextId": contextId,
	})
	return err
}

// CreateTargetWithContext creates a new target (tab) in the specified browser context.
// Note: Do NOT set newWindow to false when creating targets in a fresh BrowserContext,
// as Chrome will return "no browser is open" error when there are no existing windows.
// Let Chrome decide whether to create a new window or reuse existing ones.
func (c *CDPClient) CreateTargetWithContext(ctx context.Context, url string, contextId string) (string, error) {
	log.Printf("[CreateTargetWithContext] Creating target with URL: %s, contextId: %s", url, contextId)
	result, err := c.execute(ctx, "Target.createTarget", map[string]interface{}{
		"url":              url,
		"browserContextId": contextId,
		// Note: Not setting newWindow parameter - Chrome will auto-create a window if needed
	})
	if err != nil {
		log.Printf("[CreateTargetWithContext] Error: %v", err)
		return "", err
	}
	targetID, ok := result["targetId"].(string)
	if !ok {
		return "", fmt.Errorf("CreateTarget response missing targetId")
	}
	log.Printf("[CreateTargetWithContext] Successfully created target: %s", targetID)
	return targetID, nil
}

// GetTargets returns a list of all browser targets.
func (c *CDPClient) GetTargets(ctx context.Context) ([]*CDPTarget, error) {
	result, err := c.execute(ctx, "Target.getTargets", nil)
	if err != nil {
		return nil, err
	}

	targetsData, ok := result["targetInfos"]
	if !ok {
		return nil, fmt.Errorf("GetTargets response missing targetInfos")
	}

	targetsJSON, err := json.Marshal(targetsData)
	if err != nil {
		return nil, err
	}

	var targets []*CDPTarget
	if err := json.Unmarshal(targetsJSON, &targets); err != nil {
		return nil, err
	}

	return targets, nil
}

// IsConnected checks if the CDP connection is still alive by sending a simple command.
// Returns true if the connection is valid, false otherwise.
func (c *CDPClient) IsConnected(ctx context.Context) bool {
	// Create a context with a short timeout to avoid hanging
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Try to execute a simple command to verify the connection
	_, err := c.execute(ctx, "Target.getTargets", nil)
	return err == nil
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
