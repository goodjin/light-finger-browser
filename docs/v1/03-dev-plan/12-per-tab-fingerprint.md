# Per-Tab Fingerprint 开发计划

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | Light Finger Browser |
| **文档编号** | PER-TAB-FP-DEV-001 |
| **版本** | v1.0 |
| **对应架构** | `docs/v1/02-architecture/per-tab-fingerprint.md` |
| **创建日期** | 2026-05-09 |
| **状态** | 待开发 |

---

## 1. 概述

### 1.1 目标

实现基于 Chromium BrowserContext 隔离的每标签页独立指纹功能。

### 1.2 对应需求

| PRD 编号 | 需求 | 说明 |
|---------|------|------|
| FR-001 | 多实例浏览器管理 | per-tab 扩展 |
| FR-002 | 独立指纹生成与校验 | per-tab fingerprint 支持 |
| FR-009 | 关键交互 UX 约束 | 弹窗确认、按钮状态 |

### 1.3 架构定位

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Per-Tab Fingerprint 架构                          │
├─────────────────────────────────────────────────────────────────────┤
│  Frontend: TabFingerprintSelector                                    │
│  Backend:  TabService + ContextStore + CDPClient 扩展                │
│  CDP:     Target.createBrowserContext / Target.createTarget          │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. 技术设计

### 2.1 技术栈

| 层次 | 技术 | 说明 |
|------|------|------|
| 后端 | Go 1.21+ | Wails 绑定、CDP 通信 |
| 前端 | React + TypeScript | Wails 运行时绑定 |
| 协议 | Chrome DevTools Protocol | BrowserContext CRUD |

### 2.2 目录结构

```
app/commands/
├── tab.go                    # TabService (NEW)
├── tab_context_store.go       # ContextStore (NEW)
└── instance.go               # 修改：集成 contextStores

instance/
└── cdp.go                   # 修改：扩展 CDPClientInterface

app.go                        # 修改：新增 Wails 绑定

frontend/src/components/
└── TabFingerprintSelector.tsx  # 标签页选择组件 (NEW)
```

---

## 3. 开发任务拆分

### 任务约束

- **代码变更**: ≤ 200行
- **涉及文件**: ≤ 5个
- **测试用例**: ≤ 10个

### 任务清单

| 任务编号 | 任务名称 | 优先级 | 依赖 | 代码行数 |
|---------|---------|--------|------|---------|
| T-01 | CDPClient BrowserContext 扩展 | P0 | - | ~80 |
| T-02 | ContextStore 实现 | P0 | T-01 | ~150 |
| T-03 | TabService 实现 | P0 | T-02 | ~180 |
| T-04 | InstanceService context 集成 | P0 | T-03 | ~100 |
| T-05 | Wails 绑定 (app.go) | P0 | T-04 | ~50 |
| T-06 | TabFingerprintSelector 组件 | P1 | T-05 | ~200 |
| T-07 | Wails 类型重新生成 | P1 | T-05 | - |
| T-08 | 集成测试验证 | P1 | T-06 | ~100 |

---

## 4. 详细任务定义

### T-01: CDPClient BrowserContext 扩展

**任务概述**: 扩展 CDPClientInterface，新增 BrowserContext 管理方法

**对应架构**: `per-tab-fingerprint.md` 第 3.3 节

**输入**:
- `instance/cdp.go`
- CDP 协议规范

**输出**:
- `instance/cdp.go` (修改)

**实现要求**:

