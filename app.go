package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/tmos/fingerbrower/app/commands"
	"github.com/tmos/fingerbrower/storage/sqlite"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx            context.Context
	mu             sync.Mutex
	db             *sqlite.DB
	instanceSvc    *commands.InstanceService
	fingerprintSvc *commands.FingerprintService
	remoteSvc      *commands.RemoteBrowserService
	accountSvc     *commands.AccountService
	proxySvc       *commands.ProxyService
	releaseSvc     *commands.ReleaseService
	fpServerSvc   *commands.FingerprintServerService
}

func NewApp() *App {
	return &App{}
}

func init() {
	// Setup file logging
	logDir := filepath.Join(os.Getenv("HOME"), "Library", "Logs", "fingerbrower")
	os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, "fingerbrower.log")

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(f)
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	}
}

func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	log.Println("fingerbrower starting...")

	// Initialize SQLite database - use absolute path in Application Support
	appDataDir := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "fingerbrower")
	os.MkdirAll(appDataDir, 0755)
	dbPath := filepath.Join(appDataDir, "fingerbrower.db")
	log.Println("[Step 1/6] Opening database:", dbPath)
	var err error
	a.db, err = sqlite.NewDB(dbPath)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return
	}
	log.Println("[Step 2/6] Database opened, running migrations...")

	// Run migrations
	if err := a.db.Migrate(); err != nil {
		log.Printf("Failed to run migrations: %v", err)
		return
	}
	log.Println("[Step 3/6] Migrations complete, initializing services...")

	// Initialize services
	a.instanceSvc = commands.NewInstanceService(a.db)
	a.fingerprintSvc = commands.NewFingerprintService()
	a.remoteSvc = commands.NewRemoteBrowserService()
	a.accountSvc = commands.NewAccountService(a.db, a.instanceSvc)
	a.proxySvc = commands.NewProxyService(a.db)
	a.releaseSvc = commands.NewReleaseService()
	a.fpServerSvc = commands.NewFingerprintServerService()

	log.Println("fingerbrower started successfully")
}

func (a *App) OnDomReady(ctx context.Context) {
	runtime.LogInfof(a.ctx, "DOM is ready")
}

func (a *App) OnBeforeClose(ctx context.Context) bool {
	runtime.LogInfof(a.ctx, "Application is closing...")
	if a.db != nil {
		a.db.Close()
	}
	return false
}

func (a *App) OnShutdown(ctx context.Context) {
	runtime.LogInfof(a.ctx, "Application shutdown complete")
}

func (a *App) appContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// ==================== Instance Commands ====================

// CreateInstance creates a new browser instance with the given configuration.
func (a *App) CreateInstance(cfg *commands.InstanceConfig) (*commands.BrowserInstance, error) {
	inst, err := a.instanceSvc.CreateInstance(a.appContext(), cfg)
	if err != nil {
		return nil, err
	}
	return commands.ToBrowserInstance(inst), nil
}

// DestroyInstance stops a browser instance.
func (a *App) DestroyInstance(id string) error {
	return a.instanceSvc.DestroyInstance(a.appContext(), id)
}

// StopInstance stops a browser instance without deleting it.
func (a *App) StopInstance(id string) error {
	return a.instanceSvc.StopInstance(a.appContext(), id)
}

// RestartInstance restarts a stopped browser instance.
func (a *App) RestartInstance(id string) (*commands.BrowserInstance, error) {
	inst, err := a.instanceSvc.RestartInstance(a.appContext(), id)
	if err != nil {
		return nil, err
	}
	return commands.ToBrowserInstance(inst), nil
}

// DeleteInstance removes a stopped browser instance and its data.
func (a *App) DeleteInstance(id string) error {
	return a.instanceSvc.DeleteInstance(a.appContext(), id)
}

// GetInstance retrieves an instance by ID.
func (a *App) GetInstance(id string) (*commands.BrowserInstance, error) {
	inst, err := a.instanceSvc.GetInstance(a.appContext(), id)
	if err != nil {
		return nil, err
	}
	return commands.ToBrowserInstance(inst), nil
}

// ListInstances returns all instances matching the filter.
func (a *App) ListInstances(filter *commands.InstanceFilter) ([]*commands.BrowserInstance, error) {
	instances, err := a.instanceSvc.ListInstances(a.appContext(), filter)
	if err != nil {
		return nil, err
	}
	result := commands.ToBrowserInstances(instances)

	accounts, err := a.accountSvc.ListAccounts(a.appContext())
	if err != nil {
		return nil, err
	}
	accountMap := make(map[string]*commands.Account, len(accounts))
	for _, account := range accounts {
		accountMap[account.ID] = account
	}

	for _, inst := range result {
		if account, ok := accountMap[inst.AccountID]; ok {
			inst.AccountLabel = account.Label
			inst.ProxyURL = account.ProxyURL
		}
	}

	return result, nil
}

