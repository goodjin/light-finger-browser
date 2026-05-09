# Per-Tab Fingerprint: Browser Context Isolation

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | Light Finger Browser |
| **文档编号** | PER-TAB-FP-001 |
| **版本** | v1.0 |
| **更新日期** | 2026-05-09 |
| **对应 PRD** | FR-001, FR-002, FR-009 |
| **状态** | 设计中 |

---

## 1. 背景与目标

### 1.1 当前架构问题

当前架构为 **单实例 = 单指纹 = 所有标签页共享同一指纹**：

```
┌─────────────────────────────────────────────────────┐
│           Browser Instance (Single Context)           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐              │
│  │  Tab 1  │  │  Tab 2  │  │  Tab 3  │  ← 同一指纹  │
│  └─────────┘  └─────────┘  └─────────┘              │
└─────────────────────────────────────────────────────┘
```

**问题**：所有标签页共享同一个 BrowserContext，无法实现标签页级别的指纹隔离。

### 1.2 目标架构

使用 Chromium 的 **BrowserContext（浏览器上下文）** 机制实现每标签页独立指纹：

```
┌─────────────────────────────────────────────────────────────────┐
│                    Browser Instance (Process)                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  Context A     │  │  Context B     │  │  Context C     │  │
│  │  Fingerprint A  │  │  Fingerprint B │  │  Fingerprint C │  │
│  │  ┌─────────┐   │  │  ┌─────────┐   │  │  ┌─────────┐   │  │
│  │  │  Tab 1  │   │  │  │  Tab 2  │   │  │  │  Tab 3  │   │  │
│  │  └─────────┘   │  │  └─────────┘   │  │  └─────────┘   │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Wails Frontend (React)                      │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐          │
│  │  Tab Manager  │  │  Tab Card    │  │  FP Selector │          │
│  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘          │
│          │                  │                  │                     │
└──────────┼──────────────────┼──────────────────┼──────────────────┘
           │  Wails Bindings   │                  │
           ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Go Backend (App)                            │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    TabService (NEW)                         │    │
│  │  - CreateTab(ctx, instanceID, tabConfig)                   │    │
│  │  - CloseTab(ctx, instanceID, tabID)                        │    │
│  │  - ListTabs(ctx, instanceID)                               │    │
│  │  - GetTabCDPClient(ctx, instanceID, tabID)                 │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                      │
│  ┌───────────────────────────┼───────────────────────────────┐      │
│  │              InstanceService (EXISTING)                    │      │
│  │  - CreateInstance / DestroyInstance / GetCDPClient        │      │
│  │  - contextManager: sync.Map  (instanceID → ContextStore)  │      │
│  └───────────────────────────┼───────────────────────────────┘      │
│                              │                                      │
│  ┌───────────────────────────┼───────────────────────────────┐      │
│  │           CDPClient (EXISTING + EXTENDED)                  │      │
│  │  - CreateBrowserContext() → contextId                      │      │
│  │  - CreateTarget(url, contextId) → targetId                 │      │
│  │  - CloseBrowserContext(contextId)                          │      │
│  │  - GetCDPClientForTarget(targetId)                         │      │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Chromium Browser Process                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ Context A    │  │ Context B    │  │ Context C    │              │
│  │ (FingerprintA)│  │ (FingerprintB│  │ (FingerprintC)│              │
│  │  └── Tab 1   │  │  └── Tab 2   │  │  └── Tab 3   │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 模块职责

| 模块 | 职责 | 位置 |
|------|------|------|
| TabService | 标签页生命周期管理、CDP context 路由 | `app/commands/tab.go` (NEW) |
| InstanceService | 实例管理 + context 存储 | `app/commands/instance.go` (EXTEND) |
| CDPClient | BrowserContext CRUD via CDP | `instance/cdp.go` (EXTEND) |
| Frontend TabManager | 标签页 UI 状态、CDP client 缓存 | `frontend/src/components/TabManager.tsx` (NEW) |

---

## 3. 核心数据结构

### 3.1 BrowserContext 实体

```go
// app/commands/tab.go

