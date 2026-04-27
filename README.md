# fingerbrower

## 业务定位

**一句话描述**：一个围绕浏览器指纹、代理池、浏览器实例和 CloakBrowser 集成构建的浏览器环境编排仓库。  
**服务对象**：上层账号/会话编排系统、浏览器实例调度逻辑、开发与测试人员。  
**所属模块**：浏览器环境编排与自动化执行。

### 核心功能

- **指纹生成与校验**：`fingerprint/` 生成国家维度的确定性浏览器指纹，并校验平台、GPU、时区、语言、屏幕与网络一致性。
- **代理池管理**：`proxy/` 维护代理状态、绑定关系、健康检查，并支持 Bright Data、Oxylabs、Mock 适配器。
- **浏览器实例管理**：`instance/` 负责端口分配、进程拉起、实例持久化、CDP 连接与生命周期管理。
- **CloakBrowser 集成**：`cloakbrowser/` 封装外部浏览器二进制的启动、参数拼装、健康检查与 CDP 目标发现。

## 项目概览

| 维度 | 结果 |
|------|------|
| 主要语言 | Go |
| 主要运行形态 | Go 库包 |
| 外部依赖 | CloakBrowser 二进制、代理商服务、PostgreSQL 风格存储 |
| 当前架构 | 模块化单仓库，偏库层/适配层设计 |
| 当前范围 | TMOS 参考架构中的 M1 指纹、M2 实例、M3 代理、CloakBrowser 集成子集 |

## 目录说明

| 路径 | 说明 |
|------|------|
| `fingerprint/` | 指纹数据结构、国家配置、生成器、校验器 |
| `proxy/` | 代理实体、Provider 接口、适配器、健康检查、存储 |
| `instance/` | 浏览器实例模型、进程管理、CDP 通信、端口分配、存储 |
| `cloakbrowser/` | CloakBrowser 二进制客户端封装 |
| `docs/04-execution/` | 执行记录与证据文件 |
| `docs/` | 项目分析文档与上游参考资料 |
| `resources/` | 补充资源文件 |
| `screenshots/` | 历史截图与快照输出 |

## 已确认的实现边界

- 仓库当前**没有 HTTP Controller/REST API**，对外接口主要是 Go 包导出方法。
- 仓库当前**没有 `go.mod`**，因此不能直接以标准 Go module 方式运行 `go test ./...`。
- `instance/store.go` 显示存在 `browser_instances` 表，但仓库内**没有完整 schema 文件**。
- Go 代码 import path 指向 `github.com/tmos/facebook/internal/...`。<!-- TODO: 待确认当前仓库与上游模块路径的对应关系 -->
- 仓库当前仅覆盖上游 TMOS 参考架构中的 **FR-001 / FR-002 / FR-003** 对应模块，不包含账号、验证码、AI、MCP、REST API、Docker 部署等完整 PRD 范围。

## 文档索引

- [架构说明](./docs/architecture.md)
- [分层说明](./docs/layers.md)
- [接口说明](./docs/api.md)
- [数据库说明](./docs/database.md)
- [模块说明：fingerprint](./docs/modules-fingerprint.md)
- [模块说明：proxy](./docs/modules-proxy.md)
- [模块说明：instance](./docs/modules-instance.md)
- [模块说明：cloakbrowser](./docs/modules-cloakbrowser.md)
- [后端开发计划：CloakBrowser 集成](./docs/09-backend-cloakbrowser.md)
- [后端开发计划：代理商 API 对接](./docs/10-backend-proxy-api.md)
- [执行记录与证据](./docs/04-execution/report.md)
- [上游参考架构资料](./docs/reference/fingerbrower-v1/architecture/README.md)