// ==================== Tab Commands ====================

// CreateTab creates a new tab with the specified fingerprint in an existing instance
func (a *App) CreateTab(instanceID string, cfg *commands.TabConfig) (*commands.TabInfo, error) {
	tabSvc := commands.NewTabService(a.instanceSvc)
	return tabSvc.CreateTab(a.appContext(), instanceID, cfg)
}

// CloseTab closes a specific tab
func (a *App) CloseTab(instanceID, tabID string) error {
	tabSvc := commands.NewTabService(a.instanceSvc)
	return tabSvc.CloseTab(a.appContext(), instanceID, tabID)
}

// ListTabs lists all tabs in an instance
func (a *App) ListTabs(instanceID string) ([]*commands.TabInfo, error) {
	tabSvc := commands.NewTabService(a.instanceSvc)
	return tabSvc.ListTabs(a.appContext(), instanceID)
}

// NavigateTab navigates a specific tab to a URL
func (a *App) NavigateTab(instanceID, tabID, url string) error {
	tabSvc := commands.NewTabService(a.instanceSvc)
	return tabSvc.NavigateTab(a.appContext(), instanceID, tabID, url)
}

// ==================== Release Commands ====================

func (a *App) PromoteBrowserChannel(req *commands.ReleasePromotionRequest) (*commands.ReleasePromotionResult, error) {
	return a.releaseSvc.PromoteChannel(a.appContext(), req)
}

func (a *App) RollbackBrowserChannel(req *commands.ReleaseRollbackRequest) (*commands.ReleaseRollbackResult, error) {
	return a.releaseSvc.RollbackStable(a.appContext(), req)
}

// ==================== Fingerprint Commands ====================

// GenerateFingerprint generates a fingerprint with the given seed and country.
func (a *App) GenerateFingerprint(seed, country string) (*commands.Fingerprint, error) {
	return a.fingerprintSvc.GenerateFingerprint(a.appContext(), seed, country)
}

// GenerateRandomFingerprint generates a fingerprint with a random seed.
func (a *App) GenerateRandomFingerprint(country string) (*commands.Fingerprint, error) {
	return a.fingerprintSvc.GenerateRandomFingerprint(a.appContext(), country)
}

// ValidateFingerprint validates a fingerprint's consistency.
func (a *App) ValidateFingerprint(fp *commands.Fingerprint) error {
	return a.fingerprintSvc.ValidateFingerprint(a.appContext(), fp)
}

func (a *App) GetFingerprintCoverageReport() *commands.FingerprintCoverageReport {
	return commands.GetFingerprintCoverageReport()
}

// ==================== Remote Browser Commands ====================

// ConnectRemote connects to a remote browser instance.
func (a *App) ConnectRemote(host string, port int, binaryPath string) error {
	return a.remoteSvc.Connect(a.appContext(), host, port, binaryPath)
}

// DisconnectRemote disconnects from a remote browser instance.
func (a *App) DisconnectRemote(host string, port int) error {
	return a.remoteSvc.Disconnect(a.appContext(), host, port)
}

// ListRemoteTargets lists available CDP targets on a remote browser.
func (a *App) ListRemoteTargets(host string, port int) ([]*commands.CDPTarget, error) {
	return a.remoteSvc.ListTargets(a.appContext(), host, port)
}

// GetRemoteCDPEndpoint returns the CDP endpoint for a remote browser.
func (a *App) GetRemoteCDPEndpoint(host string, port int) (string, error) {
	return a.remoteSvc.GetCDPEndpoint(a.appContext(), host, port)
}

// ==================== Account Commands ====================

func (a *App) ListAccounts() ([]*commands.Account, error) {
	return a.accountSvc.ListAccounts(a.appContext())
}

func (a *App) CreateAccount(req *commands.AccountCreateRequest) (*commands.Account, error) {
	return a.accountSvc.CreateAccount(a.appContext(), req)
}

func (a *App) UpdateAccount(req *commands.AccountUpdateRequest) (*commands.Account, error) {
	return a.accountSvc.UpdateAccount(a.appContext(), req)
}

func (a *App) RestartAccountInstance(accountID string) (*commands.Account, error) {
	return a.accountSvc.RestartAccountInstance(a.appContext(), accountID)
}

func (a *App) BindAccountInstance(req *commands.AccountInstanceBindRequest) (*commands.BrowserInstance, error) {
	inst, err := a.accountSvc.BindAccountInstance(a.appContext(), req)
	if err != nil {
		return nil, err
	}
	return commands.ToBrowserInstance(inst), nil
}