// BrowserContext represents an isolated browser context with its own fingerprint
type BrowserContext struct {
    ID            string                   // CDP contextId
    InstanceID    string                   // Parent instance
    Fingerprint   *fingerprint.Fingerprint // Context-specific fingerprint
    ProxyURL      string                   // Context-specific proxy
    TabIDs        []string                 // Active tabs in this context
    CreatedAt     time.Time
    LastActiveAt  time.Time
}

// TabInfo represents a browser tab
type TabInfo struct {
    ID              string    // CDP targetId
    ContextID       string    // Parent context
    InstanceID      string    // Parent instance
    URL             string    // Current URL (may be empty before navigation)
    Title           string    // Tab title
    FingerprintSeed string    // For display purposes
    CreatedAt       time.Time
    LastActiveAt    time.Time
}

// TabConfig contains configuration for creating a new tab
type TabConfig struct {
    URL         string                   // Initial URL
    Fingerprint *fingerprint.Fingerprint // Fingerprint for this tab
    ProxyURL    string                   // Optional proxy override
}

// ContextStore holds all contexts for an instance
type ContextStore struct {
    mu       sync.Mutex
    contexts map[string]*BrowserContext // contextId → BrowserContext
    tabs     map[string]*TabInfo       // targetId → TabInfo
}
```

### 3.2 InstanceService 扩展

```go
// app/commands/instance.go

type InstanceService struct {
    manager      browserRuntimeManager
    store        *sqlite.InstanceStore
    cdpClients   sync.Map           // instanceID → CDPClient (main browser)
    targetURLs   sync.Map           // instanceID → string (cached webSocketDebuggerUrl)

    // NEW: Per-instance context management
    contextStores sync.Map          // instanceID → *ContextStore
}
```

### 3.3 CDPClient 扩展

```go
// instance/cdp.go

type CDPClientInterface interface {
    // ... existing methods ...

    // BrowserContext management (NEW)
    CreateBrowserContext(ctx context.Context) (string, error)     // → contextId
    CloseBrowserContext(ctx context.Context, contextId string) error
    CreateTargetWithContext(ctx context.Context, url string, contextId string) (string, error) // → targetId
    GetTargets(ctx context.Context) ([]*CDPTarget, error)
}

// CDPClient extends with context support
func (c *CDPClient) CreateBrowserContext(ctx context.Context) (string, error) {
    result, err := c.execute(ctx, "Target.createBrowserContext", nil)
    if err != nil {
        return "", err
    }
    browserContextId, ok := result["browserContextId"].(string)
    if !ok {
        return "", fmt.Errorf("CreateBrowserContext response missing browserContextId")
    }
    return browserContextId, nil
}

func (c *CDPClient) CloseBrowserContext(ctx context.Context, contextId string) error {
    _, err := c.execute(ctx, "Target.closeBrowserContext", map[string]interface{}{
        "browserContextId": contextId,
    })
    return err
}

func (c *CDPClient) CreateTargetWithContext(ctx context.Context, url string, contextId string) (string, error) {
    result, err := c.execute(ctx, "Target.createTarget", map[string]interface{}{
        "url":               url,
        "browserContextId":  contextId,
    })
    if err != nil {
        return "", err
    }
    targetID, ok := result["targetId"].(string)
    if !ok {
        return "", fmt.Errorf("CreateTarget response missing targetId")
    }
    return targetID, nil
}
```

---

## 4. API 设计

### 4.1 后端 API (Go → Wails)

```go
// app/commands/tab.go

// TabService handles per-tab fingerprint operations
type TabService struct {
    instanceSvc *InstanceService
}

