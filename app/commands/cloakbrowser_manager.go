package commands

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tmos/fingerbrower/cloakbrowser"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type CloakBrowserManager struct {
	store      *sqlite.InstanceStore
	portAlloc  *instance.PortAllocator
	dataDir    string
	binaryPath string
	mu         sync.Mutex
	clients    map[string]*cloakbrowser.Client
}

func NewCloakBrowserManager(db *sqlite.DB, binaryPath string) *CloakBrowserManager {
	return &CloakBrowserManager{
		store:      sqlite.NewInstanceStore(db),
		portAlloc:  instance.NewPortAllocator(9222, 65535),
		dataDir:    os.TempDir(),
		binaryPath: strings.TrimSpace(binaryPath),
		clients:    make(map[string]*cloakbrowser.Client),
	}
}

func (m *CloakBrowserManager) Start(ctx context.Context, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error) {
	if strings.TrimSpace(m.binaryPath) == "" {
		return nil, fmt.Errorf("CloakBrowser path is not configured; set CLOAKBROWSER_PATH or BROWSER_BINARY, or use BROWSER_ENGINE=local-chrome for development fallback")
	}

	m.mu.Lock()
	port, err := m.portAlloc.Allocate()
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("no available port: %w", err)
	}

	instanceID := uuid.New().String()
	userDataDir := filepath.Join(m.dataDir, fmt.Sprintf("fingerbrower-%s", instanceID))
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	instanceName := cloakInstanceName(cfg, instanceID)
	infoURL, err := writeCloakInstanceInfoPage(userDataDir, instanceName, cfg, port, instanceID)
	if err != nil {
		fmt.Printf("Warning: failed to write instance info page: %v\n", err)
		infoURL = ""
	}

	client, err := cloakbrowser.NewClientWithUserDataDir(m.binaryPath, port, userDataDir)
	if err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, err
	}

	proxyURL := ""
	if cfg.Proxy != nil {
		proxyURL = cfg.Proxy.URL
	}

	if err := client.StartWithOptions(ctx, cfg.Fingerprint, &cloakbrowser.LaunchOptions{
		ProxyURL: proxyURL,
		Headless: cfg.Headless,
		StartURL: infoURL,
	}); err != nil {
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		m.mu.Unlock()
		return nil, err
	}
	cmd := client.Command()
	m.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("cloakbrowser process did not start correctly")
	}

	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := waitForCloakBrowserReady(readyCtx, port); err != nil {
		_ = client.Stop()
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("CloakBrowser not ready: %w", err)
	}

	if err := applyCloakFingerprintOverrides(ctx, port, cfg); err != nil {
		_ = client.Stop()
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, err
	}

	now := time.Now()
	inst := &instance.BrowserInstance{
		ID:           instanceID,
		Name:         instanceName,
		Status:       instance.StatusRunning,
		Fingerprint:  cfg.Fingerprint,
		ProxyID:      "",
		ProxyURL:     "",
		AccountID:    cfg.AccountID,
		CDPEndpoint:  fmt.Sprintf("ws://localhost:%d", port),
		PID:          cmd.Process.Pid,
		Port:         port,
		UserDataDir:  userDataDir,
		Group:        cfg.Group,
		Headless:     cfg.Headless,
		StartedAt:    now,
		LastActiveAt: now,
		CreatedAt:    now,
	}
	if cfg.Proxy != nil {
		inst.ProxyID = cfg.Proxy.ID
		inst.ProxyURL = cfg.Proxy.URL
	}

	m.mu.Lock()
	m.clients[instanceID] = client
	m.mu.Unlock()

	if _, err := m.store.Save(inst); err != nil {
		fmt.Printf("Warning: failed to save instance to store: %v\n", err)
	}

	go m.monitorClient(instanceID, client, cmd, port)

	return inst, nil
}

func (m *CloakBrowserManager) Stop(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, err := m.store.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	if client, ok := m.clients[id]; ok {
		if err := client.Stop(); err != nil {
			return err
		}
	} else {
		proc, _ := os.FindProcess(inst.PID)
		if proc != nil {
			_ = proc.Kill()
		}
	}

	m.portAlloc.Release(inst.Port)
	inst.Status = instance.StatusStopped
	inst.LastActiveAt = time.Now()
	if err := m.store.Update(inst); err != nil {
		return err
	}

	delete(m.clients, id)
	return nil
}

