package adapter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tmos/facebook/internal/proxy"
)

// BrightDataAdapter implements ProxyProvider for Bright Data service
type BrightDataAdapter struct {
	apiKey    string
	zone      string
	httpClient *http.Client
}

// NewBrightDataAdapter creates a new Bright Data adapter
func NewBrightDataAdapter(apiKey, zone string) *BrightDataAdapter {
	return &BrightDataAdapter{
		apiKey: apiKey,
		zone:   zone,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetProxy retrieves a proxy from Bright Data
func (a *BrightDataAdapter) GetProxy(ctx context.Context, country string, proxyType proxy.ProxyType) (*proxy.Proxy, error) {
	// Bright Data session-style API response parsing
	// In production, this would call their API endpoint
	resp := &BrightDataResponse{
		IP:       fmt.Sprintf("zproxy.lum-superproxy.io"),
		Port:     22225,
		Username: fmt.Sprintf("zd-%s-%s", a.zone, country),
		Password: a.apiKey,
	}

	proxyID := fmt.Sprintf("bd-%d", time.Now().UnixNano())

	return &proxy.Proxy{
		ID:       proxyID,
		IP:       resp.IP,
		Port:     resp.Port,
		Country:  country,
		Type:     proxyType,
		Username: resp.Username,
		Password: resp.Password,
		Status:   proxy.ProxyStatusAvailable,
		Provider: "brightdata",
	}, nil
}

// ReleaseProxy releases a proxy back to Bright Data
func (a *BrightDataAdapter) ReleaseProxy(ctx context.Context, id string) error {
	// Bright Data releases automatically when session ends
	return nil
}

// CheckProxy tests if a proxy is working
func (a *BrightDataAdapter) CheckProxy(ctx context.Context, p *proxy.Proxy) (bool, int, error) {
	start := time.Now()

	// Test proxy connectivity
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	proxyURL := fmt.Sprintf("http://%s:%s@%s:%d", p.Username, p.Password, p.IP, p.Port)
	transport := &http.Transport{
		Proxy: http.ProxyURL(nil),
	}

	// For testing, we use a simple check
	req, err := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/ip", nil)
	if err != nil {
		return false, 0, err
	}

	// Use the proxy
	transport.Proxy = func(*http.Request) (*url.URL, error) {
		return url.Parse(proxyURL)
	}

	client.Transport = transport

	resp, err := client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, 0, errors.New("proxy check failed")
	}

	latency := int(time.Since(start).Milliseconds())
	return true, latency, nil
}

// BrightDataResponse represents Bright Data API response
type BrightDataResponse struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}