// CreateTab creates a new tab with specific fingerprint in an existing instance
//对应用户故事: US-001 (per-tab fingerprint)
//对应功能需求: FR-001 (多实例浏览器管理 - per-tab扩展)
func (s *TabService) CreateTab(ctx context.Context, instanceID string, cfg *TabConfig) (*TabInfo, error) {
    // 1. Get instance and verify it's running
    inst, err := s.instanceSvc.GetInstance(ctx, instanceID)
    if err != nil {
        return nil, err
    }
    if inst.Status != instance.StatusRunning {
        return nil, fmt.Errorf("instance is not running")
    }

    // 2. Get main CDP client for this instance
    mainClient, err := s.instanceSvc.GetCDPClient(ctx, instanceID)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to instance: %w", err)
    }
    defer s.instanceSvc.CloseCDPClient(instanceID)

    // 3. Create isolated browser context
    contextId, err := mainClient.CreateBrowserContext(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create browser context: %w", err)
    }

    // 4. Create tab (target) within that context
    targetId, err := mainClient.CreateTargetWithContext(ctx, cfg.URL, contextId)
    if err != nil {
        // Cleanup context on failure
        _ = mainClient.CloseBrowserContext(ctx, contextId)
        return nil, fmt.Errorf("failed to create tab: %w", err)
    }

    // 5. Store context and tab info
    tabInfo := &TabInfo{
        ID:           targetId,
        ContextID:    contextId,
        InstanceID:   instanceID,
        URL:          cfg.URL,
        CreatedAt:    time.Now(),
        LastActiveAt: time.Now(),
    }

    // Store in context store (see Section 5 for details)
    s.getOrCreateContextStore(instanceID).AddContext(contextId, cfg.Fingerprint, cfg.ProxyURL)
    s.getOrCreateContextStore(instanceID).AddTab(targetId, tabInfo)

    return tabInfo, nil
}

// CloseTab closes a specific tab and its context
func (s *TabService) CloseTab(ctx context.Context, instanceID, tabID string) error {
    store := s.getContextStore(instanceID)
    if store == nil {
        return fmt.Errorf("instance has no tabs")
    }

    tab := store.GetTab(tabID)
    if tab == nil {
        return fmt.Errorf("tab not found")
    }

    // Get CDP client
    mainClient, err := s.instanceSvc.GetCDPClient(ctx, instanceID)
    if err != nil {
        return err
    }
    defer s.instanceSvc.CloseCDPClient(instanceID)

    // Close target
    _, err = mainClient.Execute(ctx, "Target.closeTarget", map[string]interface{}{
        "targetId": tabID,
    })
    if err != nil {
        log.Printf("[CloseTab] close target error: %v", err)
    }

    // Close browser context
    if tab.ContextID != "" {
        _ = mainClient.CloseBrowserContext(ctx, tab.ContextID)
        store.RemoveContext(tab.ContextID)
    }

    store.RemoveTab(tabID)
    return nil
}

// ListTabs lists all tabs in an instance
func (s *TabService) ListTabs(ctx context.Context, instanceID string) ([]*TabInfo, error) {
    store := s.getContextStore(instanceID)
    if store == nil {
        return []*TabInfo{}, nil
    }
    return store.ListTabs(), nil
}

// NavigateTab navigates a specific tab to a URL
func (s *TabService) NavigateTab(ctx context.Context, instanceID, tabID, url string) error {
    tab, err := s.getTab(instanceID, tabID)
    if err != nil {
        return err
    }

    client, err := s.instanceSvc.GetCDPClientForTab(ctx, instanceID, tabID)
    if err != nil {
        return fmt.Errorf("failed to connect to tab: %w", err)
    }
    defer s.instanceSvc.CloseCDPClient(instanceID) // Note: per-tab clients may differ

    return client.Navigate(ctx, url)
}
```

### 4.2 新增 Wails 绑定 (app.go)

```go
// app.go

// ==================== Tab Commands ====================

// CreateTab creates a new tab with the specified fingerprint in an existing instance
func (a *App) CreateTab(instanceID string, cfg *commands.TabConfig) (*commands.TabInfo, error) {
    tabSvc := commands.NewTabService(a.instanceSvc)
    return tabSvc.CreateTab(a.appContext(), instanceID, cfg)
}

// CloseTab closes a specific tab
func (a *App) CloseTab(instanceID, tabID string) error {
    tabSvc := commands.NewTabService(a.instanceSvc)
    return tabSvc.CloseTab(a.appContext(), instanceID, tabID)
}