func (m *CloakBrowserManager) Restart(ctx context.Context, inst *instance.BrowserInstance, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error) {
	if strings.TrimSpace(m.binaryPath) == "" {
		return nil, fmt.Errorf("CloakBrowser path is not configured; set CLOAKBROWSER_PATH or BROWSER_BINARY, or use BROWSER_ENGINE=local-chrome for development fallback")
	}

	m.mu.Lock()
	port, err := m.portAlloc.Allocate()
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("no available port: %w", err)
	}

	userDataDir := inst.UserDataDir
	if userDataDir == "" {
		userDataDir = filepath.Join(m.dataDir, fmt.Sprintf("fingerbrower-%s", inst.ID))
	}
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	instanceName := strings.TrimSpace(cfg.Name)
	if instanceName == "" {
		instanceName = cloakInstanceName(cfg, inst.ID)
	}

	infoURL, err := writeCloakInstanceInfoPage(userDataDir, instanceName, cfg, port, inst.ID)
	if err != nil {
		fmt.Printf("Warning: failed to write instance info page: %v\n", err)
		infoURL = ""
	}

	client, err := cloakbrowser.NewClientWithUserDataDir(m.binaryPath, port, userDataDir)
	if err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, err
	}

	proxyURL := ""
	if cfg.Proxy != nil {
		proxyURL = cfg.Proxy.URL
	}

	if err := client.StartWithOptions(ctx, cfg.Fingerprint, &cloakbrowser.LaunchOptions{
		ProxyURL: proxyURL,
		Headless: cfg.Headless,
		StartURL: infoURL,
	}); err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, err
	}
	cmd := client.Command()
	m.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		m.portAlloc.Release(port)
		return nil, fmt.Errorf("cloakbrowser process did not start correctly")
	}

	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := waitForCloakBrowserReady(readyCtx, port); err != nil {
		_ = client.Stop()
		m.portAlloc.Release(port)
		return nil, fmt.Errorf("CloakBrowser not ready: %w", err)
	}

	if err := applyCloakFingerprintOverrides(ctx, port, cfg); err != nil {
		_ = client.Stop()
		m.portAlloc.Release(port)
		return nil, err
	}

	now := time.Now()
	inst.Name = instanceName
	inst.Status = instance.StatusRunning
	inst.Fingerprint = cfg.Fingerprint
	inst.ProxyID = ""
	inst.ProxyURL = ""
	if cfg.Proxy != nil {
		inst.ProxyID = cfg.Proxy.ID
		inst.ProxyURL = cfg.Proxy.URL
	}
	inst.AccountID = cfg.AccountID
	inst.CDPEndpoint = fmt.Sprintf("ws://localhost:%d", port)
	inst.PID = cmd.Process.Pid
	inst.Port = port
	inst.UserDataDir = userDataDir
	inst.Group = cfg.Group
	inst.Headless = cfg.Headless
	inst.StartedAt = now
	inst.LastActiveAt = now
	if err := m.store.Update(inst); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.clients[inst.ID] = client
	m.mu.Unlock()
	go m.monitorClient(inst.ID, client, cmd, port)

	return inst, nil
}

func (m *CloakBrowserManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, err := m.store.Get(id)
	if err != nil {
		return err
	}
	if inst.UserDataDir != "" {
		if err := os.RemoveAll(inst.UserDataDir); err != nil {
			return fmt.Errorf("failed to remove user data dir: %w", err)
		}
	}
	if err := m.store.Delete(id); err != nil {
		return err
	}
	delete(m.clients, id)
	return nil
}

func (m *CloakBrowserManager) monitorClient(instanceID string, client *cloakbrowser.Client, cmd *exec.Cmd, port int) {
	err := cmd.Wait()
	if err != nil {
		fmt.Printf("Instance %s exited: %v\n", instanceID, err)
	}

	m.mu.Lock()
	current, ok := m.clients[instanceID]
	if !ok || current != client {
		m.mu.Unlock()
		return
	}
	delete(m.clients, instanceID)
	m.portAlloc.Release(port)

	inst, err := m.store.Get(instanceID)
	if err != nil {
		m.mu.Unlock()
		return
	}
	inst.Status = instance.StatusStopped
	inst.LastActiveAt = time.Now()
	if err := m.store.Update(inst); err != nil {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
}

func waitForCloakBrowserReady(ctx context.Context, port int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	url := fmt.Sprintf("http://localhost:%d/json", port)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				return nil
			}
		}
	}
}

