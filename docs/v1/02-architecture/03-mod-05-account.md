# MOD-05: 账号注册模块

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | TikTok Matrix Operations System (TMOS) |
| **模块编号** | MOD-05 |
| **模块名称** | 账号注册模块 |
| **版本** | v1.0 |
| **对应PRD** | FR-004, FR-006 |
| **更新日期** | 2026-04-21 |

---

## 1. 系统定位

### 1.1 在整体架构中的位置

```
┌─────────────────────────────────────────────────────┐
│                   MOD-06 AI 自动化模块               │
└───────────────────────┬─────────────────────────────┘
                        │ ▼ 调用
┌─────────────────────────────────────────────────────┐
│              ★ MOD-05 账号注册模块 ★                 │
│                                                       │
│   依赖: MOD-01 指纹 │ MOD-03 IP │ MOD-04 验证码    │
└─────────────────────────────────────────────────────┘
```

### 1.2 核心职责

- TikTok 账号批量注册
- 固定号换绑
- 账号凭证管理

---

## 2. 对应PRD

| PRD章节 | 编号 | 内容 |
|---------|-----|------|
| 功能需求 | FR-004 | 账号批量注册 |
| 功能需求 | FR-006 | 固定号换绑 |
| 用户故事 | US-001 | 账号批量注册 |
| 用户故事 | US-004 | 固定号换绑 |

---

## 3. 接口定义

### 3.1 AccountManager 接口

```go
// AccountManager 账号管理器接口
// 文件: internal/account/manager.go
type AccountManager interface {
    // RegisterByGoogle Google 账号注册 TikTok
    RegisterByGoogle(ctx context.Context, req *RegisterRequest) (*RegisterResult, error)

    // RegisterByPhone 手机号注册 TikTok
    RegisterByPhone(ctx context.Context, req *RegisterRequest) (*RegisterResult, error)

    // BindPhone 换绑手机号
    BindPhone(ctx context.Context, accountID string, phoneID string) error

    // BatchRegister 批量注册
    BatchRegister(ctx context.Context, reqs []*RegisterRequest) ([]*RegisterResult, error)

    // Get 获取账号
    Get(ctx context.Context, id string) (*TikTokAccount, error)

    // List 列出账号
    List(ctx context.Context, filter *AccountFilter) ([]*TikTokAccount, error)

    // UpdateStatus 更新账号状态
    UpdateStatus(ctx context.Context, id string, status AccountStatus) error

    // Delete 删除账号
    Delete(ctx context.Context, id string) error
}
```

### 3.2 TikTokAccount 数据结构

```go
// TikTokAccount TikTok 账号
// 文件: internal/account/types.go
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

type AccountStatus string

const (
    AccountStatusPending     AccountStatus = "pending"
    AccountStatusRegistering AccountStatus = "registering"
    AccountStatusActive      AccountStatus = "active"
    AccountStatusSuspended   AccountStatus = "suspended"
    AccountStatusBanned      AccountStatus = "banned"
)
```

---

## 4. 核心设计

### 4.1 Google 注册流程

```go
func (m *AccountManager) RegisterByGoogle(ctx context.Context, req *RegisterRequest) (*RegisterResult, error) {
    // 1. 分配指纹和 IP
    fp, _ := m.fingerprint.GenerateRandom(req.Country)
    proxy, _ := m.proxy.Acquire(ctx, req.Country, proxy.ProxyTypeResidential)

    // 2. 创建浏览器实例
    instance, err := m.instance.Create(ctx, &instance.InstanceConfig{
        Fingerprint: fp,
        Proxy:       proxy,
    })
    if err != nil {
        return &RegisterResult{Status: "failed", Message: err.Error()}, err
    }

    // 3. 获取临时手机号
    phone, err := m.phone.GetNumber(ctx, req.Country, "tiktok")
    if err != nil {
        m.instance.Destroy(ctx, instance.ID)
        return &RegisterResult{Status: "failed", Message: err.Error()}, err
    }

    // 4. 执行 TikTok 注册
    cdp, _ := m.instance.GetCDPClient(ctx, instance.ID)
    if err := m.registerByGoogle(cdp, req.Email, req.EmailPassword, phone.Number); err != nil {
        m.phone.ReleaseNumber(ctx, phone.ID)
        return &RegisterResult{Status: "failed", Message: err.Error()}, err
    }

    // 5. 等待验证码
    code, err := m.phone.WaitForCode(ctx, phone.ID, "tiktok", 0)
    if err != nil {
        return &RegisterResult{Status: "failed", Message: "code timeout"}, err
    }

    // 6. 填写验证码
    if err := m.fillVerificationCode(cdp, code.Code); err != nil {
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
    m.accountStore.Save(account)

    // 8. 释放临时资源
    m.phone.ReleaseNumber(ctx, phone.ID)
    proxy.Release(ctx, proxy.ID)

    return &RegisterResult{AccountID: account.ID, Status: "success"}, nil
}
```

### 4.2 批量注册策略

```go
func (m *AccountManager) BatchRegister(ctx context.Context, reqs []*RegisterRequest) ([]*RegisterResult, error) {
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

---

## 5. 状态机设计

### 5.1 账号注册流程状态机

**状态转换图**:

```
┌─────────┐  开始注册  ┌────────────┐  注册成功  ┌──────────┐
│ PENDING │──────────►│ REGISTERING│──────────►│ ACTIVE   │
└─────────┘           └────────────┘            └──────────┘
                           │                        ▲
                           │ 注册失败               │
                           ▼                        │ 换绑成功
                      ┌─────────┐                   │
                      │  ERROR  │───────────────────┘
                      └─────────┘
```

---

## 6. 实现文件

| 文件路径 | 职责 |
|---------|------|
| `internal/account/manager.go` | 账号管理器主逻辑 |
| `internal/account/types.go` | 数据结构定义 |
| `internal/account/store.go` | 账号存储 |
| `internal/account/workflow.go` | 注册工作流 |
| `internal/account/automation.go` | TikTok 自动化操作 |

---

## 7. 覆盖映射

| PRD类型 | PRD编号 | 架构元素 | 覆盖状态 |
|---------|---------|---------|---------|
| 功能需求 | FR-004 | MOD-05 | ✅ |
| 功能需求 | FR-006 | MOD-05 | ✅ |
| 用户故事 | US-001 | API-M5-001~004 | ✅ |
| 用户故事 | US-004 | API-M5-003 | ✅ |