package proxy

import (
	"context"
	"log"
	"sync"
	"time"
)

// HealthChecker performs periodic health checks on proxies
type HealthChecker struct {
	manager  *ProxyManager
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewHealthChecker creates a new HealthChecker
func NewHealthChecker(manager *ProxyManager, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		manager:  manager,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the health check worker
func (h *HealthChecker) Start() {
	h.wg.Add(1)
	go h.worker()
	log.Printf("Health checker started with interval: %v", h.interval)
}

// Stop stops the health check worker
func (h *HealthChecker) Stop() {
	close(h.stopCh)
	h.wg.Wait()
	log.Println("Health checker stopped")
}

func (h *HealthChecker) worker() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.checkAllProxies()
		}
	}
}

func (h *HealthChecker) checkAllProxies() {
	ctx := context.Background()

	proxies, err := h.manager.List(ctx, &ProxyFilter{
		Status: ProxyStatusPtr(ProxyStatusInUse),
	})
	if err != nil {
		log.Printf("Failed to list proxies for health check: %v", err)
		return
	}

	for _, p := range proxies {
		go h.checkProxy(p.ID)
	}
}

func (h *HealthChecker) checkProxy(proxyID string) {
	ctx := context.Background()

	err := h.manager.HealthCheck(ctx, proxyID)
	if err != nil {
		log.Printf("Health check failed for proxy %s: %v", proxyID, err)
		return
	}

	proxy, err := h.manager.Get(ctx, proxyID)
	if err != nil {
		return
	}

	if proxy.Status == ProxyStatusDead {
		h.triggerAlert(proxy)
	}
}

func (h *HealthChecker) triggerAlert(proxy *Proxy) {
	alert := map[string]interface{}{
		"type":      "proxy_dead",
		"proxy_id":  proxy.ID,
		"ip":        proxy.IP,
		"provider":  proxy.Provider,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	log.Printf("ALERT: Proxy %s (%s) marked as dead", proxy.ID, proxy.IP)

	// Send to health channel if manager has one
	if h.manager.GetHealthChannel() != nil {
		select {
		case h.manager.GetHealthChannel() <- proxy.ID:
		default:
			log.Printf("Health channel full, alert dropped for proxy %s", proxy.ID)
		}
	}

	// In production, this would also send to a proper alerting system
	// e.g., PagerDuty, Slack, etc.
	log.Printf("Alert triggered for dead proxy: %+v", alert)
}

// CheckNow performs an immediate health check on all in-use proxies
func (h *HealthChecker) CheckNow() {
	h.checkAllProxies()
}