# 后端开发计划 - AI 自动化模块

## 文档信息

| 字段 | 内容 |
|------|------|
| **模块编号** | MOD-06 |
| **模块名称** | AI 自动化模块 |
| **对应架构** | docs/v1/02-architecture/03-mod-06-ai.md |
| **优先级** | P0 |
| **预估工时** | 4 天 |

---

## 1. 模块概述

### 1.1 模块职责

- AI 内容生成
- 批量发布管理
- 养号策略自动化
- MCP 服务端
- REST API

### 1.2 对应 PRD

| PRD编号 | 功能 |
|---------|-----|
| FR-007 | AI 内容生成 |
| FR-008 | 批量发布管理 |
| FR-009 | 账号养号自动化 |
| FR-010 | MCP 服务端 |
| FR-011 | REST API |

---

## 2. 技术设计

### 2.1 技术栈

| 类型 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.21+ |
| 数据库 | PostgreSQL | 15+ |
| 协议 | MCP v1.0 | - |
| AI 客户端 | OpenAI / Claude API | - |
| 测试 | Go testing | - |

### 2.2 目录结构

```
internal/mcp/
├── server.go              # MCP 服务端主逻辑
├── types.go               # MCP 类型定义
├── tools.go               # MCP 工具定义
├── handler.go             # 请求处理器
└── server_test.go         # 单元测试

internal/content/
├── types.go               # 内容数据结构
├── generator.go           # AI 内容生成器
├── store.go               # 内容存储
└── generator_test.go      # 单元测试

internal/publish/
├── types.go               # 发布任务数据结构
├── scheduler.go           # 发布调度器
└── scheduler_test.go      # 单元测试

internal/interaction/
├── types.go               # 互动任务数据结构
├── engine.go              # 养号策略引擎
└── engine_test.go         # 单元测试

internal/ai/
└── client.go              # AI 客户端封装
```

---

## 3. 接口清单

### 3.1 MCP 工具

| 任务编号 | 工具名称 | 描述 | 复杂度 |
|---------|---------|-----|-------|
| T-01 | create_post | 创建 TikTok 帖子 | 中 |
| T-02 | generate_content | AI 生成内容文案 | 中 |
| T-03 | batch_publish | 批量发布到多个账号 | 高 |
| T-04 | interact | 执行互动操作 | 中 |
| T-05 | get_account_stats | 获取账号统计数据 | 低 |

### 3.2 REST API

| 任务编号 | 方法 | 路径 | 描述 | 复杂度 |
|---------|------|------|-----|-------|
| T-06 | POST | /api/v1/content | 创建内容 | 低 |
| T-07 | GET | /api/v1/content/{id} | 获取内容 | 低 |
| T-08 | GET | /api/v1/content | 内容列表 | 低 |
| T-09 | DELETE | /api/v1/content/{id} | 删除内容 | 低 |
| T-10 | POST | /api/v1/publish | 创建发布任务 | 中 |
| T-11 | GET | /api/v1/publish/{id} | 获取任务状态 | 低 |
| T-12 | POST | /api/v1/publish/{id}/cancel | 取消任务 | 低 |

---

## 4. 数据结构

### 4.1 Content

```go
type Content struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Tags        []string  `json:"tags"`
    VideoPath   string    `json:"video_path"`
    CreatedBy   string    `json:"created_by"` // AI/user
    CreatedAt   time.Time `json:"created_at"`
}
```

### 4.2 PublishTask

```go
type PublishTask struct {
    ID          string           `json:"id"`
    ContentID   string           `json:"content_id"`
    AccountIDs  []string         `json:"account_ids"`
    Status      PublishStatus    `json:"status"`
    ScheduledAt time.Time        `json:"scheduled_at"`
    Results     []*PublishResult `json:"results"`
    CreatedAt   time.Time       `json:"created_at"`
}
```

---

## 5. 开发任务拆分

### T-01: MCP 服务端核心实现

**任务概述**: 实现 MCP 服务端核心逻辑

**对应架构**:
- 接口规约: FR-010 (MCP 服务端)
- MCP 协议: v1.0

**输出**:
- `internal/mcp/server.go`
- `internal/mcp/types.go`

