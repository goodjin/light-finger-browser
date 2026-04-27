# 后端开发计划 - 账号注册模块

## 文档信息

| 字段 | 内容 |
|------|------|
| **模块编号** | MOD-05 |
| **模块名称** | 账号注册模块 |
| **对应架构** | docs/v1/02-architecture/03-mod-05-account.md |
| **优先级** | P0 |
| **预估工时** | 2 天 |

---

## 1. 模块概述

### 1.1 模块职责

- TikTok 账号批量注册
- 固定号换绑
- 账号凭证管理

### 1.2 对应 PRD

| PRD编号 | 功能 | 用户故事 |
|---------|-----|---------|
| FR-004 | 账号批量注册 | US-001 |
| FR-006 | 固定号换绑 | US-004 |

---

## 2. 技术设计

### 2.1 技术栈

| 类型 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.21+ |
| 数据库 | PostgreSQL | 15+ |
| 外部服务 | TikTokAutomation | - |
| 测试 | Go testing | - |

### 2.2 目录结构

```
internal/account/
├── types.go              # 数据结构定义
├── manager.go            # 账号管理器
├── store.go              # 数据存储
├── workflow.go           # 注册工作流
├── automation.go         # TikTok 自动化操作
├── manager_test.go       # 单元测试
└── automation_test.go    # 自动化测试
```

---

## 3. 接口清单

| 任务编号 | 接口编号 | 接口名称 | 复杂度 |
|---------|---------|---------|-------|
| T-01 | API-M5-001 | RegisterByGoogle(req) | 高 |
| T-02 | API-M5-002 | RegisterByPhone(req) | 高 |
| T-03 | API-M5-003 | BindPhone(accountID, phoneID) | 中 |
| T-04 | API-M5-004 | BatchRegister(reqs) | 高 |
| T-05 | API-M5-005 | Get(id) | 低 |
| T-06 | API-M5-006 | List(filter) | 低 |
| T-07 | API-M5-007 | UpdateStatus(id, status) | 低 |
| T-08 | API-M5-008 | Delete(id) | 低 |

---

## 4. 数据结构

### 4.1 TikTokAccount

```go
type TikTokAccount struct {
    ID            string        `json:"id"`
    Username      string        `json:"username"`
    Email         string        `json:"email"`
    EmailPassword string        `json:"-"`  // 加密存储
    PhoneID       string        `json:"phone_id"`
    PhoneNumber   string        `json:"phone_number"`
    Status        AccountStatus `json:"status"`
    AccountLevel  int           `json:"account_level"`
    Group         string        `json:"group"`
    InstanceID    string        `json:"instance_id"`
    CreatedAt     time.Time     `json:"created_at"`
    UpdatedAt     time.Time     `json:"updated_at"`
}
```

---

## 5. 开发任务拆分

### T-01: 数据结构与状态定义

**任务概述**: 定义核心数据结构和状态

**对应架构**:
- 数据结构规约: DATA-Account
- 接口规约: API-M5-001~008

**输出**:
- `internal/account/types.go`

**实现要求**:

```go
// types.go
type AccountStatus string

const (
    AccountStatusPending     AccountStatus = "pending"
    AccountStatusRegistering AccountStatus = "registering"
    AccountStatusActive      AccountStatus = "active"
    AccountStatusSuspended   AccountStatus = "suspended"
    AccountStatusBanned     AccountStatus = "banned"
)

type TikTokAccount struct {
    ID            string        `json:"id"`
    Username      string        `json:"username"`
    Email         string        `json:"email"`
    EmailPassword string        `json:"-"`  // 加密存储
    PhoneID       string        `json:"phone_id"`
    PhoneNumber   string        `json:"phone_number"`
    Status        AccountStatus `json:"status"`
    AccountLevel  int           `json:"account_level"`
    Group         string        `json:"group"`
    InstanceID    string        `json:"instance_id"`
    CreatedAt     time.Time     `json:"created_at"`
    UpdatedAt     time.Time     `json:"updated_at"`
}

type RegisterRequest struct {
    Email         string `json:"email"`
    EmailPassword string `json:"email_password"`
    Country       string `json:"country"`
    Group         string `json:"group"`
}

type RegisterResult struct {
    AccountID string `json:"account_id"`
    Status    string `json:"status"`
    Message   string `json:"message"`
}
```

