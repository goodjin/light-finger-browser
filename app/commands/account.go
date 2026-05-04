package commands

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/proxy"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type AccountService struct {
	accountStore  *sqlite.AccountStore
	proxyStore    *sqlite.ProxyStore
	instanceStore *sqlite.InstanceStore
	snapshotStore *sqlite.FingerprintSnapshotStore
	instanceSvc   *InstanceService
	fpGenerator   *fingerprint.GeneratorWithValidator
}

func NewAccountService(db *sqlite.DB, instanceSvc *InstanceService) *AccountService {
	return &AccountService{
		accountStore:  sqlite.NewAccountStore(db),
		proxyStore:    sqlite.NewProxyStore(db),
		instanceStore: sqlite.NewInstanceStore(db),
		snapshotStore: sqlite.NewFingerprintSnapshotStore(db),
		instanceSvc:   instanceSvc,
		fpGenerator:   fingerprint.NewGeneratorWithValidator(),
	}
}

type Account struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Label              string `json:"label"`
	Status             string `json:"status"`
	Group              string `json:"group"`
	InstanceID         string `json:"instance_id"`
	InstanceName       string `json:"instance_name"`
	InstanceStatus     string `json:"instance_status"`
	ProxyID            string `json:"proxy_id"`
	ProxyURL           string `json:"proxy_url"`
	FingerprintSeed    string `json:"fingerprint_seed"`
	FingerprintCountry string `json:"fingerprint_country"`
	Headless           bool   `json:"headless"`
	PendingRestart     bool   `json:"pending_restart"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type AccountCreateRequest struct {
	Email              string `json:"email"`
	Username           string `json:"username"`
	Group              string `json:"group"`
	InstanceName       string `json:"instance_name"`
	ProxyURL           string `json:"proxy_url"`
	FingerprintSeed    string `json:"fingerprint_seed"`
	FingerprintCountry string `json:"fingerprint_country"`
	Headless           bool   `json:"headless"`
}

type AccountUpdateRequest struct {
	ID                 string `json:"id"`
	Email              string `json:"email"`
	Username           string `json:"username"`
	Group              string `json:"group"`
	InstanceName       string `json:"instance_name"`
	ProxyURL           string `json:"proxy_url"`
	FingerprintSeed    string `json:"fingerprint_seed"`
	FingerprintCountry string `json:"fingerprint_country"`
	Headless           bool   `json:"headless"`
	Restart            bool   `json:"restart"`
}

type AccountInstanceBindRequest struct {
	AccountID    string `json:"account_id"`
	InstanceName string `json:"instance_name"`
	Group        string `json:"group"`
	Headless     bool   `json:"headless"`
	ProxyID      string `json:"proxy_id"`
	ProxyURL     string `json:"proxy_url"`
}

func (s *AccountService) ListAccounts(ctx context.Context) ([]*Account, error) {
	accounts, err := s.accountStore.List()
	if err != nil {
		return nil, err
	}
	result := make([]*Account, 0, len(accounts))
	for _, acc := range accounts {
		result = append(result, s.toAccountDTO(acc))
	}
	return result, nil
}

func (s *AccountService) CreateAccount(ctx context.Context, req *AccountCreateRequest) (*Account, error) {
	accountID := uuid.New().String()
	email := strings.TrimSpace(req.Email)
	if email == "" {
		email = fmt.Sprintf("auto-%s@local", accountID)
	}
	accountLabel := formatAccountLabel(req.Username, email)
	if accountLabel == "Account" {
		accountLabel = fmt.Sprintf("Account-%s", accountID[:8])
	}
	instanceName := strings.TrimSpace(req.InstanceName)
	if instanceName == "" {
		instanceName = accountLabel
	}

	seed := strings.TrimSpace(req.FingerprintSeed)
	country := strings.TrimSpace(req.FingerprintCountry)
	if country == "" {
		country = "US"
	}

	var fp *fingerprint.Fingerprint
	var err error
	if seed == "" {
		fp, err = s.fpGenerator.GenerateRandom(country)
		if err != nil {
			return nil, err
		}
		seed = fp.Seed
	} else {
		fp, err = s.fpGenerator.Generate(seed, country)
		if err != nil {
			return nil, err
		}
	}

	proxyID, proxyCfg, err := s.prepareProxy(accountID, req.ProxyURL)
	if err != nil {
		return nil, err
	}

	cfg := &instance.InstanceConfig{
		Name:         instanceName,
		AccountLabel: accountLabel,
		Fingerprint:  fp,
		Proxy:        proxyCfg,
		AccountID:    accountID,
		Group:        req.Group,
		Headless:     req.Headless,
	}

	inst, err := s.instanceSvc.CreateInstance(ctx, cfg)
	if err != nil {
		return nil, err
	}

	account := &sqlite.Account{
		ID:                 accountID,
		Username:           req.Username,
		Email:              email,
		Status:             "active",
		AccountLevel:       0,
		Group:              req.Group,
		InstanceID:         inst.ID,
		InstanceName:       instanceName,
		ProxyID:            proxyID,
		ProxyURL:           strings.TrimSpace(req.ProxyURL),
		FingerprintSeed:    seed,
		FingerprintCountry: country,
		Headless:           req.Headless,
		PendingRestart:     false,
	}

	if _, err := s.accountStore.Save(account); err != nil {
		return nil, err
	}

	return s.toAccountDTO(account), nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, req *AccountUpdateRequest) (*Account, error) {
	account, err := s.accountStore.Get(req.ID)
	if err != nil {
		return nil, err
	}
	previousInstanceName := account.InstanceName

	requiresRestart := account.ProxyURL != req.ProxyURL ||
		account.FingerprintSeed != req.FingerprintSeed ||
		account.FingerprintCountry != req.FingerprintCountry ||
		account.Headless != req.Headless

	account.Username = req.Username
	if strings.TrimSpace(req.Email) != "" {
		account.Email = strings.TrimSpace(req.Email)
	}
	account.Group = req.Group
	account.InstanceName = strings.TrimSpace(req.InstanceName)
	if account.InstanceName == "" {
		account.InstanceName = formatAccountLabel(account.Username, account.Email)
	}
	account.ProxyURL = strings.TrimSpace(req.ProxyURL)
	account.FingerprintSeed = strings.TrimSpace(req.FingerprintSeed)
	account.FingerprintCountry = strings.TrimSpace(req.FingerprintCountry)
	if account.FingerprintCountry == "" {
		account.FingerprintCountry = "US"
	}
	if account.FingerprintSeed == "" {
		fp, err := s.fpGenerator.GenerateRandom(account.FingerprintCountry)
		if err != nil {
			return nil, err
		}
		account.FingerprintSeed = fp.Seed
		requiresRestart = true
	}
	account.Headless = req.Headless

	if requiresRestart {
		account.PendingRestart = !req.Restart
	} else {
		account.PendingRestart = false
	}

	if err := s.accountStore.Update(account); err != nil {
		return nil, err
	}

	if account.InstanceID != "" && previousInstanceName != account.InstanceName {
		inst, err := s.instanceStore.Get(account.InstanceID)
		if err != nil {
			return nil, err
		}
		inst.Name = account.InstanceName
		if err := s.instanceStore.Update(inst); err != nil {
			return nil, err
		}
	}

	if requiresRestart && req.Restart {
		if err := s.restartInstanceForAccount(ctx, account); err != nil {
			account.PendingRestart = true
			_ = s.accountStore.Update(account)
			return nil, err
		}
	}

	return s.toAccountDTO(account), nil
}

func (s *AccountService) RestartAccountInstance(ctx context.Context, accountID string) (*Account, error) {
	account, err := s.accountStore.Get(accountID)
	if err != nil {
		return nil, err
	}
	if err := s.restartInstanceForAccount(ctx, account); err != nil {
		return nil, err
	}
	return s.toAccountDTO(account), nil
}

func (s *AccountService) BindAccountInstance(ctx context.Context, req *AccountInstanceBindRequest) (*instance.BrowserInstance, error) {
	if strings.TrimSpace(req.AccountID) == "" {
		return nil, fmt.Errorf("account ID is required")
	}
	account, err := s.accountStore.Get(req.AccountID)
	if err != nil {
		return nil, err
	}

	if req.Group != "" || account.Group == "" {
		account.Group = strings.TrimSpace(req.Group)
	}
	account.Headless = req.Headless

	if strings.TrimSpace(req.InstanceName) != "" {
		account.InstanceName = strings.TrimSpace(req.InstanceName)
	}
	if account.InstanceName == "" {
		account.InstanceName = formatAccountLabel(account.Username, account.Email)
	}

	desiredProxyID, desiredProxyURL, err := s.resolveProxySelection(account.ID, req.ProxyID, req.ProxyURL, account.ProxyID, account.ProxyURL)
	if err != nil {
		return nil, err
	}

	fp, err := s.fpGenerator.Generate(account.FingerprintSeed, account.FingerprintCountry)
	if err != nil {
		fp, err = s.fpGenerator.GenerateRandom(account.FingerprintCountry)
		if err != nil {
			return nil, err
		}
		account.FingerprintSeed = fp.Seed
	}

	var proxyCfg *instance.ProxyConfig
	if desiredProxyURL != "" {
		proxyCfg = &instance.ProxyConfig{ID: desiredProxyID, URL: desiredProxyURL}
	}

	cfg := &instance.InstanceConfig{
		Name:         account.InstanceName,
		AccountLabel: formatAccountLabel(account.Username, account.Email),
		Fingerprint:  fp,
		Proxy:        proxyCfg,
		AccountID:    account.ID,
		Group:        account.Group,
		Headless:     account.Headless,
	}

	oldProxyID := account.ProxyID

	var inst *instance.BrowserInstance
	if account.InstanceID != "" {
		existing, err := s.instanceStore.Get(account.InstanceID)
		if err != nil {
			return nil, err
		}
		if existing.Status != instance.StatusStopped {
			return nil, fmt.Errorf("account instance must be stopped before binding")
		}
		inst, err = s.instanceSvc.RestartInstanceWithConfig(ctx, account.InstanceID, cfg)
		if err != nil {
			return nil, err
		}
	} else {
		inst, err = s.instanceSvc.CreateInstance(ctx, cfg)
		if err != nil {
			return nil, err
		}
		account.InstanceID = inst.ID
	}

	account.ProxyID = desiredProxyID
	account.ProxyURL = desiredProxyURL
	account.PendingRestart = false
	if err := s.accountStore.Update(account); err != nil {
		return nil, err
	}

	if oldProxyID != "" && oldProxyID != account.ProxyID {
		_ = s.releaseProxy(oldProxyID)
	}

	return inst, nil
}

func (s *AccountService) DeleteAccount(ctx context.Context, accountID string) error {
	account, err := s.accountStore.Get(accountID)
	if err != nil {
		return err
	}

	if account.InstanceID != "" {
		inst, err := s.instanceStore.Get(account.InstanceID)
		if err == nil && inst.Status != instance.StatusStopped {
			return fmt.Errorf("account instance must be stopped before delete")
		}
		if err := s.instanceSvc.DeleteInstance(ctx, account.InstanceID); err != nil {
			return err
		}
	}

	if account.ProxyID != "" {
		_ = s.releaseProxy(account.ProxyID)
	}

	return s.accountStore.Delete(accountID)
}

func (s *AccountService) toAccountDTO(account *sqlite.Account) *Account {
	label := formatAccountLabel(account.Username, account.Email)
	if label == "Account" {
		if account.InstanceName != "" {
			label = account.InstanceName
		} else {
			label = account.ID
		}
	}
	dto := &Account{
		ID:                 account.ID,
		Username:           account.Username,
		Email:              account.Email,
		Label:              label,
		Status:             account.Status,
		Group:              account.Group,
		InstanceID:         account.InstanceID,
		InstanceName:       account.InstanceName,
		ProxyID:            account.ProxyID,
		ProxyURL:           account.ProxyURL,
		FingerprintSeed:    account.FingerprintSeed,
		FingerprintCountry: account.FingerprintCountry,
		Headless:           account.Headless,
		PendingRestart:     account.PendingRestart,
		CreatedAt:          account.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:          account.UpdatedAt.Format(time.RFC3339Nano),
	}

	if account.InstanceID != "" {
		if inst, err := s.instanceStore.Get(account.InstanceID); err == nil {
			dto.InstanceStatus = string(inst.Status)
		}
	}

	return dto
}

func formatAccountLabel(username string, email string) string {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username != "" && email != "" {
		return fmt.Sprintf("%s (%s)", username, email)
	}
	if username != "" {
		return username
	}
	if email != "" {
		return email
	}
	return "Account"
}

func (s *AccountService) restartInstanceForAccount(ctx context.Context, account *sqlite.Account) error {
	if account.InstanceID == "" {
		return fmt.Errorf("account has no instance to restart")
	}

	fp, err := s.fpGenerator.Generate(account.FingerprintSeed, account.FingerprintCountry)
	if err != nil {
		fp, err = s.fpGenerator.GenerateRandom(account.FingerprintCountry)
		if err != nil {
			return err
		}
		account.FingerprintSeed = fp.Seed
	}

	if strings.TrimSpace(account.InstanceName) == "" {
		account.InstanceName = formatAccountLabel(account.Username, account.Email)
	}

	oldProxyID := account.ProxyID
	proxyID, proxyCfg, err := s.resolveProxy(account.ID, account.ProxyURL, account.ProxyID)
	if err != nil {
		return err
	}

	cfg := &instance.InstanceConfig{
		Name:         account.InstanceName,
		AccountLabel: formatAccountLabel(account.Username, account.Email),
		Fingerprint:  fp,
		Proxy:        proxyCfg,
		AccountID:    account.ID,
		Group:        account.Group,
		Headless:     account.Headless,
	}

	newInst, err := s.instanceSvc.RestartInstanceWithConfig(ctx, account.InstanceID, cfg)
	if err != nil {
		return err
	}

	account.InstanceID = newInst.ID
	account.ProxyID = proxyID
	account.PendingRestart = false
	if err := s.accountStore.Update(account); err != nil {
		return err
	}

	if oldProxyID != "" && oldProxyID != account.ProxyID {
		_ = s.releaseProxy(oldProxyID)
	}
	return nil
}

func (s *AccountService) prepareProxy(bindID string, proxyURL string) (string, *instance.ProxyConfig, error) {
	raw := strings.TrimSpace(proxyURL)
	if raw == "" {
		return "", nil, nil
	}

	host, port, username, password, err := parseProxyURL(raw)
	if err != nil {
		return "", nil, err
	}

	proxyID := uuid.New().String()
	p := &proxy.Proxy{
		ID:       proxyID,
		IP:       host,
		Port:     port,
		Country:  "unknown",
		Type:     proxy.ProxyTypeResidential,
		Status:   proxy.ProxyStatusInUse,
		BindID:   bindID,
		Provider: "manual",
		Username: username,
		Password: password,
	}
	if _, err := s.proxyStore.Save(p); err != nil {
		return "", nil, err
	}

	return proxyID, &instance.ProxyConfig{
		ID:  proxyID,
		URL: raw,
	}, nil
}

func (s *AccountService) resolveProxy(bindID string, proxyURL string, existingProxyID string) (string, *instance.ProxyConfig, error) {
	raw := strings.TrimSpace(proxyURL)
	if raw == "" {
		return "", nil, nil
	}
	if existingProxyID != "" {
		if existing, err := s.proxyStore.Get(existingProxyID); err == nil {
			if proxyMatchesURL(existing, raw) {
				if err := s.claimProxy(existing, bindID); err != nil {
					return "", nil, err
				}
				return existingProxyID, &instance.ProxyConfig{ID: existingProxyID, URL: raw}, nil
			}
		}
	}
	return s.prepareProxy(bindID, raw)
}

func (s *AccountService) releaseProxy(proxyID string) error {
	if proxyID == "" {
		return nil
	}
	p, err := s.proxyStore.Get(proxyID)
	if err != nil {
		return err
	}
	p.Status = proxy.ProxyStatusAvailable
	p.BindID = ""
	return s.proxyStore.Update(p)
}

func (s *AccountService) claimProxy(p *proxy.Proxy, bindID string) error {
	if p == nil {
		return nil
	}
	if p.BindID != "" && p.BindID != bindID && s.isProxyBindTargetRunning(p.BindID) {
		return fmt.Errorf("proxy is already bound to another running instance")
	}
	p.Status = proxy.ProxyStatusInUse
	p.BindID = bindID
	return s.proxyStore.Update(p)
}

func (s *AccountService) isProxyBindTargetRunning(bindID string) bool {
	if bindID == "" {
		return false
	}
	if account, err := s.accountStore.Get(bindID); err == nil && account.InstanceID != "" {
		if inst, err := s.instanceStore.Get(account.InstanceID); err == nil {
			return inst.Status != instance.StatusStopped
		}
	}
	if inst, err := s.instanceStore.Get(bindID); err == nil {
		return inst.Status != instance.StatusStopped
	}
	return false
}

func (s *AccountService) resolveProxySelection(bindID string, proxyID string, proxyURL string, fallbackID string, fallbackURL string) (string, string, error) {
	selectedID := strings.TrimSpace(proxyID)
	selectedURL := strings.TrimSpace(proxyURL)

	if selectedID == "" && selectedURL == "" {
		selectedID = strings.TrimSpace(fallbackID)
		selectedURL = strings.TrimSpace(fallbackURL)
	}

	if selectedURL == "" && selectedID != "" {
		existing, err := s.proxyStore.Get(selectedID)
		if err != nil {
			return "", "", fmt.Errorf("failed to load proxy: %w", err)
		}
		selectedURL = formatProxyURL(existing)
	}

	if selectedID != "" && selectedURL == "" {
		return "", "", fmt.Errorf("proxy URL is missing for selected proxy")
	}

	if selectedURL == "" {
		return "", "", nil
	}

	resolvedID, cfg, err := s.resolveProxy(bindID, selectedURL, selectedID)
	if err != nil {
		return "", "", err
	}
	if cfg == nil {
		return "", "", nil
	}
	return resolvedID, cfg.URL, nil
}

func proxyMatchesURL(existing *proxy.Proxy, raw string) bool {
	host, port, username, password, err := parseProxyURL(raw)
	if err != nil {
		return false
	}
	return existing.IP == host &&
		existing.Port == port &&
		existing.Username == username &&
		existing.Password == password
}

func parseProxyURL(raw string) (string, int, string, string, error) {
	normalized := raw
	if !strings.Contains(raw, "://") {
		normalized = "http://" + raw
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", 0, "", "", err
	}
	host := parsed.Hostname()
	if host == "" {
		return "", 0, "", "", fmt.Errorf("invalid proxy host")
	}
	port := parsed.Port()
	if port == "" {
		return "", 0, "", "", fmt.Errorf("proxy port missing")
	}
	portNum, err := net.LookupPort("tcp", port)
	if err != nil {
		return "", 0, "", "", fmt.Errorf("invalid proxy port")
	}

	var username string
	var password string
	if parsed.User != nil {
		username = parsed.User.Username()
		if pwd, ok := parsed.User.Password(); ok {
			password = pwd
		}
	}

	return host, portNum, username, password, nil
}