// ListTabs lists all tabs in an instance
func (a *App) ListTabs(instanceID string) ([]*commands.TabInfo, error) {
    tabSvc := commands.NewTabService(a.instanceSvc)
    return tabSvc.ListTabs(a.appContext(), instanceID)
}

// NavigateTab navigates a specific tab to a URL
func (a *App) NavigateTab(instanceID, tabID, url string) error {
    tabSvc := commands.NewTabService(a.instanceSvc)
    return tabSvc.NavigateTab(a.appContext(), instanceID, tabID, url)
}
```

### 4.3 前端 API (TypeScript)

```typescript
// frontend/src/wailsjs/go/main/App.d.ts (新增)

export function CreateTab(arg1: string, arg2: commands.TabConfig): Promise<commands.TabInfo>;

export function CloseTab(arg1: string, arg2: string): Promise<void>;

export function ListTabs(arg1: string): Promise<Array<commands.TabInfo>>;

export function NavigateTab(arg1: string, arg2: string, arg3: string): Promise<void>;
```

### 4.4 前端数据模型

```typescript
// frontend/src/wailsjs/go/models/commands.ts (新增)

export class BrowserContext {
    id: string;
    instance_id: string;
    fingerprint_seed: string;
    proxy_url: string;
    tab_ids: string[];
    created_at: string;
    last_active_at: string;
}

export class TabInfo {
    id: string;              // CDP targetId
    context_id: string;     // BrowserContext id
    instance_id: string;
    url: string;
    title: string;
    fingerprint_seed: string;
    created_at: string;
    last_active_at: string;
}

export class TabConfig {
    url: string;
    fingerprint: fingerprint.Fingerprint | null;
    proxy_url: string;

    static createFrom(input: Partial<TabConfig>): TabConfig { ... }
}
```

---

## 5. 生命周期管理

### 5.1 Context Store 实现

```go
// app/commands/tab_context_store.go

type ContextStore struct {
    mu       sync.Mutex
    contexts map[string]*BrowserContext  // contextId → context
    tabs     map[string]*TabInfo         // targetId → tab
}

func (s *ContextStore) AddContext(contextId string, fp *fingerprint.Fingerprint, proxyURL string) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.contexts[contextId] = &BrowserContext{
        ID:          contextId,
        Fingerprint: fp,
        ProxyURL:    proxyURL,
        TabIDs:      []string{},
        CreatedAt:   time.Now(),
    }
}

func (s *ContextStore) AddTab(targetId string, tab *TabInfo) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.tabs[targetId] = tab
    if ctx, ok := s.contexts[tab.ContextID]; ok {
        ctx.TabIDs = append(ctx.TabIDs, targetId)
    }
}

func (s *ContextStore) RemoveTab(targetId string) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if tab, ok := s.tabs[targetId]; ok {
        if ctx, ok := s.contexts[tab.ContextID]; ok {
            ctx.TabIDs = removeString(ctx.TabIDs, targetId)
        }
        delete(s.tabs, targetId)
    }
}

func (s *ContextStore) RemoveContext(contextId string) {
    s.mu.Lock()
    defer s.mu.Unlock()

    delete(s.contexts, contextId)
}

func (s *ContextStore) GetTab(targetId string) *TabInfo {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.tabs[targetId]
}

func (s *ContextStore) ListTabs() []*TabInfo {
    s.mu.Lock()
    defer s.mu.Unlock()

    result := make([]*TabInfo, 0, len(s.tabs))
    for _, tab := range s.tabs {
        result = append(result, tab)
    }
    return result
}

func (s *ContextStore) GetContext(contextId string) *BrowserContext {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.contexts[contextId]
}

// Cleanup when instance is stopped
func (s *ContextStore) CloseAll(ctx context.Context, mainClient CDPClientInterface) error {
    s.mu.Lock()
    tabs := make([]*TabInfo, 0, len(s.tabs))
    for _, tab := range s.tabs {
        tabs = append(tabs, tab)
    }
    contexts := make([]string, 0, len(s.contexts))
    for id := range s.contexts {
        contexts = append(contexts, id)
    }
    s.mu.Unlock()

    // Close all tabs
    for _, tab := range tabs {
        _, _ = mainClient.Execute(ctx, "Target.closeTarget", map[string]interface{}{
            "targetId": tab.ID,
        })
    }

    // Close all contexts
    for _, contextId := range contexts {
        _ = mainClient.CloseBrowserContext(ctx, contextId)
    }

    return nil
}
```

### 5.2 实例停止时的清理

```go
// app/commands/instance.go - Extend StopInstance