```go
// CDPClientInterface 新增方法
type CDPClientInterface interface {
    // ... existing methods ...

    // BrowserContext management
    CreateBrowserContext(ctx context.Context) (string, error)
    CloseBrowserContext(ctx context.Context, contextId string) error
    CreateTargetWithContext(ctx context.Context, url string, contextId string) (string, error)
    GetTargets(ctx context.Context) ([]*CDPTarget, error)
}

// CDPClient 实现
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
        "url":              url,
        "browserContextId": contextId,
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

**验收标准**:
- [ ] `CreateBrowserContext` 返回有效的 contextId
- [ ] `CloseBrowserContext` 正确关闭 context
- [ ] `CreateTargetWithContext` 在指定 context 中创建 tab

**测试要求**:
- 单元测试: `instance/cdp_test.go`
- 测试用例: ≤ 5个

**预估工时**: 0.5天

**依赖**: 无

---

### T-02: ContextStore 实现

**任务概述**: 实现 ContextStore 内存存储，管理 BrowserContext 和 TabInfo

**对应架构**: `per-tab-fingerprint.md` 第 5.1 节

**输入**:
- `app/commands/tab_context_store.go` (新建)

**输出**:
- `app/commands/tab_context_store.go`

**实现要求**:

```go
// app/commands/tab_context_store.go

// BrowserContext represents an isolated browser context
type BrowserContext struct {
    ID           string
    InstanceID   string
    Fingerprint  *fingerprint.Fingerprint
    ProxyURL     string
    TabIDs       []string
    CreatedAt    time.Time
    LastActiveAt time.Time
}

// TabInfo represents a browser tab
type TabInfo struct {
    ID              string
    ContextID       string
    InstanceID      string
    URL             string
    Title           string
    FingerprintSeed string
    CreatedAt       time.Time
    LastActiveAt    time.Time
}

// ContextStore holds all contexts for an instance
type ContextStore struct {
    mu       sync.Mutex
    contexts map[string]*BrowserContext
    tabs     map[string]*TabInfo
}

func NewContextStore() *ContextStore {
    return &ContextStore{
        contexts: make(map[string]*BrowserContext),
        tabs:     make(map[string]*TabInfo),
    }
}

func (s *ContextStore) AddContext(contextId string, fp *fingerprint.Fingerprint, proxyURL string) { ... }
func (s *ContextStore) AddTab(targetId string, tab *TabInfo) { ... }
func (s *ContextStore) RemoveTab(targetId string) { ... }
func (s *ContextStore) RemoveContext(contextId string) { ... }
func (s *ContextStore) GetTab(targetId string) *TabInfo { ... }
func (s *ContextStore) ListTabs() []*TabInfo { ... }
func (s *ContextStore) GetContext(contextId string) *BrowserContext { ... }
func (s *ContextStore) CanCloseContext(contextId string) bool { ... }

// CloseAll cleanup all contexts and tabs when instance stops
func (s *ContextStore) CloseAll(ctx context.Context, mainClient CDPClientInterface) error { ... }
```

**验收标准**:
- [ ] AddContext / RemoveContext 正确管理 contexts map
- [ ] AddTab / RemoveTab 正确管理 tabs map
- [ ] ListTabs 返回所有标签页
- [ ] CloseAll 正确清理所有资源

**测试要求**:
- 测试文件: `app/commands/tab_context_store_test.go`
- 测试用例: ≤ 8个
- 覆盖率: ≥ 80%

**预估工时**: 0.5天

**依赖**: T-01

---

### T-03: TabService 实现

**任务概述**: 实现 TabService，提供标签页生命周期管理

**对应架构**: `per-tab-fingerprint.md` 第 4.1 节

**输入**:
- `app/commands/tab.go` (新建)
- `instance/cdp.go`

**输出**:
- `app/commands/tab.go`

**实现要求**:

```go
// app/commands/tab.go

type TabService struct {
    instanceSvc    *InstanceService
    contextStores sync.Map // instanceID → *ContextStore
}

func NewTabService(instanceSvc *InstanceService) *TabService {
    return &TabService{
        instanceSvc: instanceSvc,
        contextStores: sync.Map{},
    }
}

func (s *TabService) getOrCreateContextStore(instanceID string) *ContextStore {
    if store, ok := s.contextStores.Load(instanceID); ok {
        return store.(*ContextStore)
    }
    store := NewContextStore()
    s.contextStores.Store(instanceID, store)
    return store
}

