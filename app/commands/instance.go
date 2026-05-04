package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type InstanceService struct {
	manager    browserRuntimeManager
	store      *sqlite.InstanceStore
	cdpClients sync.Map // instanceID -> CDPClient
}

func NewInstanceService(db *sqlite.DB) *InstanceService {
	manager := newBrowserRuntimeManager(db)
	store := sqlite.NewInstanceStore(db)
	return &InstanceService{
		manager: manager,
		store:   store,
	}
}

func (s *InstanceService) CreateInstance(ctx context.Context, cfg *InstanceConfig) (*instance.BrowserInstance, error) {
	return s.manager.Start(ctx, cfg)
}

func (s *InstanceService) DestroyInstance(ctx context.Context, id string) error {
	return s.manager.Stop(ctx, id)
}

func (s *InstanceService) StopInstance(ctx context.Context, id string) error {
	return s.manager.Stop(ctx, id)
}

func (s *InstanceService) RestartInstance(ctx context.Context, id string) (*instance.BrowserInstance, error) {
	inst, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	cfg := &instance.InstanceConfig{
		Name:         inst.Name,
		AccountLabel: "",
		Fingerprint:  inst.Fingerprint,
		Proxy:        nil,
		AccountID:    inst.AccountID,
		Group:        inst.Group,
		Headless:     inst.Headless,
	}
	if inst.ProxyID != "" || inst.ProxyURL != "" {
		cfg.Proxy = &instance.ProxyConfig{ID: inst.ProxyID, URL: inst.ProxyURL}
	}
	return s.RestartInstanceWithConfig(ctx, id, cfg)
}

func (s *InstanceService) DeleteInstance(ctx context.Context, id string) error {
	inst, err := s.store.Get(id)
	if err != nil {
		return err
	}
	if inst.Status != instance.StatusStopped {
		return fmt.Errorf("instance must be stopped before delete")
	}
	return s.manager.Delete(id)
}

func (s *InstanceService) RestartInstanceWithConfig(ctx context.Context, id string, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error) {
	inst, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if inst.Status != instance.StatusStopped {
		if err := s.manager.Stop(ctx, id); err != nil {
			return nil, err
		}
		inst, err = s.store.Get(id)
		if err != nil {
			return nil, err
		}
	}
	return s.manager.Restart(ctx, inst, cfg)
}

func (s *InstanceService) GetInstance(ctx context.Context, id string) (*instance.BrowserInstance, error) {
	return s.store.Get(id)
}

func (s *InstanceService) ListInstances(ctx context.Context, filter *instance.InstanceFilter) ([]*instance.BrowserInstance, error) {
	return s.store.List(filter)
}

func (s *InstanceService) GetCDPClient(ctx context.Context, id string) (instance.CDPClientInterface, error) {
	if client, ok := s.cdpClients.Load(id); ok {
		return client.(instance.CDPClientInterface), nil
	}

	inst, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}

	if inst.Status != instance.StatusRunning {
		return nil, instance.ErrInstanceNotRunning
	}

	conn, _, err := instance.DefaultDialer.DialContext(ctx, "tcp", inst.CDPEndpoint)
	if err != nil {
		return nil, err
	}

	client := instance.NewCDPClient(conn)
	s.cdpClients.Store(id, client)
	return client, nil
}

func (s *InstanceService) CloseCDPClient(id string) error {
	if client, ok := s.cdpClients.LoadAndDelete(id); ok {
		return client.(instance.CDPClientInterface).Close()
	}
	return nil
}

type InstanceConfig = instance.InstanceConfig
type InstanceFilter = instance.InstanceFilter
type InstanceStatus = instance.InstanceStatus

type BrowserInstance struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Status       instance.InstanceStatus  `json:"status"`
	Fingerprint  *fingerprint.Fingerprint `json:"fingerprint"`
	ProxyID      string                   `json:"proxy_id"`
	ProxyURL     string                   `json:"proxy_url"`
	AccountID    string                   `json:"account_id"`
	AccountLabel string                   `json:"account_label"`
	CDPEndpoint  string                   `json:"cdp_endpoint"`
	PID          int                      `json:"pid"`
	Port         int                      `json:"port"`
	UserDataDir  string                   `json:"user_data_dir"`
	Group        string                   `json:"group"`
	Headless     bool                     `json:"headless"`
	StartedAt    string                   `json:"started_at"`
	LastActiveAt string                   `json:"last_active_at"`
	CreatedAt    string                   `json:"created_at"`
}

func ToBrowserInstance(inst *instance.BrowserInstance) *BrowserInstance {
	if inst == nil {
		return nil
	}
	return &BrowserInstance{
		ID:           inst.ID,
		Name:         inst.Name,
		Status:       inst.Status,
		Fingerprint:  inst.Fingerprint,
		ProxyID:      inst.ProxyID,
		ProxyURL:     inst.ProxyURL,
		AccountID:    inst.AccountID,
		CDPEndpoint:  inst.CDPEndpoint,
		PID:          inst.PID,
		Port:         inst.Port,
		UserDataDir:  inst.UserDataDir,
		Group:        inst.Group,
		Headless:     inst.Headless,
		StartedAt:    inst.StartedAt.Format(time.RFC3339Nano),
		LastActiveAt: inst.LastActiveAt.Format(time.RFC3339Nano),
		CreatedAt:    inst.CreatedAt.Format(time.RFC3339Nano),
	}
}

func ToBrowserInstances(list []*instance.BrowserInstance) []*BrowserInstance {
	if len(list) == 0 {
		return []*BrowserInstance{}
	}
	result := make([]*BrowserInstance, 0, len(list))
	for _, inst := range list {
		result = append(result, ToBrowserInstance(inst))
	}
	return result
}

// InstanceStatus constants
const (
	StatusPending  = instance.StatusPending
	StatusStarting = instance.StatusStarting
	StatusRunning  = instance.StatusRunning
	StatusStopping = instance.StatusStopping
	StatusStopped  = instance.StatusStopped
	StatusError    = instance.StatusError
)