**实现要求**:

```go
// types.go
type MCPToolDefinition struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema *MCPToolInputSchema    `json:"inputSchema"`
}

type MCPRequest struct {
    Tool      string                 `json:"tool"`
    Arguments map[string]interface{}  `json:"arguments"`
}

type MCPResponse struct {
    IsError bool                    `json:"isError"`
    Content []interface{}            `json:"content"`
}

// server.go
type MCPServer struct {
    tools    []MCPToolDefinition
    handlers map[string]ToolHandler
}

func (s *MCPServer) HandleRequest(ctx context.Context, req *MCPRequest) (*MCPResponse, error) {
    handler, ok := s.handlers[req.Tool]
    if !ok {
        return nil, fmt.Errorf("unknown tool: %s", req.Tool)
    }

    result, err := handler(ctx, req.Arguments)
    if err != nil {
        return &MCPResponse{
            IsError: true,
            Content: []interface{}{err.Error()},
        }, nil
    }

    return &MCPResponse{
        IsError: false,
        Content: []interface{}{result},
    }, nil
}

type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)
```

**验收标准**:
- [ ] **协议实现**: MCP v1.0 协议正确实现
- [ ] **工具注册**: 工具注册机制正常
- [ ] **请求处理**: 请求处理流程正确

**预估工时**: 0.5 天

**依赖**: 无

---

### T-02: MCP 工具定义与处理器

**任务概述**: 实现 MCP 工具定义和处理器

**对应架构**:
- 接口规约: FR-010
- MCP 工具: create_post, generate_content, batch_publish, interact, get_account_stats

**输入**:
- T-01 的 MCP 服务端

**输出**:
- `internal/mcp/tools.go`
- `internal/mcp/handler.go`

**实现要求**:

```go
// tools.go
var MCPTools = []MCPToolDefinition{
    {
        Name: "create_post",
        Description: "创建 TikTok 帖子",
        InputSchema: &MCPToolInputSchema{
            Type: "object",
            Properties: map[string]*MCPProperty{
                "account_id": {"type": "string", "description": "账号 ID"},
                "video_path": {"type": "string", "description": "视频文件路径"},
                "title": {"type": "string", "description": "视频标题"},
                "tags": {"type": "array", "description": "标签列表"},
            },
            Required: []string{"account_id", "video_path"},
        },
    },
    // ... 其他工具定义
}

// handler.go
func (s *MCPServer) registerTools() {
    s.handlers["create_post"] = s.handleCreatePost
    s.handlers["generate_content"] = s.handleGenerateContent
    s.handlers["batch_publish"] = s.handleBatchPublish
    s.handlers["interact"] = s.handleInteract
    s.handlers["get_account_stats"] = s.handleGetStats
}

func (s *MCPServer) handleCreatePost(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    accountID := args["account_id"].(string)
    videoPath := args["video_path"].(string)
    title := args["title"].(string)
    tags := args["tags"].([]interface{})

    // 创建发布任务
    task := &PublishTask{
        ID:         uuid.New().String(),
        ContentID:  videoPath, // 实际应该是 content ID
        AccountIDs: []string{accountID},
        Status:     PublishStatusPending,
    }

    // 执行发布
    err := s.publishScheduler.Schedule(ctx, task)
    if err != nil {
        return nil, err
    }

    return map[string]interface{}{
        "task_id": task.ID,
        "status":  task.Status,
    }, nil
}
```

**验收标准**:
- [ ] **工具完整性**: 所有 5 个 MCP 工具实现
- [ ] **参数验证**: 参数验证正确
- [ ] **错误处理**: 错误处理完善

**预估工时**: 0.5 天

**依赖**: T-01

---

### T-03: AI 客户端封装

**任务概述**: 实现 AI 客户端封装

**对应架构**:
- AI 客户端: OpenAI / Claude API

**输出**:
- `internal/ai/client.go`

**实现要求**:

```go
type AIClient interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    GenerateContent(ctx context.Context, req *GenerateRequest) (*Content, error)
}

type OpenAIClient struct {
    apiKey  string
    baseURL string
    client  *http.Client
}

type ChatRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatResponse struct {
    Content string `json:"content"`
}

func (c *OpenAIClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    choices := result["choices"].([]interface{})
    content := choices[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)

    return &ChatResponse{Content: content}, nil
}
```