func applyCloakFingerprintOverrides(ctx context.Context, port int, cfg *instance.InstanceConfig) error {
	if cfg == nil || cfg.Fingerprint == nil {
		return nil
	}
	timezone := strings.TrimSpace(cfg.Fingerprint.Timezone)
	if timezone == "" {
		return nil
	}

	wsURL, err := resolveWebSocketURL(port)
	if err != nil {
		return fmt.Errorf("failed to resolve CDP target: %w", err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect CDP: %w", err)
	}
	defer conn.Close()

	payload := map[string]interface{}{
		"id":     1,
		"method": "Emulation.setTimezoneOverride",
		"params": map[string]interface{}{
			"timezoneId": timezone,
		},
	}
	if err := conn.WriteJSON(payload); err != nil {
		return fmt.Errorf("failed to set timezone: %w", err)
	}

	for {
		var resp map[string]interface{}
		if err := conn.ReadJSON(&resp); err != nil {
			return fmt.Errorf("failed to read CDP response: %w", err)
		}
		if !isResponseID(resp, 1) {
			continue
		}
		if errObj, ok := resp["error"]; ok && errObj != nil {
			return fmt.Errorf("cdp error: %v", errObj)
		}
		return nil
	}
}

func cloakInstanceName(cfg *instance.InstanceConfig, fallbackID string) string {
	instanceName := strings.TrimSpace(cfg.Name)
	if instanceName == "" {
		if cfg.AccountLabel != "" {
			instanceName = cfg.AccountLabel
		} else {
			instanceName = fmt.Sprintf("Instance-%s", fallbackID[:8])
		}
	}
	return instanceName
}

type cloakInstanceInfoData struct {
	Title               string
	InstanceID          string
	AccountID           string
	AccountLabel        string
	ProxyURL            string
	Port                int
	Group               string
	FingerprintSeed     string
	FingerprintPlatform string
	Timezone            string
}

func writeCloakInstanceInfoPage(userDataDir string, instanceName string, cfg *instance.InstanceConfig, port int, instanceID string) (string, error) {
	infoPath := filepath.Join(userDataDir, "instance-info.html")
	fpSeed := ""
	fpPlatform := ""
	fpTimezone := ""
	if cfg.Fingerprint != nil {
		fpSeed = cfg.Fingerprint.Seed
		fpPlatform = cfg.Fingerprint.Platform
		fpTimezone = cfg.Fingerprint.Timezone
	}

	data := cloakInstanceInfoData{
		Title:               instanceName,
		InstanceID:          instanceID,
		AccountID:           cfg.AccountID,
		AccountLabel:        cfg.AccountLabel,
		ProxyURL:            "",
		Port:                port,
		Group:               cfg.Group,
		FingerprintSeed:     fpSeed,
		FingerprintPlatform: fpPlatform,
		Timezone:            fpTimezone,
	}
	if cfg.Proxy != nil {
		data.ProxyURL = cfg.Proxy.URL
	}

	tpl := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>{{.Title}}</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 24px; color: #111827; }
    h1 { font-size: 20px; margin-bottom: 12px; }
    .meta { display: grid; grid-template-columns: 140px 1fr; row-gap: 8px; column-gap: 12px; }
    .label { color: #6b7280; }
    .value { color: #111827; word-break: break-all; }
  </style>
</head>
<body>
  <h1>{{.Title}}</h1>
  <div class="meta">
    <div class="label">Instance ID</div><div class="value">{{.InstanceID}}</div>
    <div class="label">Account</div><div class="value">{{if .AccountLabel}}{{.AccountLabel}}{{else}}-{{end}}</div>
    <div class="label">Account ID</div><div class="value">{{if .AccountID}}{{.AccountID}}{{else}}-{{end}}</div>
    <div class="label">Proxy</div><div class="value">{{if .ProxyURL}}{{.ProxyURL}}{{else}}-{{end}}</div>
    <div class="label">Port</div><div class="value">{{.Port}}</div>
    <div class="label">Group</div><div class="value">{{if .Group}}{{.Group}}{{else}}-{{end}}</div>
    <div class="label">FP Seed</div><div class="value">{{if .FingerprintSeed}}{{.FingerprintSeed}}{{else}}-{{end}}</div>
    <div class="label">FP Platform</div><div class="value">{{if .FingerprintPlatform}}{{.FingerprintPlatform}}{{else}}-{{end}}</div>
    <div class="label">Timezone</div><div class="value">{{if .Timezone}}{{.Timezone}}{{else}}-{{end}}</div>
  </div>
</body>
</html>`

	t, err := template.New("instance-info").Parse(tpl)
	if err != nil {
		return "", err
	}
	file, err := os.Create(infoPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := t.Execute(file, data); err != nil {
		return "", err
	}

	u := url.URL{Scheme: "file", Path: infoPath}
	return u.String(), nil
}