func (s *TabService) CreateTab(ctx context.Context, instanceID string, cfg *TabConfig) (*TabInfo, error) {
    // 1. Verify instance is running
    inst, err := s.instanceSvc.GetInstance(ctx, instanceID)
    if err != nil {
        return nil, err
    }
    if inst.Status != instance.StatusRunning {
        return nil, fmt.Errorf("instance is not running")
    }

    // 2. Get main CDP client
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

    // 4. Create tab within that context
    targetId, err := mainClient.CreateTargetWithContext(ctx, cfg.URL, contextId)
    if err != nil {
        _ = mainClient.CloseBrowserContext(ctx, contextId)
        return nil, fmt.Errorf("failed to create tab: %w", err)
    }

    // 5. Store context and tab info
    tabInfo := &TabInfo{
        ID:              targetId,
        ContextID:       contextId,
        InstanceID:      instanceID,
        URL:             cfg.URL,
        CreatedAt:       time.Now(),
        LastActiveAt:    time.Now(),
    }

    store := s.getOrCreateContextStore(instanceID)
    store.AddContext(contextId, cfg.Fingerprint, cfg.ProxyURL)
    store.AddTab(targetId, tabInfo)

    return tabInfo, nil
}

func (s *TabService) CloseTab(ctx context.Context, instanceID, tabID string) error { ... }
func (s *TabService) ListTabs(ctx context.Context, instanceID string) ([]*TabInfo, error) { ... }
func (s *TabService) NavigateTab(ctx context.Context, instanceID, tabID, url string) error { ... }
```

**验收标准**:
- [ ] CreateTab 创建独立的 BrowserContext 和 Tab
- [ ] CloseTab 正确关闭 Tab 和 Context
- [ ] ListTabs 返回所有标签页
- [ ] NavigateTab 导航指定标签页

**测试要求**:
- 测试文件: `app/commands/tab_test.go`
- 测试用例: ≤ 10个
- 覆盖率: ≥ 80%

**预估工时**: 1天

**依赖**: T-02

---

### T-04: InstanceService context 集成

**任务概述**: 在 InstanceService 中集成 contextStores，修改 StopInstance 清理逻辑

**对应架构**: `per-tab-fingerprint.md` 第 5.2 节

**输入**:
- `app/commands/instance.go`

**输出**:
- `app/commands/instance.go` (修改)

**实现要求**:

```go
// InstanceService 新增字段
type InstanceService struct {
    manager      browserRuntimeManager
    store        *sqlite.InstanceStore
    cdpClients   sync.Map
    targetURLs   sync.Map

    // NEW: Per-instance context management
    contextStores sync.Map // instanceID → *ContextStore
}