func (s *InstanceService) StopInstance(ctx context.Context, id string) error {
    // NEW: Cleanup all contexts and tabs before stopping
    if store, ok := s.contextStores.LoadAndDelete(id); ok {
        mainClient, err := s.GetCDPClient(ctx, id)
        if err == nil {
            defer s.CloseCDPClient(id)
            store.CloseAll(ctx, mainClient)
        }
    }

    return s.manager.Stop(ctx, id)
}
```

### 5.3 生命周期状态图

```
标签页生命周期：

  CreateTab
      │
      ▼
┌─────────────┐     CloseTab      ┌─────────────┐
│  ACTIVE     │ ───────────────▶  │  CLOSING    │
│             │                   │             │
│  - Context   │                   │  - Close CDP│
│    created  │                   │  - Close    │
│  - Tab       │                   │    Context  │
│    created  │                   └──────┬──────┘
└──────┬──────┘                          │
       │                                  ▼
       │                           ┌─────────────┐
       │ NavigateTab               │   CLOSED    │
       │ (url/title update)        │             │
       └────────────────────────▶  └─────────────┘

实例停止时：
  StopInstance
      │
      ▼
  ┌────────────────────────────────┐
  │ For each Context in ContextStore │
  │   → Close all Tabs              │
  │   → Close BrowserContext        │
  └────────────────────────────────┘
      │
      ▼
  Stop Browser Process
```

---

## 6. CDP 集成详情

### 6.1 CDP 命令映射

| 操作 | CDP Command | 参数 | 返回值 |
|------|------------|------|--------|
| 创建 Context | `Target.createBrowserContext` | `{}` | `{browserContextId: string}` |
| 关闭 Context | `Target.closeBrowserContext` | `{browserContextId: string}` | `{}` |
| 创建标签页 | `Target.createTarget` | `{url: string, browserContextId: string}` | `{targetId: string}` |
| 关闭标签页 | `Target.closeTarget` | `{targetId: string}` | `{}` |
| 获取所有目标 | `Target.getTargets` | `{}` | `{targetInfos: Target[]}` |
| 连接标签页 CDP | `IO.read` + WebSocket | `WebSocketDebuggerUrl` from Target | CDP client |

### 6.2 标签页 CDP Client 获取

关键问题：每个标签页有自己的 WebSocket 连接，不能复用主浏览器的 CDP client。

```go
// app/commands/instance.go

