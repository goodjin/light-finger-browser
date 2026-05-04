# PRD-架构覆盖报告

> 本报告只覆盖当前活动文档集：**指纹、实例、代理、浏览器运行时、浏览器内核计划**。旧 TMOS/TikTok 业务文档已从活动文档集中移除，不再参与覆盖统计。

## 文档信息

| 字段 | 内容 |
|------|------|
| **版本** | v2.1 |
| **更新日期** | 2026-04-30 |
| **对应 PRD** | `docs/v1/01-prd.md` v2.0 |

---

## 状态图例

| 符号 | 含义 | 判定标准 |
|------|------|---------|
| ✅ | 已实现 / 已覆盖 | 已有明确架构承接，且代码或产物已落地 |
| 🟡 | 部分落地 | 已有代码、存储或契约，但尚未形成完整闭环或发布门禁 |
| ⏳ | 已设计 / 已规划 | 已有架构或计划，但尚未进入实现阶段 |

## 覆盖统计

| 类型 | PRD总数 | 已有架构/计划承接 | 待补充 | 覆盖率 |
|-----|--------|------------------|-------|-------|
| 功能需求 | 8 | 8 | 0 | 100% |
| 用户故事 | 3 | 3 | 0 | 100% |
| 数据实体 | 5 | 5 | 0 | 100% |
| 业务规则 | 5 | 5 | 0 | 100% |

---

## 功能需求覆盖

| PRD编号 | 功能名称 | 主承接文档 | 代码/产物 | 状态 |
|---------|---------|-----------|----------|------|
| FR-001 | 多实例浏览器管理 | `03-mod-02-instance.md` | `instance/` | ✅ 已实现 |
| FR-002 | 独立指纹生成与校验 | `03-mod-01-fingerprint.md` | `fingerprint/` | ✅ 已实现 |
| FR-003 | 代理绑定与健康检查 | `03-mod-03-proxy.md` | `proxy/` | ✅ 已实现 |
| FR-004 | 浏览器运行时发现 | `01-browser-build-runtime.md` | `app/commands/browser_runtime.go` | ✅ 已实现 |
| FR-005 | 浏览器产物清单契约 | `01-browser-build-runtime.md` | `resources/cloakbrowser/artifacts.json` | ✅ 已落地基础契约 |
| FR-006 | Chromium 源码基线与 patch overlay | `02-browser-core-technical-design.md` / `10-browser-core-program.md` | - | ⏳ 已设计，待执行 |
| FR-007 | 指纹诊断与验证闭环 | `03-mod-01-fingerprint.md` / `01-browser-build-runtime.md` | `app/commands/fingerprint_diagnostics.go`, `app/commands/fingerprint_coverage.go` | 🟡 已有采样与覆盖报告，但未纳入发布门禁 |
| FR-008 | 构建、签名、发布、回滚流水线 | `09-browser-source-build.md` / `10-browser-core-program.md` | - | ⏳ 已规划，待执行 |

---

## 用户故事覆盖

| PRD编号 | 用户故事 | 主承接文档 | 状态 |
|---------|---------|-----------|------|
| US-001 | 创建一个可复现的隔离浏览器实例 | `03-mod-01-fingerprint.md` + `03-mod-02-instance.md` + `03-mod-03-proxy.md` | ✅ |
| US-002 | 交付一套可被应用层消费的浏览器产物 | `01-browser-build-runtime.md` + `09-browser-source-build.md` | ✅ |
| US-003 | 在发布前验证真实运行时指纹 | `01-browser-build-runtime.md` + `02-browser-core-technical-design.md` | ✅ |

---

## 数据实体覆盖

| PRD编号 | 实体名称 | 主承接文档 | 代码/产物 | 状态 |
|---------|---------|-----------|----------|------|
| Entity-001 | BrowserInstance | `03-mod-02-instance.md` | `instance/types.go` | ✅ |
| Entity-002 | FingerprintProfile | `03-mod-01-fingerprint.md` | `fingerprint/types.go` | ✅ |
| Entity-003 | ProxyEndpoint | `03-mod-03-proxy.md` | `proxy/types.go` | ✅ |
| Entity-004 | BrowserArtifactManifest | `01-browser-build-runtime.md` | `resources/cloakbrowser/artifacts.json` | ✅ |
| Entity-005 | FingerprintSnapshot | `01-browser-build-runtime.md` | `app/commands/fingerprint_diagnostics.go`, `storage/sqlite/fingerprint_snapshot.go` | 🟡 已有采样与持久化，但未形成完整验证闭环 |

---

## 业务规则覆盖

| PRD编号 | 规则名称 | 主承接文档 | 状态 |
|---------|---------|-----------|------|
| Rule-001 | 一实例一代理 | `03-mod-03-proxy.md` | ✅ |
| Rule-002 | 指纹一致性 | `03-mod-01-fingerprint.md` | ✅ |
| Rule-003 | 浏览器版本锁定 | `01-browser-build-runtime.md` | ✅ |
| Rule-004 | 运行时发现优先级 | `01-browser-build-runtime.md` | ✅ |
| Rule-005 | 发布门禁 | `02-browser-core-technical-design.md` | ✅ |

---

## 当前未落地项

1. 自维护 Chromium 源码 fork
2. stealth / anti-detect patch overlay
3. 三平台构建、签名、发布、回滚流水线
4. 诊断与验证能力的发布门禁化

---

## 活动模块映射

| 文档 | 角色 | 当前状态 |
|------|------|---------|
| `03-mod-01-fingerprint.md` | 当前仓库已实现模块 | ✅ |
| `03-mod-02-instance.md` | 当前仓库已实现模块 | ✅ |
| `03-mod-03-proxy.md` | 当前仓库已实现模块 | ✅ |
| `01-browser-build-runtime.md` | 当前仓库运行时架构 | ✅ |
| `02-browser-core-technical-design.md` | 浏览器内核目标设计 | ✅ |
| `09-browser-source-build.md` | 浏览器专项改进方案 | ✅ |
| `10-browser-core-program.md` | 浏览器内核执行计划 | ✅ |

---

## 文档编号规则

`02-architecture/` 目录当前采用**分类编号**而非连续流水号：

1. `00-*`：索引 / 覆盖类文档
2. `01-*`：浏览器运行时架构
3. `02-*`：浏览器内核技术设计
4. `03-*`：当前活动模块架构文档

> 旧业务模块文档移除后，不再为已删除类别回填新编号。

---

## 变更历史

| 版本 | 日期 | 变更内容 | 作者 |
|-----|------|---------|------|
| 2.1 | 2026-04-30 | 补充文档信息、状态图例与架构编号规则，明确 🟡 的定义 | Claude |
| 2.0 | 2026-04-30 | 收缩覆盖范围到当前浏览器产品文档集，移除 TMOS/TikTok 业务模块映射 | Claude |
| 1.3 | 2026-04-30 | 将浏览器扩展覆盖口径调整为 Chromium 基线 + 自研 stealth 浏览器方案，不再假设可获取 CloakBrowser 完整源码 | Claude |
| 1.0 | 2026-04-21 | 初始版本 | Claude |