// NEW: GetCDPClientForTab gets a CDP client for a specific tab
func (s *InstanceService) GetCDPClientForTab(ctx context.Context, instanceID, tabID string) (instance.CDPClientInterface, error) {
    inst, err := s.store.Get(instanceID)
    if err != nil {
        return nil, err
    }

    wsURL := inst.CDPEndpoint
    if !strings.HasPrefix(wsURL, "ws://") {
        wsURL = "ws://" + wsURL
    }

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

// Modify StopInstance to cleanup contexts
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

**验收标准**:
- [ ] GetCDPClientForTab 正确获取指定 tab 的 CDP client
- [ ] StopInstance 调用时清理所有 contexts
- [ ] contextStores 正确管理每个实例的 ContextStore

**测试要求**:
- 集成测试为主
- 测试用例: ≤ 5个

**预估工时**: 0.5天

**依赖**: T-03

---

### T-05: Wails 绑定 (app.go)

**任务概述**: 在 app.go 中新增 Tab 相关 Wails 绑定

**对应架构**: `per-tab-fingerprint.md` 第 4.2 节

**输入**:
- `app.go`
- `app/commands/tab.go`

**输出**:
- `app.go` (修改)

**实现要求**:

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

**验收标准**:
- [ ] 所有 Tab 相关方法正确暴露给前端
- [ ] Wails 绑定生成成功

**测试要求**:
- 通过前端调用验证绑定正确性

**预估工时**: 0.25天

**依赖**: T-04

---

### T-06: TabFingerprintSelector 组件

**任务概述**: 实现前端标签页选择组件

**对应架构**: `per-tab-fingerprint.md` 第 8.1 节

**输入**:
- `frontend/src/components/TabFingerprintSelector.tsx` (新建)
- Wails 绑定类型

**输出**:
- `frontend/src/components/TabFingerprintSelector.tsx`

**实现要求**:

```tsx
// frontend/src/components/TabFingerprintSelector.tsx

interface TabFingerprintSelectorProps {
    instanceId: string;
    onTabCreated?: (tab: commands.TabInfo) => void;
    onTabClosed?: (tabId: string) => void;
    onTabNavigated?: (tabId: string, url: string) => void;
}

export function TabFingerprintSelector({ instanceId, onTabCreated, onTabClosed, onTabNavigated }: TabFingerprintSelectorProps) {
    const [tabs, setTabs] = useState<commands.TabInfo[]>([]);
    const [loading, setLoading] = useState(false);
    const [selectedTabId, setSelectedTabId] = useState<string | null>(null);
    const [showCreate, setShowCreate] = useState(false);
    const [url, setUrl] = useState('');
    const [error, setError] = useState<string | null>(null);

    // 加载标签页列表
    useEffect(() => {
        loadTabs();
    }, [instanceId]);

    async function loadTabs() {
        try {
            const list = await ListTabs(instanceId);
            setTabs(list || []);
            setError(null);
        } catch (err) {
            setError(String(err));
        }
    }

    async function createTab() {
        try {
            setLoading(true);
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
            onTabCreated?.(tab);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    }

    async function closeTab(tabId: string) {
        try {
            await CloseTab(instanceId, tabId);
            setTabs(prev => prev.filter(t => t.id !== tabId));
            if (selectedTabId === tabId) {
                setSelectedTabId(tabs[0]?.id || null);
            }
            onTabClosed?.(tabId);
        } catch (err) {
            setError(String(err));
        }
    }

    async function navigateTab(tabId: string, navigateUrl: string) {
        try {
            await NavigateTab(instanceId, tabId, navigateUrl);
            setTabs(prev => prev.map(t =>
                t.id === tabId ? { ...t, url: navigateUrl } : t
            ));
            onTabNavigated?.(tabId, navigateUrl);
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
                        <span className="tab-fp">{tab.fingerprint_seed?.slice(0, 8) || 'N/A'}</span>
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
                        <div className="form-group">
                            <label>URL</label>
                            <input
                                type="text"
                                placeholder="https://example.com"
                                value={url}
                                onChange={e => setUrl(e.target.value)}
                            />
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

**验收标准**:
- [ ] 标签页列表正确显示
- [ ] 创建标签页弹窗正常工作
- [ ] 关闭标签页功能正常
- [ ] 错误处理完整

**测试要求**:
- 组件测试: `TabFingerprintSelector.test.tsx`
- 测试用例: ≤ 8个

**预估工时**: 0.5天

**依赖**: T-05

---

### T-07: Wails 类型重新生成

**任务概述**: 重新生成 Wails 类型声明和模型

**输入**:
- `app.go`
- `frontend/src/wailsjs/go/main/App.d.ts`

**输出**:
- `frontend/src/wailsjs/go/main/App.d.ts` (重新生成)
- `frontend/src/wailsjs/go/models/commands.ts` (更新)

**实现要求**:

运行 `wails generate bindings` 重新生成 TypeScript 类型声明。

新增类型:

```typescript
// frontend/src/wailsjs/go/models/commands.ts

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
    id: string;
    context_id: string;
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

**验收标准**:
- [ ] `CreateTab`, `CloseTab`, `ListTabs`, `NavigateTab` 类型声明正确
- [ ] `BrowserContext`, `TabInfo`, `TabConfig` 模型正确

**测试要求**:
- 无

**预估工时**: 0.25天

**依赖**: T-05

---

### T-08: 集成测试验证

**任务概述**: 对整个 per-tab fingerprint 功能进行集成测试

**对应架构**: `per-tab-fingerprint.md` 第 12 节

**输入**:
- 所有已实现的模块
- 实际浏览器实例

**输出**:
- 集成测试结果

**测试场景**:

| 场景编号 | 场景 | 前置条件 | 测试步骤 | 预期结果 |
|---------|------|---------|---------|---------|
| TC-001 | 创建 3 个不同指纹的标签页 | 实例运行中 | 1. 调用 CreateTab 3次 | 每个标签页有独立 contextId |
| TC-002 | 关闭中间标签页 | 3个标签页运行中 | 1. 调用 CloseTab | 其他2个标签页不受影响 |
| TC-003 | 实例停止时清理 | 3个标签页运行中 | 1. 调用 StopInstance | 所有 contexts 正确关闭 |
| TC-004 | 标签页导航 | 标签页存在 | 1. 调用 NavigateTab | URL 更新 |

**验收标准**:
- [ ] TC-001 通过
- [ ] TC-002 通过
- [ ] TC-003 通过
- [ ] TC-004 通过

**测试要求**:
- 集成测试文件: `tests/integration/tab_test.go`
- E2E 测试文件: `tests/e2e/tab.spec.ts`
- 测试用例: ≤ 10个

**预估工时**: 0.5天

**依赖**: T-06, T-07

---

## 5. 执行顺序

```
T-01 (0.5d) ──┐
                ├── T-02 (0.5d) ──┐
                │                 ├── T-03 (1d) ──┐
                │                 │               ├── T-04 (0.5d) ──┐
                │                 │               │                 ├── T-05 (0.25d) ──┐
                │                 │               │                 │                   ├── T-06 (0.5d) ──┐
                │                 │               │                 │                   │                   ├── T-07 (0.25d) ──┤
                │                 │               │                 │                   │                   │                   │
                │                 │               │                 │                   │                   │                   └── T-08 (0.5d)
                │                 │               │                 │                   │                   │
                └─────────────────┴───────────────┴─────────────────┴───────────────────┴───────────────────┘

总计: ~3.5 天
```

---

## 6. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| CDP 连接管理复杂 | tab 关闭后连接可能 stale | 使用完毕后立即关闭连接 |
| 内存泄漏 | contexts 未正确关闭 | StopInstance 时强制 CloseAll |
| 并发问题 | 多个 tab 操作并发 | ContextStore 使用 mutex 保护 |

---

## 7. 验收清单

### 7.1 功能验收

- [ ] T-01: CDPClient BrowserContext 扩展完成
- [ ] T-02: ContextStore 实现完成
- [ ] T-03: TabService 实现完成
- [ ] T-04: InstanceService context 集成完成
- [ ] T-05: Wails 绑定完成
- [ ] T-06: TabFingerprintSelector 组件完成
- [ ] T-07: Wails 类型重新生成完成
- [ ] T-08: 集成测试验证完成

### 7.2 质量验收

- [ ] 代码覆盖率 ≥ 80%
- [ ] 无 lint 错误
- [ ] 无 panic 或未处理错误

---

## 8. 覆盖映射

| 架构元素 | 架构编号 | 任务 | 覆盖状态 |
|---------|---------|------|---------|
| BrowserContext 数据结构 | DATA-CTX-001 | T-02 | ✅ |
| TabInfo 数据结构 | DATA-TAB-001 | T-02 | ✅ |
| CreateTab API | API-TAB-001 | T-03, T-05 | ✅ |
| CloseTab API | API-TAB-002 | T-03, T-05 | ✅ |
| ListTabs API | API-TAB-003 | T-03, T-05 | ✅ |
| NavigateTab API | API-TAB-004 | T-03, T-05 | ✅ |
| CDP BrowserContext | CDP-001 | T-01 | ✅ |
| TabFingerprintSelector | UI-TAB-001 | T-06 | ✅ |
| 生命周期管理 | LIFE-001 | T-03, T-04 | ✅ |
| 错误处理 | ERR-001 | T-03 | ✅ |