// GetCDPClientForTab gets a CDP client for a specific tab
func (s *InstanceService) GetCDPClientForTab(ctx context.Context, instanceID, tabID string) (instance.CDPClientInterface, error) {
    inst, err := s.store.Get(instanceID)
    if err != nil {
        return nil, err
    }

    // Get main browser CDP endpoint
    wsURL := inst.CDPEndpoint
    if !strings.HasPrefix(wsURL, "ws://") {
        wsURL = "ws://" + wsURL
    }

    // Query /json for the specific tab's WebSocket URL
    jsonURL := strings.Replace(wsURL, "ws://", "http://", 1) + "/json"
    resp, err := http.Get(jsonURL)
    if err != nil {
        return nil, fmt.Errorf("failed to query CDP targets: %w", err)
    }
    defer resp.Body.Close()

    var targets []map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
        return nil, fmt.Errorf("failed to decode CDP targets: %w", err)
    }

    // Find the specific tab by targetId
    for _, t := range targets {
        if tid, ok := t["id"].(string); ok && tid == tabID {
            if wsu, ok := t["webSocketDebuggerUrl"].(string); ok {
                conn, _, err := instance.DefaultDialer.DialContext(ctx, "tcp", wsu)
                if err != nil {
                    return nil, fmt.Errorf("failed to dial tab CDP: %w", err)
                }
                return instance.NewCDPClient(conn), nil
            }
        }
    }

    return nil, fmt.Errorf("tab not found in CDP targets")
}
```

### 6.3 Context 与 Tab 的关系

```
┌─────────────────────────────────────────────────────────────────┐
│                     Browser Process                              │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  CDP Endpoint: ws://localhost:9222/json                 │    │
│  │                                                          │    │
│  │  Target 1 (main):                                        │    │
│  │    id: "A1B2C3D4"                                        │    │
│  │    type: "page"                                          │    │
│  │    url: "chrome://newtab"                                │    │
│  │    webSocketDebuggerUrl: "ws://localhost:9222/devtools..│    │
│  │                                                          │    │
│  │  Target 2 (tab in Context A):                            │    │
│  │    id: "E5F6G7H8"                                        │    │
│  │    type: "page"                                          │    │
│  │    browserContextId: "CONTEXT-A"                         │    │
│  │    url: "https://example.com"                            │    │
│  │    webSocketDebuggerUrl: "ws://localhost:9222/devtools..│    │
│  │                                                          │    │
│  │  Target 3 (tab in Context B):                            │    │
│  │    id: "I9J0K1L2"                                        │    │
│  │    type: "page"                                          │    │
│  │    browserContextId: "CONTEXT-B"                         │    │
│  │    url: "https://google.com"                             │    │
│  │    webSocketDebuggerUrl: "ws://localhost:9222/devtools..│    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. 内存与性能考虑

### 7.1 内存开销分析

| 组件 | 开销 | 说明 |
|------|------|------|
| BrowserContext | ~10-50MB | Chromium 每个 context 维护独立的 JS 堆、DOM存储、cookie jar |
| Tab (含 CDP conn) | ~5-20MB | 每个 tab 的渲染进程 + WebSocket 连接 |
| CDP Client | ~1MB | Go 端的 CDP 协议开销 |

**经验法则**：
- 1个 BrowserContext + 1个 Tab ≈ 50-100MB 额外内存
- 10个 Contexts + 10个 Tabs ≈ 500MB - 1GB 额外内存
- 建议单个实例最多 20-30 个 Context（取决于可用内存）

### 7.2 优化策略

```go
// 1. Context 复用：同一指纹配置的 Context 可被多个 Tab 复用
type ContextStore struct {
    // Key: fingerprint seed + proxy URL hash
    // Value: contextId
    contextPool map[string]string
}

// 2. 懒加载 Context：Tab 关闭后延迟 N 秒再关闭 Context
type TabService struct {
    pendingCloseTimer *time.Timer
    pendingCloseCtx   map[string]*BrowserContext  // contextId → timer
}

// 3. CDP Client 池化：复用 CDP 连接而非每次创建新连接
type CDPClientPool struct {
    clients map[string]*CDPClient  // targetId → client
    mu      sync.Mutex
}
```

### 7.3 资源限制

```go
// instance/types.go

// 新增限制常量
const (
    MaxContextsPerInstance = 30   // 单实例最多 Context 数
    MaxTabsPerInstance = 100      // 单实例最多标签页数
    ContextIdleTimeout = 5 * time.Minute  // Context 空闲超时
)
```

---

## 8. 前端 UI 设计

### 8.1 标签页选择器组件

