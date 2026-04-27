# 开发计划总览

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | TikTok Matrix Operations System (TMOS) |
| **版本** | v1.0 |
| **对应架构** | docs/v1/02-architecture/ |
| **创建日期** | 2026-04-22 |

---

## 1. 项目概述

TikTok Matrix Operations System (TMOS) 是一个**指纹浏览器矩阵管理工具**，使用 Go 语言开发，包含 6 个核心模块，实现 200+ 账号同时在线管理与自动化运营。

### 1.1 核心功能

| 模块 | 功能 | 技术栈 |
|------|------|--------|
| M1 指纹引擎 | 唯一指纹生成 + 一致性校验 | Go |
| M2 实例管理 | 200+ 浏览器实例生命周期管理 | Go + CDP |
| M3 IP 池 | 住宅代理 IP 分配与绑定 | Go + 代理商 API |
| M4 验证码 | 接码 + 固定号管理 | Go + 多平台适配器 |
| M5 账号注册 | TikTok 注册 + 换绑 | Go + TikTokAutomation |
| M6 AI 自动化 | MCP 服务端 + REST API | Go + MCP Protocol |

### 1.2 技术架构

```
AI Agent ──MCP──► MOD-06 AI 自动化
                          │
                          ▼
                    MOD-05 账号注册
                          │
              ┌───────────┼───────────┐
              ▼           ▼           ▼
         MOD-04       MOD-02       MOD-03
         验证码       实例管理       IP 池
              │           │           │
              └───────────┴───────────┘
                          │
                          ▼
                    MOD-01 指纹引擎
                          │
              CloakBrowser 二进制
```

---

## 2. 模块开发计划

| 批次 | 模块 | 开发计划 | 任务数 | 预估工时 | 状态 |
|------|------|---------|-------|---------|------|
| 1 | MOD-01 指纹引擎 | 01-backend-mod-01.md | 8 | 2天 | ✅ 已完成 |
| 1 | MOD-04 验证码 | 02-backend-mod-04.md | 8 | 2天 | ✅ 已完成 |
| 2 | MOD-02 实例管理 | 03-backend-mod-02.md | 10 | 3天 | ✅ 已完成 |
| 2 | MOD-03 IP 池 | 04-backend-mod-03.md | 8 | 2天 | ✅ 已完成 |
| 3 | MOD-05 账号注册 | 05-backend-mod-05.md | 8 | 2天 | ✅ 已完成 |
| 4 | MOD-06 AI 自动化 | 06-backend-mod-06.md | 12 | 4天 | ✅ 已完成 |
| 5 | 集成测试 | 07-integration.md | 10 | 3天 | ✅ 已完成 |
| 6 | Docker 部署 | 08-deployment.md | 6 | 2天 | ✅ 已完成 |

**外部依赖集成任务（补充）**:
| 批次 | 模块 | 开发计划 | 任务数 | 预估工时 | 状态 |
|------|------|---------|-------|---------|------|
| - | CloakBrowser 集成 | 09-backend-cloakbrowser.md | 4 | 1天 | 待开发 |
| - | 代理商 API 对接 | 10-backend-proxy-api.md | 3 | 0.5天 | 待开发 |

**总计**: 77 个开发任务，预估 21.5 天

---

## 3. 开发顺序

### 第 0 批（外部依赖，独立开发）

0. **CloakBrowser 集成** - 外部依赖二进制集成
   - 优先级: P0（MOD-01、MOD-02 都依赖）
1. **代理商 API 对接** - 外部依赖 API 集成
   - 优先级: P0（MOD-03 依赖）

### 第 1 批（无依赖，独立开发）

1. **MOD-01 指纹引擎** - 指纹生成与校验
2. **MOD-04 验证码** - 号码管理与验证码接收

### 第 2 批（依赖第 1 批）

3. **MOD-02 实例管理** - 依赖 M1 的指纹生成
4. **MOD-03 IP 池** - 依赖 M2 实例管理

### 第 3 批（依赖第 1、2 批）

5. **MOD-05 账号注册** - 依赖 M1、M3、M4

### 第 4 批（依赖前几批）

6. **MOD-06 AI 自动化** - 依赖 M5

### 第 5 批

7. **集成测试** - 全模块集成

### 第 6 批

8. **Docker 部署** - 容器化部署

---

## 4. 覆盖映射

