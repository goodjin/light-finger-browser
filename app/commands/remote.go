package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CDPTarget represents a Chrome DevTools Protocol target (page/tab).
type CDPTarget struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	WebSocket string `json:"webSocketDebuggerUrl"`
}

// RemoteBrowserClient manages connections to remote browser instances.
type RemoteBrowserClient struct {
	host       string
	port       int
	httpClient *http.Client
}

func newRemoteBrowserClient(host string, port int) (*RemoteBrowserClient, error) {
	return &RemoteBrowserClient{
		host: host,
		port: port,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *RemoteBrowserClient) GetCDPEndpoint() string {
	return fmt.Sprintf("ws://%s:%d/devtools/browser", c.host, c.port)
}

func (c *RemoteBrowserClient) ListTargets() ([]*CDPTarget, error) {
	url := fmt.Sprintf("http://%s:%d/json", c.host, c.port)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get targets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CDP endpoint returned status %d", resp.StatusCode)
	}

	var targets []*CDPTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("failed to decode targets: %w", err)
	}

	return targets, nil
}

func (c *RemoteBrowserClient) IsRunning() bool {
	conn, err := c.httpClient.Get(fmt.Sprintf("http://%s:%d/json", c.host, c.port))
	if err != nil {
		return false
	}
	conn.Body.Close()
	return conn.StatusCode == http.StatusOK
}

// RemoteBrowserService manages connections to remote browser instances.
type RemoteBrowserService struct {
	mu      sync.Mutex
	clients sync.Map // key: "host:port", value: *RemoteBrowserClient
}

func NewRemoteBrowserService() *RemoteBrowserService {
	return &RemoteBrowserService{}
}

func (s *RemoteBrowserService) Connect(ctx context.Context, host string, port int, binaryPath string) error {
	key := fmt.Sprintf("%s:%d", host, port)

	// Verify the remote browser is accessible
	client, err := newRemoteBrowserClient(host, port)
	if err != nil {
		return fmt.Errorf("failed to create remote browser client: %w", err)
	}

	if !client.IsRunning() {
		return fmt.Errorf("remote browser not accessible at %s:%d", host, port)
	}

	s.clients.Store(key, client)
	return nil
}

func (s *RemoteBrowserService) Disconnect(ctx context.Context, host string, port int) error {
	key := fmt.Sprintf("%s:%d", host, port)

	_, ok := s.clients.Load(key)
	if !ok {
		return fmt.Errorf("not connected to %s:%d", host, port)
	}

	s.clients.Delete(key)
	return nil
}

func (s *RemoteBrowserService) ListTargets(ctx context.Context, host string, port int) ([]*CDPTarget, error) {
	key := fmt.Sprintf("%s:%d", host, port)

	client, ok := s.clients.Load(key)
	if !ok {
		return nil, fmt.Errorf("not connected to %s:%d", host, port)
	}

	return client.(*RemoteBrowserClient).ListTargets()
}

func (s *RemoteBrowserService) GetCDPEndpoint(ctx context.Context, host string, port int) (string, error) {
	key := fmt.Sprintf("%s:%d", host, port)

	client, ok := s.clients.Load(key)
	if !ok {
		return "", fmt.Errorf("not connected to %s:%d", host, port)
	}

	return client.(*RemoteBrowserClient).GetCDPEndpoint(), nil
}