```tsx
// frontend/src/components/TabFingerprintSelector.tsx

interface TabFingerprintSelectorProps {
    instanceId: string;
    onTabCreated: (tab: commands.TabInfo) => void;
    onTabClosed: (tabId: string) => void;
    onTabNavigated: (tabId: string, url: string) => void;
}

export function TabFingerprintSelector({ instanceId, onTabCreated, onTabClosed, onTabNavigated }: TabFingerprintSelectorProps) {
    const [tabs, setTabs] = useState<commands.TabInfo[]>([]);
    const [loading, setLoading] = useState(false);
    const [selectedTabId, setSelectedTabId] = useState<string | null>(null);
    const [showCreate, setShowCreate] = useState(false);
    const [url, setUrl] = useState('');

    // Tab 创建
    async function createTab() {
        try {
            setLoading(true);
            // 使用实例的默认指纹或指定指纹
            const fp = await GenerateRandomFingerprint('US');
            const tab = await CreateTab(instanceId, {
                url: url || 'about:blank',
                fingerprint: fp,
                proxy_url: '',
            });
            setTabs(prev => [...prev, tab]);
            setSelectedTabId(tab.id);
            setShowCreate(false);
            setUrl('');
            onTabCreated(tab);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    }

    // Tab 关闭
    async function closeTab(tabId: string) {
        try {
            await CloseTab(instanceId, tabId);
            setTabs(prev => prev.filter(t => t.id !== tabId));
            if (selectedTabId === tabId) {
                setSelectedTabId(tabs[0]?.id || null);
            }
            onTabClosed(tabId);
        } catch (err) {
            setError(String(err));
        }
    }

    // Tab 导航
    async function navigateTab(tabId: string, navigateUrl: string) {
        try {
            await NavigateTab(instanceId, tabId, navigateUrl);
            setTabs(prev => prev.map(t =>
                t.id === tabId ? { ...t, url: navigateUrl } : t
            ));
            onTabNavigated(tabId, navigateUrl);
        } catch (err) {
            setError(String(err));
        }
    }

    return (
        <div className="tab-fingerprint-selector">
            {/* 标签页列表 */}
            <div className="tab-list">
                {tabs.map(tab => (
                    <div
                        key={tab.id}
                        className={`tab-item ${selectedTabId === tab.id ? 'selected' : ''}`}
                        onClick={() => setSelectedTabId(tab.id)}
                    >
                        <span className="tab-fp">{tab.fingerprint_seed?.slice(0, 8)}</span>
                        <span className="tab-url">{tab.url || 'about:blank'}</span>
                        <button onClick={() => closeTab(tab.id)}>×</button>
                    </div>
                ))}
                <button onClick={() => setShowCreate(true)}>+ New Tab</button>
            </div>

            {/* 创建标签页弹窗 */}
            {showCreate && (
                <div className="modal-overlay">
                    <div className="modal">
                        <h3>Create New Tab</h3>
                        <input
                            type="text"
                            placeholder="URL (optional)"
                            value={url}
                            onChange={e => setUrl(e.target.value)}
                        />
                        {/* 指纹选择器：使用实例指纹或自定义 */}
                        <div className="form-group">
                            <label>Fingerprint</label>
                            <select>
                                <option value="instance">Use Instance Default</option>
                                <option value="custom">Custom Fingerprint</option>
                            </select>
                        </div>
                        <div className="modal-actions">
                            <button onClick={() => setShowCreate(false)}>Cancel</button>
                            <button onClick={createTab} disabled={loading}>
                                {loading ? 'Creating...' : 'Create'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
```

### 8.2 实例卡片扩展

在 `InstancesPage.tsx` 的实例卡片中，添加标签页管理按钮：

```tsx
// 在实例卡片的 actions 部分添加
<button
    className="btn-secondary"
    onClick={() => openTabManager(inst.id)}
    disabled={inst.status !== 'running'}
>
    Manage Tabs
</button>
```

---

## 9. 错误处理

### 9.1 错误类型

| 错误码 | 场景 | 处理方式 |
|-------|------|---------|
| `ERR_CONTEXT_CREATE_FAILED` | BrowserContext 创建失败 | 回滚、返回错误、记录日志 |
| `ERR_TAB_CREATE_FAILED` | Tab 创建失败 | 关闭已创建 Context、返回错误 |
| `ERR_TAB_NOT_FOUND` | Tab 不存在 | 返回 404、清理 stale 状态 |
| `ERR_CONTEXT_LIMIT` | 超出 Context 数量限制 | 返回错误、提示用户关闭部分标签页 |
| `ERR_CDP_DISCONNECTED` | CDP 连接断开 | 尝试重连、清理本地状态 |
| `ERR_INSTANCE_NOT_RUNNING` | 实例未运行 | 返回错误、提示启动实例 |