**验收标准**:
- [ ] **API 调用**: OpenAI/Claude API 调用正常
- [ ] **错误处理**: 网络错误、超时处理正确
- [ ] **重试机制**: API 失败重试机制正常

**预估工时**: 0.25 天

**依赖**: 无

---

### T-04: AI 内容生成器实现

**任务概述**: 实现 AI 内容生成器

**对应架构**:
- 接口规约: FR-007
- 数据结构: Content

**输入**:
- T-03 的 AI 客户端

**输出**:
- `internal/content/generator.go`
- `internal/content/types.go`

**实现要求**:

```go
// types.go
type Content struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Tags        []string  `json:"tags"`
    VideoPath   string    `json:"video_path"`
    CreatedBy   string    `json:"created_by"` // AI/user
    CreatedAt   time.Time `json:"created_at"`
}

type GenerateRequest struct {
    Topic  string `json:"topic"`
    Style  string `json:"style"` // funny/serious/educational
    Length int    `json:"length"` // 字符数
}

// generator.go
type ContentGenerator struct {
    aiClient AIClient
}

func (g *ContentGenerator) Generate(ctx context.Context, req *GenerateRequest) (*Content, error) {
    prompt := fmt.Sprintf(`
        生成 TikTok 视频文案:
        主题: %s
        风格: %s
        长度: %d 字符

        返回 JSON 格式:
        {
            "title": "标题 (最多100字符)",
            "description": "描述 (最多2200字符)",
            "tags": ["标签1", "标签2", "标签3", "标签4", "标签5"]
        }
    `, req.Topic, req.Style, req.Length)

    resp, err := g.aiClient.Chat(ctx, &ChatRequest{
        Model: "gpt-4",
        Messages: []Message{
            {Role: "user", Content: prompt},
        },
    })
    if err != nil {
        return nil, err
    }

    var content Content
    json.Unmarshal([]byte(resp.Content), &content)
    content.ID = uuid.New().String()
    content.CreatedBy = "AI"
    content.CreatedAt = time.Now()

    return &content, nil
}
```

**验收标准**:
- [ ] **内容生成**: AI 生成内容正常
- [ ] **格式正确**: JSON 格式解析正确
- [ ] **标签生成**: 标签数量正确 (5个)

**预估工时**: 0.5 天

**依赖**: T-03

---

### T-05: 内容存储层实现

**任务概述**: 实现内容数据存储

**对应架构**:
- 数据结构: Content

**输出**:
- `internal/content/store.go`

**实现要求**:

