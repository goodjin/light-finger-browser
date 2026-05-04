# fingerbrower

## 业务定位

**一句话描述**：一个围绕浏览器指纹、代理池、浏览器实例和自建 stealth 浏览器运行时构建的浏览器环境编排仓库。
**服务对象**：上层账号/会话编排系统、浏览器实例调度逻辑、开发与测试人员。  
**所属模块**：浏览器环境编排与自动化执行。

### 核心功能

- **指纹生成与校验**：`fingerprint/` 生成国家维度的确定性浏览器指纹，并校验平台、GPU、时区、语言、屏幕与网络一致性。
- **代理池管理**：`proxy/` 维护代理状态、绑定关系、健康检查，并支持 Bright Data、Oxylabs、Mock 适配器。
- **浏览器实例管理**：`instance/` 负责端口分配、进程拉起、实例持久化、CDP 连接与生命周期管理。
- **浏览器运行时接入**：`cloakbrowser/` 目前封装兼容 CloakBrowser 的浏览器启动与 CDP 接入；目标是逐步切换到自建 Chromium-based stealth 浏览器产物。

## 项目概览

| 维度 | 结果 |
|------|------|
| 主要语言 | Go |
| 主要运行形态 | Go 库包 |
| 外部依赖 | 浏览器产物、代理商服务、PostgreSQL 风格存储 |
| 当前架构 | 模块化单仓库，偏库层/适配层设计 |
| 当前范围 | 指纹、实例、代理、浏览器运行时发现，以及浏览器内核专项规划 |

## 当前运行时约定

- **默认浏览器引擎**：实例服务当前默认走 **CloakBrowser-compatible runtime**。
- **开发回退**：仅在显式设置 `BROWSER_ENGINE=local-chrome` 时回退到本地 Chrome。
- **二进制解析顺序**：优先读取 `CLOAKBROWSER_PATH` / `BROWSER_BINARY`，否则再尝试 `resources/cloakbrowser/artifacts.json` 和打包后 app-relative 的浏览器产物路径。
- **诊断入口**：设置页可查看当前激活引擎，以及每个指纹字段在运行时的注入覆盖状态。

## 目录说明

| 路径 | 说明 |
|------|------|
| `fingerprint/` | 指纹数据结构、国家配置、生成器、校验器 |
| `proxy/` | 代理实体、Provider 接口、适配器、健康检查、存储 |
| `instance/` | 浏览器实例模型、进程管理、CDP 通信、端口分配、存储 |
| `cloakbrowser/` | 浏览器运行时客户端封装（当前兼容 CloakBrowser，目标切换自建产物） |
| `resources/cloakbrowser/` | 浏览器产物清单与后续打包入口 |
| `docs/` | 项目文档与测试指南 |
| `resources/` | 补充资源文件 |
| `screenshots/` | 历史截图与快照输出 |

## 已确认的实现边界

- 仓库当前**没有完整业务域 HTTP Controller/REST API 实现**，对外能力仍以 Go 包导出方法与 Wails 命令为主。
- 仓库当前**已经有 `go.mod`**，可直接运行 `go test ./...` 与 `go build ./...`。
- `instance/store.go` 显示存在 `browser_instances` 表，但仓库内**没有完整 schema 文件**。
- 主模块 import path 已切换到 `github.com/tmos/fingerbrower`，但 `proxy/adapter/` 中仍保留少量上游 `github.com/tmos/facebook/internal/proxy` 引用，需要后续统一。
- 仓库当前活动文档集只保留与**指纹浏览器产品本身**直接相关的内容；旧的 TMOS/TikTok 业务文档已移出活动文档集。

## 文档索引

- [PRD](./docs/v1/01-prd.md)
- [架构覆盖报告](./docs/v1/02-architecture/00-coverage-report.md)
- [模块说明：MOD-01 指纹引擎](./docs/v1/02-architecture/03-mod-01-fingerprint.md)
- [模块说明：MOD-02 实例管理](./docs/v1/02-architecture/03-mod-02-instance.md)
- [模块说明：MOD-03 IP 池](./docs/v1/02-architecture/03-mod-03-proxy.md)
- [浏览器源码构建与运行时发现架构](./docs/v1/02-architecture/01-browser-build-runtime.md)
- [浏览器内核专项技术设计](./docs/v1/02-architecture/02-browser-core-technical-design.md)
- [开发计划总览](./docs/v1/03-dev-plan/00-overview.md)
- [后端开发计划：MOD-01 指纹引擎](./docs/v1/03-dev-plan/01-backend-mod-01.md)
- [后端开发计划：MOD-02 实例管理](./docs/v1/03-dev-plan/03-backend-mod-02.md)
- [后端开发计划：MOD-03 IP 池](./docs/v1/03-dev-plan/04-backend-mod-03.md)
- [浏览器源码构建改进方案](./docs/v1/03-dev-plan/09-browser-source-build.md)
- [浏览器内核专项开发计划](./docs/v1/03-dev-plan/10-browser-core-program.md)
- [单元测试指南](./docs/unit-test.md)
- [集成测试指南](./docs/integration-test.md)
- [E2E 测试指南](./docs/e2e-test.md)
- [Mock 策略](./docs/testing/mock-strategies.md)
