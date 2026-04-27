# MOD-06: AI 自动化模块

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | TikTok Matrix Operations System (TMOS) |
| **模块编号** | MOD-06 |
| **模块名称** | AI 自动化模块 |
| **版本** | v1.0 |
| **对应PRD** | FR-007, FR-008, FR-009, FR-010, FR-011 |
| **更新日期** | 2026-04-21 |

---

## 1. 系统定位

### 1.1 在整体架构中的位置

```
┌─────────────────────────────────────────────────────┐
│                   AI Agent (外部)                    │
└───────────────────────┬─────────────────────────────┘
                        │ ▼ MCP / REST
┌─────────────────────────────────────────────────────┐
│              ★ MOD-06 AI 自动化模块 ★                │
│                                                       │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│   │ MCP服务  │  │ REST API │  │ 内容生成 │        │
│   └──────────┘  └──────────┘  └──────────┘        │
└─────────────────────────────────────────────────────┘
                        │ ▼ 调用
┌─────────────────────────────────────────────────────┐
│                   MOD-05 账号注册模块                  │
└─────────────────────────────────────────────────────┘
```

### 1.2 核心职责

- AI 内容生成
- 批量发布管理
- 养号策略自动化
- MCP 服务端
- REST API

---

## 2. 对应PRD

| PRD章节 | 编号 | 内容 |
|---------|-----|------|
| 功能需求 | FR-007 | AI 内容生成 |
| 功能需求 | FR-008 | 批量发布管理 |
| 功能需求 | FR-009 | 账号养号自动化 |
| 功能需求 | FR-010 | MCP 服务端 |
| 功能需求 | FR-011 | REST API |

---

## 3. 接口定义

### 3.1 MCP 工具定义

```go
// MCP 服务端暴露的工具
// 文件: internal/mcp/server.go

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
    {
        Name: "generate_content",
        Description: "AI 生成内容文案",
        InputSchema: &MCPToolInputSchema{
            Type: "object",
            Properties: map[string]*MCPProperty{
                "topic": {"type": "string", "description": "内容主题"},
                "style": {"type": "string", "description": "风格 (funny/serious/educational)"},
                "length": {"type": "integer", "description": "长度 (字符数)"},
            },
            Required: []string{"topic"},
        },
    },
    {
        Name: "batch_publish",
        Description: "批量发布到多个账号",
        InputSchema: &MCPToolInputSchema{
            Type: "object",
            Properties: map[string]*MCPProperty{
                "content_id": {"type": "string", "description": "内容 ID"},
                "account_ids": {"type": "array", "description": "账号 ID 列表"},
                "scheduled_at": {"type": "string", "description": "定时发布时间"},
            },
            Required: []string{"content_id", "account_ids"},
        },
    },
    {
        Name: "interact",
        Description: "执行互动操作 (点赞/评论/关注)",
        InputSchema: &MCPToolInputSchema{
            Type: "object",
            Properties: map[string]*MCPProperty{
                "account_id": {"type": "string", "description": "账号 ID"},
                "action": {"type": "string", "description": "操作类型 (like/comment/follow)"},
                "target": {"type": "string", "description": "目标 URL 或用户名"},
            },
            Required: []string{"account_id", "action", "target"},
        },
    },
    {
        Name: "get_account_stats",
        Description: "获取账号统计数据",
        InputSchema: &MCPToolInputSchema{
            Type: "object",
            Properties: map[string]*MCPProperty{
                "account_id": {"type": "string", "description": "账号 ID"},
            },
            Required: []string{"account_id"},
        },
    },
}
```

### 3.2 Content 数据结构

