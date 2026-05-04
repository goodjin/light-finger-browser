package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/proxy"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type ProxyService struct {
	proxyStore    *sqlite.ProxyStore
	accountStore  *sqlite.AccountStore
	instanceStore *sqlite.InstanceStore
}

func NewProxyService(db *sqlite.DB) *ProxyService {
	return &ProxyService{
		proxyStore:    sqlite.NewProxyStore(db),
		accountStore:  sqlite.NewAccountStore(db),
		instanceStore: sqlite.NewInstanceStore(db),
	}
}

type ProxyDTO struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	BindID    string `json:"bind_id"`
	Country   string `json:"country"`
	Type      string `json:"type"`
	Provider  string `json:"provider"`
	CreatedAt string `json:"created_at"`
}

type ProxyCreateRequest struct {
	URL      string `json:"url"`
	Country  string `json:"country"`
	Type     string `json:"type"`
	Provider string `json:"provider"`
}

func (s *ProxyService) ListProxies(ctx context.Context) ([]*ProxyDTO, error) {
	proxies, err := s.proxyStore.List()
	if err != nil {
		return nil, err
	}
	result := make([]*ProxyDTO, 0, len(proxies))
	for _, p := range proxies {
		result = append(result, toProxyDTO(p))
	}
	return result, nil
}

func (s *ProxyService) CreateProxy(ctx context.Context, req *ProxyCreateRequest) (*ProxyDTO, error) {
	raw := strings.TrimSpace(req.URL)
	if raw == "" {
		return nil, fmt.Errorf("proxy URL is required")
	}
	host, port, username, password, err := parseProxyURL(raw)
	if err != nil {
		return nil, err
	}

	proxyType := proxy.ProxyTypeResidential
	if strings.EqualFold(req.Type, string(proxy.ProxyTypeDatacenter)) {
		proxyType = proxy.ProxyTypeDatacenter
	}
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = "manual"
	}
	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = "unknown"
	}

	p := &proxy.Proxy{
		ID:        uuid.New().String(),
		IP:        host,
		Port:      port,
		Country:   country,
		Type:      proxyType,
		Status:    proxy.ProxyStatusAvailable,
		BindID:    "",
		Provider:  provider,
		Username:  username,
		Password:  password,
		CreatedAt: time.Now(),
	}
	created, err := s.proxyStore.Save(p)
	if err != nil {
		return nil, err
	}
	return toProxyDTO(created), nil
}

func (s *ProxyService) DeleteProxy(ctx context.Context, proxyID string) error {
	instances, err := s.instanceStore.List(&instance.InstanceFilter{ProxyID: proxyID})
	if err != nil {
		return err
	}
	for _, inst := range instances {
		if inst.Status != instance.StatusStopped {
			return fmt.Errorf("proxy is still bound to a running instance")
		}
	}

	accounts, err := s.accountStore.List()
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if account.ProxyID != proxyID {
			continue
		}
		if account.InstanceID != "" {
			inst, err := s.instanceStore.Get(account.InstanceID)
			if err == nil && inst.Status != instance.StatusStopped {
				return fmt.Errorf("proxy is still bound to a running instance")
			}
		}
		account.ProxyID = ""
		account.ProxyURL = ""
		account.PendingRestart = false
		if err := s.accountStore.Update(account); err != nil {
			return err
		}
	}

	return s.proxyStore.Delete(proxyID)
}

func toProxyDTO(p *proxy.Proxy) *ProxyDTO {
	return &ProxyDTO{
		ID:        p.ID,
		URL:       formatProxyURL(p),
		Status:    string(p.Status),
		BindID:    p.BindID,
		Country:   p.Country,
		Type:      string(p.Type),
		Provider:  p.Provider,
		CreatedAt: p.CreatedAt.Format(time.RFC3339Nano),
	}
}

func formatProxyURL(p *proxy.Proxy) string {
	host := fmt.Sprintf("%s:%d", p.IP, p.Port)
	if p.Username == "" {
		return host
	}
	if p.Password != "" {
		return fmt.Sprintf("%s:%s@%s", p.Username, p.Password, host)
	}
	return fmt.Sprintf("%s@%s", p.Username, host)
}
