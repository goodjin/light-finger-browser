# MOD-04: 验证码模块

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | TikTok Matrix Operations System (TMOS) |
| **模块编号** | MOD-04 |
| **模块名称** | 验证码模块 |
| **版本** | v1.0 |
| **对应PRD** | FR-005, FR-006 |
| **更新日期** | 2026-04-21 |

---

## 1. 系统定位

### 1.1 在整体架构中的位置

```
┌─────────────────────────────────────────────────────┐
│                   MOD-05 账号注册模块                  │
└───────────────────────┬─────────────────────────────┘
                        │ ▼ 获取号码/验证码
┌─────────────────────────────────────────────────────┐
│              ★ MOD-04 验证码模块 ★                   │
└─────────────────────────────────────────────────────┘
                        │ ▼ 调用
┌─────────────────────────────────────────────────────┐
│              接码平台 (SMS-Activate, TextNow)       │
└─────────────────────────────────────────────────────┘
```

### 1.2 核心职责

- 管理手机号码资源（接码 + 固定号）
- 对接多个接码平台
- 提供统一的验证码接收接口

---

## 2. 对应PRD

| PRD章节 | 编号 | 内容 |
|---------|-----|------|
| 功能需求 | FR-005 | 验证码自动接收 |
| 功能需求 | FR-006 | 固定号换绑 |
| 用户故事 | US-001 | 账号批量注册 |
| 用户故事 | US-004 | 固定号换绑 |
| 验收标准 | AC-001-03 | 验证码自动接收 |
| 业务规则 | Rule-003 | 验证码接收重试规则 |

---

## 3. 接口定义

### 3.1 PhoneManager 接口

```go
// PhoneManager 手机号管理器接口
// 文件: internal/phone/manager.go
type PhoneManager interface {
    // GetNumber 获取可用号码
    GetNumber(ctx context.Context, country string, service string) (*PhoneNumber, error)

    // ReleaseNumber 释放号码
    ReleaseNumber(ctx context.Context, id string) error

    // GetCode 获取验证码
    GetCode(ctx context.Context, phoneID string, service string) (*VerificationCode, error)

    // WaitForCode 等待验证码 (带超时)
    WaitForCode(ctx context.Context, phoneID string, service string, timeout time.Duration) (*VerificationCode, error)

    // RentNumber 租用号码
    RentNumber(ctx context.Context, country string, duration time.Duration) (*PhoneNumber, error)

    // RenewRent 续租
    RenewRent(ctx context.Context, id string, duration time.Duration) error

    // List 列出号码列表
    List(ctx context.Context, filter *NumberFilter) ([]*PhoneNumber, error)
}
```

### 3.2 PhoneNumber 数据结构

```go
// PhoneNumber 手机号码
// 文件: internal/phone/types.go
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
    NumberTypeDisposable NumberType = "disposable"  // 一次性接码
    NumberTypeRental     NumberType = "rental"      // 租用号
    NumberTypeTextNow    NumberType = "textnow"    // TextNow 虚拟号
    NumberTypeESim       NumberType = "esim"       // eSIM
)

type NumberStatus string

const (
    NumberStatusAvailable NumberStatus = "available"
    NumberStatusActive   NumberStatus = "active"
    NumberStatusUsed     NumberStatus = "used"
    NumberStatusExpired  NumberStatus = "expired"
    NumberStatusFailed   NumberStatus = "failed"
)
```

---

## 4. 核心设计

### 4.1 接码平台适配器

```go
// SMSProvider 接码平台接口
// 文件: internal/phone/adapter/smsactivate.go
type SMSProvider interface {
    GetBalance(ctx context.Context) (float64, error)
    GetNumber(ctx context.Context, country string, service string) (*PhoneNumber, error)
    ReleaseNumber(ctx context.Context, id int64) error
    GetCode(ctx context.Context, id int64, service string) (string, error)
    SetStatus(ctx context.Context, id int64, status string) error
}

// SMSActivateAdapter SMS-Activate 适配器
type SMSActivateAdapter struct {
    apiKey string
    baseURL string
}

func (a *SMSActivateAdapter) GetNumber(ctx context.Context, country string, service string) (*PhoneNumber, error) {
    // 调用 SMS-Activate API
    resp, err := http.PostForm(a.baseURL+"/getNumber", url.Values{
        "api_key": {a.apiKey},
        "service": {service},
        "country": {country},
    })
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        ID     int    `json:"id"`
        Number string `json:"number"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return &PhoneNumber{
        ID:       strconv.Itoa(result.ID),
        Number:   result.Number,
        Provider: "smsactivate",
        Status:   NumberStatusActive,
    }, nil
}
```

### 4.2 验证码等待逻辑

```go
const (
    MaxRetryCount = 3
    RetryInterval = 10 * time.Second
    CodeTimeout   = 10 * time.Minute
)

func (m *PhoneManager) WaitForCode(ctx context.Context, phoneID string, service string, timeout time.Duration) (*VerificationCode, error) {
    if timeout == 0 {
        timeout = CodeTimeout
    }

    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    retries := 0

    for {
        code, err := m.getCode(phoneID, service)
        if err == nil && code != "" {
            return &VerificationCode{
                ID:          uuid.New().String(),
                PhoneID:     phoneID,
                Code:        code,
                Source:      service,
                ReceivedAt:  time.Now(),
                ExpiresAt:   time.Now().Add(5 * time.Minute),
                Status:      "verified",
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

---

## 5. 边界条件

### 5.1 验证码重试规则 (Rule-003)

| 规则 | 值 |
|------|---|
| 重试次数 | 最多 3 次 |
| 重试间隔 | 10 秒 |
| 超时时间 | 10 分钟 |

---

## 6. 实现文件

| 文件路径 | 职责 |
|---------|------|
| `internal/phone/manager.go` | 手机号管理器主逻辑 |
| `internal/phone/types.go` | 数据结构定义 |
| `internal/phone/adapter/` | 平台适配器 (SMS-Activate, TextNow, eSIM) |

---

## 7. 覆盖映射

| PRD类型 | PRD编号 | 架构元素 | 覆盖状态 |
|---------|---------|---------|---------|
| 功能需求 | FR-005 | MOD-04 | ✅ |
| 功能需求 | FR-006 | MOD-04 | ✅ |
| 验收标准 | AC-001-03 | BOUND-003 | ✅ |
| 业务规则 | Rule-003 | BOUND-003 | ✅ |