| 架构模块 | 开发计划 | 覆盖状态 |
|---------|---------|---------|
| MOD-01 | ✅ 01-backend-mod-01.md | ✅ 已完成 |
| MOD-02 | ✅ 03-backend-mod-02.md | ✅ 已完成 |
| MOD-03 | ✅ 04-backend-mod-03.md | ✅ 已完成 |
| MOD-04 | ✅ 02-backend-mod-04.md | ✅ 已完成 |
| MOD-05 | ✅ 05-backend-mod-05.md | ✅ 已完成 |
| MOD-06 | ✅ 06-backend-mod-06.md | ✅ 已完成 |
| 集成测试 | ✅ 07-integration.md | ✅ 已完成 |
| Docker 部署 | ✅ 08-deployment.md | ✅ 已完成 |
| **外部依赖** | 开发计划 | 覆盖状态 |
| CloakBrowser 二进制 | ✅ 09-backend-cloakbrowser.md | ✅ 已补充 |
| 代理商 API | ✅ 10-backend-proxy-api.md | ✅ 已补充 |

---

## 5. 技术栈

| 类型 | 技术 | 版本 |
|------|------|------|
| **语言** | Go | 1.21+ |
| **框架** | 标准库 + 轻量框架 | - |
| **数据库** | PostgreSQL | 15+ |
| **缓存** | Redis | 7+ |
| **容器** | Docker Compose | 24+ |
| **浏览器引擎** | CloakBrowser | 最新稳定版 |
| **协议** | MCP v1.0, REST | - |

---

## 6. 项目结构

```
tiktok-matrix/
├── cmd/
│   ├── api/              # REST API 服务
│   │   └── main.go
│   ├── mcp/              # MCP 服务端
│   │   └── main.go
│   └── worker/            # 后台任务 Worker
│       └── main.go
├── internal/
│   ├── fingerprint/      # MOD-01 指纹引擎
│   ├── instance/         # MOD-02 实例管理
│   ├── proxy/           # MOD-03 IP 池
│   ├── phone/           # MOD-04 验证码
│   ├── account/         # MOD-05 账号注册
│   ├── content/         # MOD-06 内容管理
│   ├── publish/         # MOD-06 发布调度
│   ├── interaction/     # MOD-06 互动引擎
│   └── mcp/            # MOD-06 MCP 服务端
├── pkg/
│   ├── models/          # 数据模型
│   ├── store/           # 数据访问层
│   └── utils/           # 工具函数
├── migrations/          # 数据库迁移
├── config/             # 配置文件
├── docker-compose.yml   # Docker Compose 配置
├── Dockerfile           # Dockerfile
└── go.mod              # Go 模块
```

---

## 7. 开发规范

### 7.1 代码规范

- 遵循 Go 官方代码规范 (gofmt)
- 命名规范：驼峰命名 (camelCase)
- 错误处理：始终检查 error返回值
- 日志：使用结构化日志 (slog)

### 7.2 提交规范

```
<type>(<scope>): <subject>

Types:
- feat: 新功能
- fix: 修复bug
- docs: 文档变更
- refactor: 重构
- test: 测试
- chore: 构建/工具
```

### 7.3 测试规范

| 测试类型 | 覆盖率要求 | 工具 |
|---------|-----------|------|
| 单元测试 | ≥ 80% | Go testing |
| 集成测试 | ≥ 70% | go test + Docker |

---

## 8. 里程碑

| 里程碑 | 日期 | 交付内容 |
|---------|------|---------|
| M1 | 第 2 周 | MOD-01 + MOD-04 完成 |
| M2 | 第 4 周 | MOD-02 + MOD-03 完成 |
| M3 | 第 6 周 | MOD-05 完成 |
| M4 | 第 8 周 | MOD-06 完成 |
| M5 | 第 10 周 | 集成测试完成 |
| M6 | 第 12 周 | Docker 部署完成 |

---

## 9. 风险与依赖

| 风险/依赖 | 影响 | 缓解措施 |
|----------|------|---------|
| CloakBrowser 二进制兼容性 | 高 | 预留 2 周缓冲测试 |
| 接码平台 API 变更 | 中 | 抽象适配器层 |
| TikTok 反爬升级 | 高 | 建立监控告警机制 |
| 住宅 IP 质量 | 中 | 多代理商备份 |