```go
// Content 内容
// 文件: internal/content/types.go
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

### 3.3 REST API 路由

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/content | 创建内容 |
| GET | /api/v1/content/{id} | 获取内容 |
| GET | /api/v1/content | 内容列表 |
| DELETE | /api/v1/content/{id} | 删除内容 |
| POST | /api/v1/publish | 创建发布任务 |
| GET | /api/v1/publish/{id} | 获取任务状态 |
| POST | /api/v1/publish/{id}/cancel | 取消任务 |
| POST | /api/v1/interaction | 执行互动 |
| GET | /api/v1/interaction/history | 互动历史 |
| GET | /api/v1/stats/account/{id} | 账号统计 |

---

## 4. 核心设计

### 4.1 AI 内容生成

```go
// ContentGenerator AI 内容生成器
// 文件: internal/content/generator.go
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

### 4.2 批量发布调度

```go
// PublishScheduler 发布调度器
// 文件: internal/publish/scheduler.go
type PublishScheduler struct {
    accountManager AccountManager
    instanceManager InstanceManager
    contentStore   Store
}

func (s *PublishScheduler) Schedule(ctx context.Context, task *PublishTask) error {
    // 1. 验证内容存在
    content, err := s.contentStore.Get(task.ContentID)
    if err != nil {
        return err
    }

    // 2. 遍历账号列表
    for _, accountID := range task.AccountIDs {
        // 创建实例
        instance, err := s.instanceManager.Create(ctx, &InstanceConfig{
            AccountID: accountID,
            // ...
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
        if err := s.publish(instance, content); err != nil {
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

### 4.3 养号策略引擎

```go
// InteractionEngine 互动引擎 (养号策略)
// 文件: internal/interaction/engine.go
type InteractionEngine struct {
    instanceManager InstanceManager
}

func (e *InteractionEngine) Run(ctx context.Context, task *InteractionTask) error {
    // 获取实例
    instance, err := e.instanceManager.Get(ctx, task.AccountID)
    if err != nil {
        return err
    }

    cdp, _ := e.instanceManager.GetCDPClient(ctx, instance.ID)

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
    // 模拟人类行为: 随机延迟，滚动，点赞
    for i := 0; i < count; i++ {
        cdp.Navigate(ctx, targetURL)
        time.Sleep(randomDelay(2*time.Second, 5*time.Second))

        // 滚动到视频位置
        cdp.Evaluate(ctx, "window.scrollTo(0, 300)")

        // 点赞
        cdp.Click(ctx, ".like-button")
        time.Sleep(randomDelay(1*time.Second, 3*time.Second))
    }
    return nil
}
```

---

## 5. MCP 服务端实现

```go
// MCPServer MCP 服务端
// 文件: internal/mcp/server.go
type MCPServer struct {
    tools []MCPToolDefinition
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

// 工具处理器映射
func (s *MCPServer) registerTools() {
    s.handlers["create_post"] = s.handleCreatePost
    s.handlers["generate_content"] = s.handleGenerateContent
    s.handlers["batch_publish"] = s.handleBatchPublish
    s.handlers["interact"] = s.handleInteract
    s.handlers["get_account_stats"] = s.handleGetStats
}
```

---

## 6. 实现文件

| 文件路径 | 职责 |
|---------|------|
| `internal/mcp/server.go` | MCP 服务端主逻辑 |
| `internal/mcp/tools.go` | MCP 工具定义 |
| `internal/content/generator.go` | AI 内容生成器 |
| `internal/content/store.go` | 内容存储 |
| `internal/publish/scheduler.go` | 发布调度器 |
| `internal/interaction/engine.go` | 养号策略引擎 |
| `internal/ai/client.go` | AI 客户端封装 |
| `internal/api/` | REST API Handler |

---

## 7. 覆盖映射

| PRD类型 | PRD编号 | 架构元素 | 覆盖状态 |
|---------|---------|---------|---------|
| 功能需求 | FR-007 | AI 内容生成 | ✅ |
| 功能需求 | FR-008 | 批量发布管理 | ✅ |
| 功能需求 | FR-009 | 账号养号自动化 | ✅ |
| 功能需求 | FR-010 | MCP 服务端 | ✅ |
| 功能需求 | FR-011 | REST API | ✅ |