```go
type Store interface {
    Save(content *Content) (*Content, error)
    Get(id string) (*Content, error)
    List(filter *ContentFilter) ([]*Content, error)
    Delete(id string) error
}

type PostgresStore struct {
    db *sql.DB
}

func (s *PostgresStore) Save(content *Content) (*Content, error) {
    query := `
        INSERT INTO contents (id, title, description, tags, video_path, created_by, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING *
    `
    err := s.db.QueryRow(query,
        content.ID, content.Title, content.Description,
        content.Tags, content.VideoPath, content.CreatedBy, content.CreatedAt,
    ).Scan(/* ... */)
    return content, err
}
```

**验收标准**:
- [ ] **CRUD 功能**: 所有 CRUD 操作正确
- [ ] **索引**: 支持按 created_by 筛选

**预估工时**: 0.25 天

**依赖**: T-04

---

### T-06: 批量发布调度器实现

**任务概述**: 实现批量发布调度器

**对应架构**:
- 接口规约: FR-008

**输入**:
- MOD-02 实例管理模块
- MOD-05 账号注册模块

**输出**:
- `internal/publish/scheduler.go`
- `internal/publish/types.go`

**实现要求**:

```go
// types.go
type PublishStatus string

const (
    PublishStatusPending   PublishStatus = "pending"
    PublishStatusRunning  PublishStatus = "running"
    PublishStatusCompleted PublishStatus = "completed"
    PublishStatusFailed   PublishStatus = "failed"
    PublishStatusCancelled PublishStatus = "cancelled"
)

type PublishTask struct {
    ID          string           `json:"id"`
    ContentID   string           `json:"content_id"`
    AccountIDs  []string         `json:"account_ids"`
    Status      PublishStatus    `json:"status"`
    ScheduledAt time.Time        `json:"scheduled_at"`
    Results     []*PublishResult `json:"results"`
    CreatedAt   time.Time        `json:"created_at"`
}

type PublishResult struct {
    AccountID string `json:"account_id"`
    Status    string `json:"status"`
    Error     string `json:"error,omitempty"`
}

// scheduler.go
type PublishScheduler struct {
    accountManager account.AccountManager
    instanceManager instance.InstanceManager
    contentStore   content.Store
}

func (s *PublishScheduler) Schedule(ctx context.Context, task *PublishTask) error {
    task.Status = PublishStatusRunning

    // 遍历账号列表
    for _, accountID := range task.AccountIDs {
        // 创建实例
        instance, err := s.instanceManager.Create(ctx, &instance.InstanceConfig{
            AccountID: accountID,
        })
        if err != nil {
            task.Results = append(task.Results, &PublishResult{
                AccountID: accountID,
                Status:    "failed",
                Error:     err.Error(),
            })
            continue
        }

        // 执行发布
        if err := s.publish(instance, task.ContentID); err != nil {
            task.Results = append(task.Results, &PublishResult{
                AccountID: accountID,
                Status:    "failed",
                Error:     err.Error(),
            })
        } else {
            task.Results = append(task.Results, &PublishResult{
                AccountID: accountID,
                Status:    "success",
            })
        }

        // 销毁实例
        s.instanceManager.Destroy(ctx, instance.ID)
    }

    task.Status = PublishStatusCompleted
    return nil
}
```

**验收标准**:
- [ ] **发布功能**: 发布流程正常
- [ ] **状态更新**: 任务状态正确更新
- [ ] **结果记录**: 发布结果正确记录

**预估工时**: 0.5 天

**依赖**: T-05

---

### T-07: 养号策略引擎实现

**任务概述**: 实现养号策略引擎

**对应架构**:
- 接口规约: FR-009

**输入**:
- MOD-02 实例管理模块

**输出**:
- `internal/interaction/engine.go`
- `internal/interaction/types.go`

**实现要求**:

```go
// types.go
type InteractionType string

const (
    InteractionTypeLike    InteractionType = "like"
    InteractionTypeFollow InteractionType = "follow"
    InteractionTypeComment InteractionType = "comment"
)

type InteractionTask struct {
    AccountID string           `json:"account_id"`
    Type      InteractionType  `json:"type"`
    TargetURL string           `json:"target_url"`
    Count     int              `json:"count"`
}

// engine.go
type InteractionEngine struct {
    instanceManager instance.InstanceManager
}

func (e *InteractionEngine) Run(ctx context.Context, task *InteractionTask) error {
    instance, err := e.instanceManager.Get(ctx, task.AccountID)
    if err != nil {
        return err
    }

    cdp, err := e.instanceManager.GetCDPClient(ctx, instance.ID)
    if err != nil {
        return err
    }

    switch task.Type {
    case InteractionTypeLike:
        return e.likeVideos(cdp, task.TargetURL, task.Count)
    case InteractionTypeFollow:
        return e.followUsers(cdp, task.TargetURL, task.Count)
    case InteractionTypeComment:
        return e.commentVideos(cdp, task.TargetURL, task.Count)
    }
    return nil
}

func (e *InteractionEngine) likeVideos(cdp CDPClient, targetURL string, count int) error {
    for i := 0; i < count; i++ {
        cdp.Navigate(ctx, targetURL)
        time.Sleep(randomDelay(2*time.Second, 5*time.Second))

        cdp.Evaluate(ctx, "window.scrollTo(0, 300)")
        cdp.Click(ctx, ".like-button")
        time.Sleep(randomDelay(1*time.Second, 3*time.Second))
    }
    return nil
}

func randomDelay(min, max time.Duration) time.Duration {
    return time.Duration(rand.Int63n(int64(max-min))) + min
}
```

**验收标准**:
- [ ] **点赞功能**: 点赞操作正常
- [ ] **关注功能**: 关注操作正常
- [ ] **评论功能**: 评论操作正常
- [ ] **随机延迟**: 模拟人类行为延迟正常

**预估工时**: 0.5 天

**依赖**: T-06

---

### T-08: REST API 实现

**任务概述**: 实现 REST API Handler

**对应架构**:
- 接口规约: FR-011

**输出**:
- `internal/api/content.go`
- `internal/api/publish.go`

**实现要求**:

```go
// content.go
func CreateContent(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Title       string   `json:"title"`
        Description string   `json:"description"`
        Tags        []string `json:"tags"`
        VideoPath   string   `json:"video_path"`
        CreatedBy   string   `json:"created_by"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    content := &Content{
        ID:          uuid.New().String(),
        Title:       req.Title,
        Description: req.Description,
        Tags:        req.Tags,
        VideoPath:   req.VideoPath,
        CreatedBy:   req.CreatedBy,
        CreatedAt:   time.Now(),
    }

    saved, err := contentStore.Save(content)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    json.NewEncoder(w).Encode(saved)
}