func (a *App) DeleteAccount(accountID string) error {
	return a.accountSvc.DeleteAccount(a.appContext(), accountID)
}

func (a *App) CheckFingerprint(instanceID string) (*commands.FingerprintCheckResult, error) {
	return a.accountSvc.CheckFingerprint(a.appContext(), instanceID)
}

// ==================== Proxy Commands ====================

func (a *App) ListProxies() ([]*commands.ProxyDTO, error) {
	return a.proxySvc.ListProxies(a.appContext())
}

func (a *App) CreateProxy(req *commands.ProxyCreateRequest) (*commands.ProxyDTO, error) {
	return a.proxySvc.CreateProxy(a.appContext(), req)
}

func (a *App) DeleteProxy(proxyID string) error {
	return a.proxySvc.DeleteProxy(a.appContext(), proxyID)
}

// ==================== Fingerprint Server Commands ====================

// StartFingerprintServer starts the mock fingerprint server.
func (a *App) StartFingerprintServer() error {
	return a.fpServerSvc.StartServer(a.appContext())
}

// StopFingerprintServer stops the mock fingerprint server.
func (a *App) StopFingerprintServer() error {
	return a.fpServerSvc.StopServer()
}

// GetFingerprintServerStatus returns the status of the fingerprint server.
func (a *App) GetFingerprintServerStatus() *commands.FingerprintServerStatus {
	status := a.fpServerSvc.GetStatus()
	return &status
}

// LaunchFingerprintBrowser launches a browser with the fingerprint test page.
func (a *App) LaunchFingerprintBrowser(browserBinaryPath string) (int, error) {
	return a.fpServerSvc.LaunchBrowser(a.appContext(), browserBinaryPath)
}

// CollectFingerprint collects fingerprint data from the mock server.
func (a *App) CollectFingerprint() (*commands.FingerprintVerificationResult, error) {
	return a.fpServerSvc.CollectFingerprint(a.appContext())
}

// RunFingerprintVerification runs a full fingerprint verification.
func (a *App) RunFingerprintVerification() (*commands.FingerprintVerificationResult, error) {
	// First collect the fingerprint
	result, err := a.fpServerSvc.CollectFingerprint(a.appContext())
	if err != nil {
		return nil, err
	}
	return result, nil
}

// NavigateInstanceBrowser navigates a browser instance to the specified URL.
func (a *App) NavigateInstanceBrowser(instanceID string, url string) error {
	log.Printf("[NavigateInstanceBrowser] instanceID=%s, url=%s", instanceID, url)
	cdpClient, err := a.instanceSvc.GetCDPClient(a.appContext(), instanceID)
	if err != nil {
		log.Printf("[NavigateInstanceBrowser] GetCDPClient error: %v", err)
		return fmt.Errorf("failed to connect to browser: %v", err)
	}
	log.Printf("[NavigateInstanceBrowser] CDP client acquired, calling Navigate...")
	err = cdpClient.Navigate(a.appContext(), url)
	if err != nil {
		log.Printf("[NavigateInstanceBrowser] Navigate error: %v", err)
	} else {
		log.Printf("[NavigateInstanceBrowser] Success")
	}
	a.instanceSvc.CloseCDPClient(instanceID)
	return err
}

// NavigateInstanceBrowserNewTab opens a URL in a new tab of the browser instance.
func (a *App) NavigateInstanceBrowserNewTab(instanceID string, url string) error {
	log.Printf("[NavigateInstanceBrowserNewTab] instanceID=%s, url=%s", instanceID, url)
	cdpClient, err := a.instanceSvc.GetCDPClient(a.appContext(), instanceID)
	if err != nil {
		log.Printf("[NavigateInstanceBrowserNewTab] GetCDPClient error: %v", err)
		return fmt.Errorf("failed to connect to browser: %v", err)
	}

	// Use Target.createTarget to create a new tab via CDP (not JS window.open)
	targetID, err := cdpClient.CreateTarget(a.appContext(), url)
	if err != nil {
		log.Printf("[NavigateInstanceBrowserNewTab] CreateTarget error: %v", err)
		// Fallback: try window.open
		_, evalErr := cdpClient.Evaluate(a.appContext(), fmt.Sprintf(`window.open("%s", "_blank");`, url))
		if evalErr != nil {
			log.Printf("[NavigateInstanceBrowserNewTab] Evaluate fallback error: %v", evalErr)
		}
	} else {
		log.Printf("[NavigateInstanceBrowserNewTab] Created new target: %s", targetID)
	}
	log.Printf("[NavigateInstanceBrowserNewTab] Success")
	a.instanceSvc.CloseCDPClient(instanceID)
	return nil
}
