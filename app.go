package main

import (
	"context"
	"log"
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
}

func NewApp() *App {
	return &App{}
}

func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	log.Println("fingerbrower starting...")

	// Initialize SQLite database
	dbPath := "fingerbrower.db"
	var err error
	a.db, err = sqlite.NewDB(dbPath)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return
	}

	// Run migrations
	if err := a.db.Migrate(); err != nil {
		log.Printf("Failed to run migrations: %v", err)
		return
	}

	// Initialize services
	a.instanceSvc = commands.NewInstanceService(a.db)
	a.fingerprintSvc = commands.NewFingerprintService()
	a.remoteSvc = commands.NewRemoteBrowserService()
	a.accountSvc = commands.NewAccountService(a.db, a.instanceSvc)
	a.proxySvc = commands.NewProxyService(a.db)
	a.releaseSvc = commands.NewReleaseService()

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

// ConnectRemote connects to a remote CloakBrowser instance.
func (a *App) ConnectRemote(host string, port int, binaryPath string) error {
	return a.remoteSvc.Connect(a.appContext(), host, port, binaryPath)
}

// DisconnectRemote disconnects from a remote CloakBrowser instance.
func (a *App) DisconnectRemote(host string, port int) error {
	return a.remoteSvc.Disconnect(a.appContext(), host, port)
}

// ListRemoteTargets lists available CDP targets on a remote CloakBrowser.
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
