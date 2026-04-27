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

// ==================== Instance Commands ====================

// CreateInstance creates a new browser instance with the given configuration.
func (a *App) CreateInstance(ctx context.Context, cfg *commands.InstanceConfig) (*commands.BrowserInstance, error) {
	return a.instanceSvc.CreateInstance(ctx, cfg)
}

// DestroyInstance stops and removes a browser instance.
func (a *App) DestroyInstance(ctx context.Context, id string) error {
	return a.instanceSvc.DestroyInstance(ctx, id)
}

// GetInstance retrieves an instance by ID.
func (a *App) GetInstance(ctx context.Context, id string) (*commands.BrowserInstance, error) {
	return a.instanceSvc.GetInstance(ctx, id)
}

// ListInstances returns all instances matching the filter.
func (a *App) ListInstances(ctx context.Context, filter *commands.InstanceFilter) ([]*commands.BrowserInstance, error) {
	return a.instanceSvc.ListInstances(ctx, filter)
}

// ==================== Fingerprint Commands ====================

// GenerateFingerprint generates a fingerprint with the given seed and country.
func (a *App) GenerateFingerprint(ctx context.Context, seed, country string) (*commands.Fingerprint, error) {
	return a.fingerprintSvc.GenerateFingerprint(ctx, seed, country)
}

// GenerateRandomFingerprint generates a fingerprint with a random seed.
func (a *App) GenerateRandomFingerprint(ctx context.Context, country string) (*commands.Fingerprint, error) {
	return a.fingerprintSvc.GenerateRandomFingerprint(ctx, country)
}

// ValidateFingerprint validates a fingerprint's consistency.
func (a *App) ValidateFingerprint(ctx context.Context, fp *commands.Fingerprint) error {
	return a.fingerprintSvc.ValidateFingerprint(ctx, fp)
}

// ==================== Remote Browser Commands ====================

// ConnectRemote connects to a remote CloakBrowser instance.
func (a *App) ConnectRemote(ctx context.Context, host string, port int, binaryPath string) error {
	return a.remoteSvc.Connect(ctx, host, port, binaryPath)
}

// DisconnectRemote disconnects from a remote CloakBrowser instance.
func (a *App) DisconnectRemote(ctx context.Context, host string, port int) error {
	return a.remoteSvc.Disconnect(ctx, host, port)
}

// ListRemoteTargets lists available CDP targets on a remote CloakBrowser.
func (a *App) ListRemoteTargets(ctx context.Context, host string, port int) ([]*commands.CDPTarget, error) {
	return a.remoteSvc.ListTargets(ctx, host, port)
}

// GetRemoteCDPEndpoint returns the CDP endpoint for a remote browser.
func (a *App) GetRemoteCDPEndpoint(ctx context.Context, host string, port int) (string, error) {
	return a.remoteSvc.GetCDPEndpoint(ctx, host, port)
}
