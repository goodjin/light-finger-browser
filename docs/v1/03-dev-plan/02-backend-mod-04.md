# 后端开发计划 - 验证码模块

## 文档信息

| 字段 | 内容 |
|------|------|
| **模块编号** | MOD-04 |
| **模块名称** | 验证码模块 |
| **对应架构** | docs/v1/02-architecture/03-mod-04-phone.md |
| **优先级** | P0 |
| **预估工时** | 2 天 |

---

## 1. 模块概述

### 1.1 模块职责

- 管理手机号码资源（接码 + 固定号）
- 对接多个接码平台
- 提供统一的验证码接收接口

### 1.2 对应 PRD

| PRD编号 | 功能 | 用户故事 |
|---------|-----|---------|
| FR-005 | 验证码自动接收 | US-001 |
| FR-006 | 固定号换绑 | US-004 |

---

## 2. 技术设计

### 2.1 技术栈

| 类型 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.21+ |
| 数据库 | PostgreSQL | 15+ |
| 外部服务 | SMS-Activate API | - |
| 测试 | Go testing | - |

### 2.2 目录结构

```
internal/phone/
├── types.go                 # 数据结构定义
├── manager.go               # 手机号管理器
├── store.go                 # 数据存储
├── adapter/
│   ├── interface.go         # Provider 接口
│   ├── smsactivate.go       # SMS-Activate 适配器
│   ├── textnow.go           # TextNow 适配器
│   └── mock.go              # Mock 适配器 (测试用)
├── manager_test.go          # 单元测试
└── adapter_test.go         # 适配器测试
```

---

## 3. 接口清单

| 任务编号 | 接口编号 | 接口名称 | 复杂度 |
|---------|---------|---------|-------|
| T-01 | API-M4-001 | GetNumber(country, service) | 中 |
| T-02 | API-M4-002 | ReleaseNumber(id) | 低 |
| T-03 | API-M4-003 | GetCode(phoneID, service) | 中 |
| T-04 | API-M4-004 | WaitForCode(phoneID, service, timeout) | 高 |
| T-05 | API-M4-005 | RentNumber(country, duration) | 中 |

---

## 4. 数据结构

### 4.1 PhoneNumber

```go
type PhoneNumber struct {
    ID            string       `json:"id"`
    Number        string       `json:"number"`
    Country       string       `json:"country"`
    Type          NumberType   `json:"type"`
    Provider      string       `json:"provider"`
    Status        NumberStatus `json:"status"`
    RentExpiresAt time.Time    `json:"rent_expires_at"`
    InstanceID    string       `json:"instance_id"`
    CreatedAt     time.Time    `json:"created_at"`
    LastUsedAt    time.Time    `json:"last_used_at"`
}

type NumberType string

const (
    NumberTypeDisposable NumberType = "disposable"
    NumberTypeRental     NumberType = "rental"
    NumberTypeTextNow   NumberType = "textnow"
    NumberTypeESim       NumberType = "esim"
)
```

---

## 5. 开发任务拆分

### T-01: 数据结构与 Provider 接口

**任务概述**: 定义核心数据结构和 Provider 接口

**对应架构**:
- 数据结构规约: DATA-004
- 接口规约: API-M4-001~005

**输入**:
- 架构数据结构规约文档

**输出**:
- `internal/phone/types.go`
- `internal/phone/adapter/interface.go`

**实现要求**:

```go
// types.go
type PhoneNumber struct {
    ID            string       `json:"id"`
    Number        string       `json:"number"`
    Country       string       `json:"country"`
    Type          NumberType   `json:"type"`
    Provider      string       `json:"provider"`
    Status        NumberStatus `json:"status"`
    RentExpiresAt time.Time    `json:"rent_expires_at"`
    InstanceID    string       `json:"instance_id"`
    CreatedAt     time.Time    `json:"created_at"`
    LastUsedAt    time.Time    `json:"last_used_at"`
}

type NumberStatus string

const (
    NumberStatusAvailable NumberStatus = "available"
    NumberStatusActive   NumberStatus = "active"
    NumberStatusUsed     NumberStatus = "used"
    NumberStatusExpired  NumberStatus = "expired"
    NumberStatusFailed   NumberStatus = "failed"
)

// adapter/interface.go
type SMSProvider interface {
    GetBalance(ctx context.Context) (float64, error)
    GetNumber(ctx context.Context, country string, service string) (*PhoneNumber, error)
    ReleaseNumber(ctx context.Context, id string) error
    GetCode(ctx context.Context, phoneID string, service string) (string, error)
}
```

**验收标准**:
- [ ] **数据展示验证**: 数据结构与架构规约一致
- [ ] **接口完整性**: Provider 接口包含所有必需方法