### 9.2 错误恢复

```go
// CreateTab 错误恢复
func (s *TabService) CreateTab(...) (*TabInfo, error) {
    contextId, err := mainClient.CreateBrowserContext(ctx)
    if err != nil {
        return nil, &TabError{
            Code:    ErrContextCreateFailed,
            Message: fmt.Sprintf("failed to create context: %v", err),
            Retriable: false,
        }
    }

    targetId, err := mainClient.CreateTargetWithContext(ctx, cfg.URL, contextId)
    if err != nil {
        // 清理已创建的 Context
        _ = mainClient.CloseBrowserContext(ctx, contextId)
        return nil, &TabError{
            Code:    ErrTabCreateFailed,
            Message: fmt.Sprintf("failed to create tab: %v", err),
            Retriable: true,  // 可重试
        }
    }
    // ... 保存状态 ...
}
```

---

## 10. 安全性考虑

### 10.1 Context 隔离

每个 BrowserContext 拥有独立的：
- Cookie 存储 (`CookieStorage`)
- LocalStorage / SessionStorage
- 缓存分区 (Partitioned Cookies)
- Service Workers

**注意**：BrowserContext 隔离不适用于 `devicePixelRatio`、`timezone` 等硬件级指纹，这些需要在 Chromium 源码层面注入。

### 10.2 Context 销毁验证

```go
// 关闭 Context 前验证所有 Tab 已关闭
func (s *ContextStore) CanCloseContext(contextId string) bool {
    s.mu.Lock()
    defer s.mu.Unlock()

    ctx, ok := s.contexts[contextId]
    if !ok {
        return true
    }
    return len(ctx.TabIDs) == 0
}
```

---

## 11. 实现文件清单

| 文件路径 | 职责 | 状态 |
|---------|------|------|
| `app/commands/tab.go` | TabService 定义 | 新建 |
| `app/commands/tab_context_store.go` | ContextStore 实现 | 新建 |
| `instance/cdp.go` | 扩展 CDPClientInterface | 修改 |
| `app/commands/instance.go` | 集成 contextStores | 修改 |
| `app.go` | 新增 Wails 绑定 | 修改 |
| `frontend/src/components/TabFingerprintSelector.tsx` | 标签页选择组件 | 新建 |
| `frontend/src/wailsjs/go/main/App.d.ts` | Wails 类型声明 | 重新生成 |
| `frontend/src/wailsjs/go/models/commands.ts` | 命令模型 | 重新生成 |

---

## 12. 测试场景

| 测试编号 | 场景 | 预期结果 |
|---------|------|---------|
| TC-001 | 创建 3 个不同指纹的标签页 | 每个标签页有独立的 BrowserContext 和指纹 |
| TC-001 | 关闭中间标签页 | 其他标签页不受影响，Context 正确清理 |
| TC-003 | 标签页间 Cookie 隔离 | Tab A 的 Cookie 不影响 Tab B |
| TC-004 | 实例停止时清理 | 所有 Context 和 Tab 正确关闭 |
| TC-005 | 超出 Context 限制 | 返回错误，提示用户 |
| TC-006 | 标签页导航 | 导航后 URL 更新，标签页仍属于同一 Context |
| TC-007 | 重启标签页所在实例 | 标签页状态丢失（需重新创建） |

---

## 13. 已知限制

1. **硬件指纹共享**：同一进程的 BrowserContext 共享 `navigator.hardwareConcurrency`、`navigator.deviceMemory` 等硬件级属性。如需真正隔离，需 Chromium 源码层面 patch。

2. **Context 数量限制**：Chromium 对单进程 BrowserContext 数量有内部限制（约 100+），实际受内存限制更严格。

3. **WebGL 上下文共享**：同一 GPU 设备的 WebGL 上下文可能在 Context 间共享，需在源码层面处理。

4. **Debug Port 限制**：当前架构每个实例只有一个调试端口，所有 Tab 的 CDP 连接都通过同一个 `/json` 端点发现。

---

## 14. 变更历史

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|---------|------|
| 1.0 | 2026-05-09 | 初始版本 | Claude |
