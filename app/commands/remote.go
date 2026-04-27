package commands

import (
	"context"
	"fmt"
	"sync"

	"github.com/tmos/fingerbrower/cloakbrowser"
)

type RemoteBrowserService struct {
	mu      sync.Mutex
	clients sync.Map // key: "host:port", value: *cloakbrowser.Client
}

func NewRemoteBrowserService() *RemoteBrowserService {
	return &RemoteBrowserService{}
}

func (s *RemoteBrowserService) Connect(ctx context.Context, host string, port int, binaryPath string) error {
	key := fmt.Sprintf("%s:%d", host, port)

	client, err := cloakbrowser.NewClient(binaryPath, port)
	if err != nil {
		return fmt.Errorf("failed to create CloakBrowser client: %w", err)
	}
	if err := client.Start(ctx, nil); err != nil {
		return fmt.Errorf("failed to connect to remote CloakBrowser at %s:%d: %w", host, port, err)
	}

	s.clients.Store(key, client)
	return nil
}

func (s *RemoteBrowserService) Disconnect(ctx context.Context, host string, port int) error {
	key := fmt.Sprintf("%s:%d", host, port)

	client, ok := s.clients.Load(key)
	if !ok {
		return fmt.Errorf("not connected to %s:%d", host, port)
	}

	if err := client.(*cloakbrowser.Client).Stop(); err != nil {
		return err
	}

	s.clients.Delete(key)
	return nil
}

func (s *RemoteBrowserService) ListTargets(ctx context.Context, host string, port int) ([]*cloakbrowser.CDPTarget, error) {
	key := fmt.Sprintf("%s:%d", host, port)

	client, ok := s.clients.Load(key)
	if !ok {
		return nil, fmt.Errorf("not connected to %s:%d", host, port)
	}

	return client.(*cloakbrowser.Client).ListTargets()
}

func (s *RemoteBrowserService) GetCDPEndpoint(ctx context.Context, host string, port int) (string, error) {
	key := fmt.Sprintf("%s:%d", host, port)

	client, ok := s.clients.Load(key)
	if !ok {
		return "", fmt.Errorf("not connected to %s:%d", host, port)
	}

	return client.(*cloakbrowser.Client).GetCDPEndpoint(), nil
}

type CDPTarget = cloakbrowser.CDPTarget
