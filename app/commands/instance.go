package commands

import (
	"context"
	"sync"

	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type InstanceService struct {
	manager    *LocalChromeManager
	store      *sqlite.InstanceStore
	cdpClients sync.Map // instanceID -> CDPClient
}

func NewInstanceService(db *sqlite.DB) *InstanceService {
	manager := NewLocalChromeManager(db)
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
type BrowserInstance = instance.BrowserInstance
type InstanceFilter = instance.InstanceFilter
type InstanceStatus = instance.InstanceStatus

// InstanceStatus constants
const (
	StatusPending  = instance.StatusPending
	StatusStarting = instance.StatusStarting
	StatusRunning  = instance.StatusRunning
	StatusStopping = instance.StatusStopping
	StatusStopped  = instance.StatusStopped
	StatusError    = instance.StatusError
)