**测试要求**:
- 测试文件: `types_test.go`
- 测试用例: ≤ 5 个

**预估工时**: 0.25 天

**依赖**: 无

---

### T-02: SMS-Activate 适配器实现

**任务概述**: 实现 SMS-Activate 接码平台适配器

**对应架构**:
- 接口规约: API-M4-001~005
- 架构设计: 4.1 接码平台适配器

**输入**:
- T-01 的接口定义

**输出**:
- `internal/phone/adapter/smsactivate.go`

**实现要求**:

```go
type SMSActivateAdapter struct {
    apiKey string
    baseURL string
    httpClient *http.Client
}

func (a *SMSActivateAdapter) GetNumber(ctx context.Context, country string, service string) (*PhoneNumber, error) {
    resp, err := a.httpClient.PostForm(a.baseURL+"/getNumber", url.Values{
        "api_key": {a.apiKey},
        "service": {service},
        "country": {country},
    })
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        ID     string `json:"id"`
        Number string `json:"number"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return &PhoneNumber{
        ID:       result.ID,
        Number:   result.Number,
        Country:  country,
        Type:     NumberTypeDisposable,
        Provider: "smsactivate",
        Status:   NumberStatusActive,
    }, nil
}

func (a *SMSActivateAdapter) GetCode(ctx context.Context, phoneID string, service string) (string, error) {
    // 调用 API 获取验证码
    // ...
}
```

**验收标准**:
- [ ] **功能验证**: 能够成功获取号码和验证码
- [ ] **错误处理**: 网络错误、超时处理正确

**测试要求**:
- 测试文件: `adapter/smsactivate_test.go`
- 使用 Mock 服务器测试

**预估工时**: 0.5 天

**依赖**: T-01

---

### T-03: TextNow 适配器实现

**任务概述**: 实现 TextNow 虚拟号适配器

**对应架构**:
- 接口规约: API-M4-005

**输入**:
- T-01 的接口定义

**输出**:
- `internal/phone/adapter/textnow.go`

**实现要求**:

```go
type TextNowAdapter struct {
    apiKey string
    httpClient *http.Client
}

func (a *TextNowAdapter) GetNumber(ctx context.Context, country string) (*PhoneNumber, error) {
    // TextNow API 调用
    // ...
}

func (a *TextNowAdapter) GetCode(ctx context.Context, phoneID string) (string, error) {
    // 获取 TextNow 短信
    // ...
}
```

**验收标准**:
- [ ] **功能验证**: 能够获取 TextNow 号码
- [ ] **验证码获取**: 能够接收验证码

**预估工时**: 0.5 天

**依赖**: T-01

---

### T-04: 手机号管理器实现

**任务概述**: 实现 PhoneManager 主逻辑

**对应架构**:
- 接口规约: API-M4-001~005
- 边界条件: BOUND-003

**输入**:
- T-01, T-02, T-03 的输出

**输出**:
- `internal/phone/manager.go`
- `internal/phone/store.go`

**实现要求**:

```go
type PhoneManager struct {
    store    Store
    adapters map[string]SMSProvider
}

const (
    MaxRetryCount = 3
    RetryInterval = 10 * time.Second
    CodeTimeout   = 10 * time.Minute
)

func (m *PhoneManager) GetNumber(ctx context.Context, country string, service string) (*PhoneNumber, error) {
    // 1. 尝试各个适配器
    for name, adapter := range m.adapters {
        phone, err := adapter.GetNumber(ctx, country, service)
        if err == nil {
            phone.Provider = name
            return m.store.Save(phone)
        }
    }
    return nil, ErrNoAvailableNumber
}

func (m *PhoneManager) WaitForCode(ctx context.Context, phoneID string, service string, timeout time.Duration) (*VerificationCode, error) {
    if timeout == 0 {
        timeout = CodeTimeout
    }

    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    retries := 0

    for {
        phone, err := m.store.Get(phoneID)
        if err != nil {
            return nil, err
        }

        adapter := m.adapters[phone.Provider]
        code, err := adapter.GetCode(ctx, phone.ExternalID, service)
        if err == nil && code != "" {
            return &VerificationCode{
                ID:     uuid.New().String(),
                PhoneID: phoneID,
                Code:   code,
            }, nil
        }

        if retries >= MaxRetryCount {
            return nil, ErrMaxRetriesExceeded
        }

        select {
        case <-time.After(RetryInterval):
            retries++
        case <-ctx.Done():
            return nil, ErrCodeTimeout
        }
    }
}
```

**验收标准**:
- [ ] **功能验证**: GetNumber、WaitForCode 功能正确
- [ ] **重试机制**: 3 次重试后切换
- [ ] **超时处理**: 10 分钟超时正确处理

**测试要求**:
- 测试文件: `manager_test.go`
- 使用 Mock 适配器

**预估工时**: 0.5 天

**依赖**: T-02, T-03

---

### T-05: 数据存储层实现

**任务概述**: 实现手机号数据存储

**对应架构**:
- 数据结构规约: DATA-004

**输入**:
- T-01 的数据结构

**输出**:
- `internal/phone/store.go`

**实现要求**:

```go
type Store interface {
    Save(phone *PhoneNumber) (*PhoneNumber, error)
    Get(id string) (*PhoneNumber, error)
    List(filter *NumberFilter) ([]*PhoneNumber, error)
    Update(phone *PhoneNumber) error
    Delete(id string) error
}