func GetContent(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    content, err := contentStore.Get(id)
    if err != nil {
        http.Error(w, "not found", 404)
        return
    }
    json.NewEncoder(w).Encode(content)
}

// publish.go
func CreatePublishTask(w http.ResponseWriter, r *http.Request) {
    var req struct {
        ContentID   string    `json:"content_id"`
        AccountIDs  []string  `json:"account_ids"`
        ScheduledAt time.Time `json:"scheduled_at"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    task := &PublishTask{
        ID:          uuid.New().String(),
        ContentID:   req.ContentID,
        AccountIDs:  req.AccountIDs,
        Status:      PublishStatusPending,
        ScheduledAt: req.ScheduledAt,
        CreatedAt:   time.Now(),
    }

    go scheduler.Schedule(context.Background(), task)

    json.NewEncoder(w).Encode(task)
}
```

**验收标准**:
- [ ] **API 完整性**: 所有 API 端点实现
- [ ] **JSON 处理**: 请求/响应 JSON 正确
- [ ] **错误处理**: 错误响应正确

**预估工时**: 0.5 天

**依赖**: T-05, T-06

---

### T-09 ~ T-12: 单元测试

**任务概述**: 完整的测试覆盖

**测试用例**:
1. TC-M6-01: MCP 工具调用成功
2. TC-M6-02: AI 内容生成成功
3. TC-M6-03: 批量发布成功
4. TC-M6-04: 互动操作成功
5. TC-M6-05: REST API 端点测试

**验收标准**:
- [ ] **覆盖率**: ≥ 80%

**预估工时**: 0.5 天

**依赖**: T-01 ~ T-08

---

## 6. 验收清单

### 6.1 功能验收

- [ ] 所有 MCP 工具实现完成
- [ ] AI 内容生成正常
- [ ] 批量发布正常
- [ ] 养号策略正常
- [ ] REST API 正常
- [ ] 所有单元测试通过

### 6.2 质量验收

- [ ] 测试覆盖率 ≥ 80%

---

## 7. 覆盖映射

| 架构元素 | 架构编号 | 任务 | 覆盖状态 |
|---------|---------|------|---------|
| MCP工具 | create_post | T-02 | ✅ |
| MCP工具 | generate_content | T-02, T-04 | ✅ |
| MCP工具 | batch_publish | T-02, T-06 | ✅ |
| MCP工具 | interact | T-02, T-07 | ✅ |
| MCP工具 | get_account_stats | T-02 | ✅ |
| REST API | /api/v1/content | T-08 | ✅ |
| REST API | /api/v1/publish | T-08 | ✅ |
| 功能 | FR-007 AI内容生成 | T-04 | ✅ |
| 功能 | FR-008 批量发布 | T-06 | ✅ |
| 功能 | FR-009 养号自动化 | T-07 | ✅ |
| 功能 | FR-010 MCP服务端 | T-01, T-02 | ✅ |
| 功能 | FR-011 REST API | T-08 | ✅ |