**验收标准**:
- [ ] **数据展示验证**: 数据结构与架构规约一致
- [ ] **状态定义验证**: 所有账号状态定义完整

**预估工时**: 0.25 天

**依赖**: 无

---

### T-02: 账号存储层实现

**任务概述**: 实现账号数据存储

**对应架构**:
- 数据结构规约: DATA-Account

**输入**:
- T-01 的数据结构

**输出**:
- `internal/account/store.go`

**实现要求**:

```go
type Store interface {
    Save(account *TikTokAccount) (*TikTokAccount, error)
    Get(id string) (*TikTokAccount, error)
    GetByEmail(email string) (*TikTokAccount, error)
    List(filter *AccountFilter) ([]*TikTokAccount, error)
    Update(account *TikTokAccount) error
    UpdateStatus(id string, status AccountStatus) error
    Delete(id string) error
    Count(filter *AccountFilter) (int, error)
}

type PostgresStore struct {
    db *sql.DB
}

func (s *PostgresStore) Save(account *TikTokAccount) (*TikTokAccount, error) {
    query := `
        INSERT INTO tiktok_accounts (id, username, email, email_password, phone_id, phone_number, status, account_level, account_group, instance_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        RETURNING *
    `
    err := s.db.QueryRow(query,
        account.ID, account.Username, account.Email, account.EmailPassword,
        account.PhoneID, account.PhoneNumber, account.Status, account.AccountLevel,
        account.Group, account.InstanceID, account.CreatedAt, account.UpdatedAt,
    ).Scan(/* ... */)
    return account, err
}
```

**验收标准**:
- [ ] **CRUD 功能**: 所有 CRUD 操作正确
- [ ] **索引**: 支持按 email、status、group 筛选

**预估工时**: 0.25 天

**依赖**: T-01

---

### T-03: TikTok 自动化操作实现

**任务概述**: 实现 TikTok 注册自动化操作封装

**对应架构**:
- 接口规约: API-M5-001, API-M5-002
- 边界条件: BOUND-006, BOUND-007

**输入**:
- T-01 的数据结构
- TikTokAutomation SDK

**输出**:
- `internal/account/automation.go`

**实现要求**:

```go
type TikTokAutomator struct {
    cdp CDPClient
}

func (a *TikTokAutomator) RegisterByGoogle(email, password, phone string) error {
    // 1. 打开 TikTok 注册页面
    a.cdp.Navigate(ctx, "https://www.tiktok.com/register")

    // 2. 点击 Google 注册
    a.cdp.Click(ctx, ".google-login-btn")

    // 3. 填写 Google 账号信息
    // ... 切换 iframe，填写 email/password

    // 4. 填写手机号
    a.cdp.Type(ctx, ".phone-input", phone)

    // 5. 获取验证码
    a.cdp.Click(ctx, ".send-code-btn")

    return nil
}

func (a *TikTokAutomator) FillVerificationCode(code string) error {
    a.cdp.Type(ctx, ".verification-code-input", code)
    a.cdp.Click(ctx, ".verify-btn")
    return nil
}

func (a *TikTokAutomator) CompleteProfile(username string) error {
    // 填写用户名
    a.cdp.Type(ctx, ".username-input", username)
    a.cdp.Click(ctx, ".next-btn")

    // 设置生日（随机）
    a.cdp.Click(ctx, ".birthday-select")
    // ...

    return nil
}
```

**验收标准**:
- [ ] **注册流程**: Google 注册流程完整
- [ ] **验证码填写**: 验证码填写功能正确
- [ ] **错误处理**: 超时、失败处理正确

**预估工时**: 0.5 天

**依赖**: T-01

---

### T-04: 注册工作流实现

**任务概述**: 实现账号注册工作流

**对应架构**:
- 接口规约: API-M5-001, API-M5-002
- 状态机: 账号注册流程状态机
- 边界条件: BOUND-006, BOUND-007

**输入**:
- T-01, T-02, T-03 的输出
- MOD-01 指纹模块
- MOD-03 IP 模块
- MOD-04 验证码模块

**输出**:
- `internal/account/workflow.go`

**实现要求**:

```go
func (w *RegisterWorkflow) Execute(ctx context.Context, req *RegisterRequest) (*RegisterResult, error) {
    // 1. 分配指纹和 IP
    fp, err := w.fingerprint.GenerateRandom(req.Country)
    if err != nil {
        return &RegisterResult{Status: "failed", Message: "fingerprint failed"}, err
    }

    proxy, err := w.proxy.Acquire(ctx, req.Country, proxy.ProxyTypeResidential)
    if err != nil {
        return &RegisterResult{Status: "failed", Message: "proxy failed"}, err
    }

    // 2. 创建浏览器实例
    instance, err := w.instance.Create(ctx, &instance.InstanceConfig{
        Fingerprint: fp,
        Proxy:       proxy,
    })
    if err != nil {
        w.proxy.Release(ctx, proxy.ID)
        return &RegisterResult{Status: "failed", Message: err.Error()}, err
    }

    // 3. 获取临时手机号
    phone, err := w.phone.GetNumber(ctx, req.Country, "tiktok")
    if err != nil {
        w.instance.Destroy(ctx, instance.ID)
        w.proxy.Release(ctx, proxy.ID)
        return &RegisterResult{Status: "failed", Message: "phone failed"}, err
    }

    // 4. 执行 TikTok 注册
    cdp, err := w.instance.GetCDPClient(ctx, instance.ID)
    if err != nil {
        w.cleanup(instance, proxy, phone)
        return &RegisterResult{Status: "failed", Message: err.Error()}, err
    }

    automator := NewTikTokAutomator(cdp)
    if err := automator.RegisterByGoogle(req.Email, req.EmailPassword, phone.Number); err != nil {
        w.cleanup(instance, proxy, phone)
        return &RegisterResult{Status: "failed", Message: err.Error()}, err
    }

    // 5. 等待验证码
    code, err := w.phone.WaitForCode(ctx, phone.ID, "tiktok", 0)
    if err != nil {
        w.cleanup(instance, proxy, phone)
        return &RegisterResult{Status: "failed", Message: "code timeout"}, err
    }

    // 6. 填写验证码
    if err := automator.FillVerificationCode(code.Code); err != nil {
        w.cleanup(instance, proxy, phone)
        return &RegisterResult{Status: "failed", Message: err.Error()}, err
    }

    // 7. 保存账号
    account := &TikTokAccount{
        ID:         uuid.New().String(),
        Username:   extractUsername(req.Email),
        Email:      req.Email,
        PhoneID:    phone.ID,
        Status:     AccountStatusActive,
        InstanceID: instance.ID,
    }
    w.store.Save(account)

    // 8. 释放临时资源
    w.phone.ReleaseNumber(ctx, phone.ID)
    w.proxy.Release(ctx, proxy.ID)

    return &RegisterResult{AccountID: account.ID, Status: "success"}, nil
}

func (w *RegisterWorkflow) cleanup(instance *instance.BrowserInstance, proxy *proxy.Proxy, phone *phone.PhoneNumber) {
    if instance != nil {
        w.instance.Destroy(context.Background(), instance.ID)
    }
    if proxy != nil {
        w.proxy.Release(context.Background(), proxy.ID)
    }
    if phone != nil {
        w.phone.ReleaseNumber(context.Background(), phone.ID)
    }
}
```

**验收标准**:
- [ ] **注册功能**: 完整注册流程正常
- [ ] **资源清理**: 失败时资源正确释放
- [ ] **状态更新**: 账号状态正确更新

**预估工时**: 0.5 天

**依赖**: T-02, T-03

---

### T-05: 账号管理器实现

**任务概述**: 实现 AccountManager 主逻辑

**对应架构**:
- 接口规约: API-M5-001~008

**输入**:
- T-01, T-02, T-03, T-04 的输出

**输出**:
- `internal/account/manager.go`

**实现要求**:

```go
type AccountManager interface {
    RegisterByGoogle(ctx context.Context, req *RegisterRequest) (*RegisterResult, error)
    RegisterByPhone(ctx context.Context, req *RegisterRequest) (*RegisterResult, error)
    BindPhone(ctx context.Context, accountID string, phoneID string) error
    BatchRegister(ctx context.Context, reqs []*RegisterRequest) ([]*RegisterResult, error)
    Get(ctx context.Context, id string) (*TikTokAccount, error)
    List(ctx context.Context, filter *AccountFilter) ([]*TikTokAccount, error)
    UpdateStatus(ctx context.Context, id string, status AccountStatus) error
    Delete(ctx context.Context, id string) error
}

type accountManager struct {
    store       Store
    workflow    *RegisterWorkflow
    phoneMgr    phone.PhoneManager
    instanceMgr instance.InstanceManager
    proxyMgr    proxy.ProxyManager
}

func (m *accountManager) RegisterByGoogle(ctx context.Context, req *RegisterRequest) (*RegisterResult, error) {
    return m.workflow.Execute(ctx, req)
}

func (m *accountManager) RegisterByPhone(ctx context.Context, req *RegisterRequest) (*RegisterResult, error) {
    // 类似 RegisterByGoogle，但使用手机号注册流程
    // ...
}

func (m *accountManager) BindPhone(ctx context.Context, accountID string, phoneID string) error {
    account, err := m.store.Get(accountID)
    if err != nil {
        return err
    }

    // 获取新手机号
    phone, err := m.phoneMgr.GetNumber(ctx, account.Email[:2], "tiktok")
    if err != nil {
        return err
    }

    // 创建实例进行换绑操作
    instance, err := m.instanceMgr.Create(ctx, &instance.InstanceConfig{
        AccountID: accountID,
    })
    if err != nil {
        return err
    }
    defer m.instanceMgr.Destroy(ctx, instance.ID)

    // 执行换绑
    cdp, err := m.instanceMgr.GetCDPClient(ctx, instance.ID)
    if err != nil {
        return err
    }

    // 导航到账号设置页面，执行换绑
    // ...

    // 更新账号信息
    account.PhoneID = phone.ID
    account.PhoneNumber = phone.Number
    return m.store.Update(account)
}

func (m *accountManager) BatchRegister(ctx context.Context, reqs []*RegisterRequest) ([]*RegisterResult, error) {
    // 并发控制: 最多 10 个并行
    sem := make(chan struct{}, 10)
    var wg sync.WaitGroup
    results := make([]*RegisterResult, len(reqs))

    for i, req := range reqs {
        wg.Add(1)
        go func(idx int, r *RegisterRequest) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            result, _ := m.RegisterByGoogle(ctx, r)
            results[idx] = result
        }(i, req)
    }

    wg.Wait()
    return results, nil
}
```

**验收标准**:
- [ ] **注册功能**: Google/Phone 注册成功
- [ ] **换绑功能**: 手机号换绑成功
- [ ] **批量注册**: 10 并发注册正常

**预估工时**: 0.5 天

**依赖**: T-04

---

### T-06 ~ T-08: 单元测试

**任务概述**: 完整的测试覆盖

**测试用例**:
1. TC-M5-01: RegisterByGoogle 注册成功
2. TC-M5-02: RegisterByPhone 注册成功
3. TC-M5-03: BindPhone 换绑成功
4. TC-M5-04: BatchRegister 10 并发注册
5. TC-M5-05: 注册失败资源释放

**验收标准**:
- [ ] **覆盖率**: ≥ 80%

**预估工时**: 0.25 天

**依赖**: T-01 ~ T-05

---

## 6. 验收清单

### 6.1 功能验收

- [ ] 所有接口实现完成
- [ ] Google/Phone 注册正常
- [ ] 换绑功能正常
- [ ] 批量注册正常
- [ ] 所有单元测试通过

### 6.2 质量验收

- [ ] 测试覆盖率 ≥ 80%

---

## 7. 覆盖映射

| 架构元素 | 架构编号 | 任务 | 覆盖状态 |
|---------|---------|------|---------|
| 接口 | API-M5-001 | T-04, T-05 | ✅ |
| 接口 | API-M5-002 | T-04, T-05 | ✅ |
| 接口 | API-M5-003 | T-05 | ✅ |
| 接口 | API-M5-004 | T-05 | ✅ |
| 接口 | API-M5-005 | T-05 | ✅ |
| 接口 | API-M5-006 | T-05 | ✅ |
| 接口 | API-M5-007 | T-05 | ✅ |
| 接口 | API-M5-008 | T-05 | ✅ |
| 数据结构 | DATA-Account | T-01, T-02 | ✅ |
| 状态机 | STATE-Account | T-04 | ✅ |
| 边界条件 | BOUND-006 | T-03 | ✅ |
| 边界条件 | BOUND-007 | T-03 | ✅ |