type PostgresStore struct {
    db *sql.DB
}

func (s *PostgresStore) Save(phone *PhoneNumber) (*PhoneNumber, error) {
    query := `
        INSERT INTO phone_numbers (id, number, country, type, provider, status, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING *
    `
    err := s.db.QueryRow(query,
        phone.ID, phone.Number, phone.Country, phone.Type, phone.Provider, phone.Status, time.Now(),
    ).Scan(&phone.ID)
    if err != nil {
        return nil, err
    }
    return phone, nil
}
```

**验收标准**:
- [ ] **CRUD 功能**: 所有 CRUD 操作正确
- [ ] **索引**: 支持按 country、status 筛选

**预估工时**: 0.25 天

**依赖**: T-01

---

### T-06: 租用号功能实现

**任务概述**: 实现固定号租用逻辑

**对应架构**:
- 接口规约: API-M4-005

**输入**:
- T-04 的管理器实现

**输出**:
- `internal/phone/manager.go` (补充)

**实现要求**:

```go
func (m *PhoneManager) RentNumber(ctx context.Context, country string, duration time.Duration) (*PhoneNumber, error) {
    // 调用租用号 API
    for name, adapter := range m.adapters {
        phone, err := adapter.RentNumber(ctx, country, duration)
        if err == nil {
            phone.Provider = name
            phone.Type = NumberTypeRental
            phone.RentExpiresAt = time.Now().Add(duration)
            return m.store.Save(phone)
        }
    }
    return nil, ErrNoAvailableNumber
}
```

**验收标准**:
- [ ] **租用功能**: 成功租用号码
- [ ] **过期处理**: 到期后状态正确更新

**预估工时**: 0.25 天

**依赖**: T-04, T-05

---

### T-07: 单元测试 - 基础功能

**任务概述**: 基础功能的单元测试

**输入**:
- T-04, T-05 的实现

**输出**:
- `manager_test.go`

**测试用例**:
1. TC-M4-01: GetNumber 获取可用号码成功
2. TC-M4-02: ReleaseNumber 释放号码成功
3. TC-M4-03: GetCode 获取验证码成功
4. TC-M4-04: WaitForCode 超时返回错误

**验收标准**:
- [ ] **功能验证**: 4 个测试用例通过
- [ ] **覆盖率**: ≥ 80%

**预估工时**: 0.25 天

**依赖**: T-06

---

### T-08: 集成测试

**任务概述**: 与接码平台的集成测试

**输入**:
- T-02, T-03, T-07 的实现

**输出**:
- `adapter/integration_test.go`

**测试用例**:
1. TC-M4-05: SMS-Activate 真实 API 测试 (需要 API Key)
2. TC-M4-06: TextNow 真实 API 测试 (需要 API Key)

**验收标准**:
- [ ] **集成验证**: 真实 API 调用成功
- [ ] **错误处理**: API 异常处理正确

**预估工时**: 0.25 天

**依赖**: T-07

---

## 6. 验收清单

### 6.1 功能验收

- [ ] 所有接口实现完成
- [ ] SMS-Activate 适配器完成
- [ ] TextNow 适配器完成
- [ ] 租用号功能完成
- [ ] 所有单元测试通过

### 6.2 质量验收

- [ ] 测试覆盖率 ≥ 80%
- [ ] 无 lint 错误

---

## 7. 覆盖映射

| 架构元素 | 架构编号 | 任务 | 覆盖状态 |
|---------|---------|------|---------|
| 接口 | API-M4-001 | T-01, T-04 | ✅ |
| 接口 | API-M4-002 | T-01, T-04 | ✅ |
| 接口 | API-M4-003 | T-04 | ✅ |
| 接口 | API-M4-004 | T-04 | ✅ |
| 接口 | API-M4-005 | T-06 | ✅ |
| 数据结构 | DATA-004 | T-01, T-05 | ✅ |
| 边界条件 | BOUND-003 | T-04, T-07 | ✅ |