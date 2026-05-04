# 浏览器内核专项开发计划

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | Light Finger Browser |
| **计划主题** | Chromium-based stealth 浏览器专项计划 |
| **版本** | v1.1 |
| **对应 PRD** | docs/v1/01-prd.md v2.0 |
| **对应设计** | docs/v1/02-architecture/02-browser-core-technical-design.md |
| **更新日期** | 2026-04-30 |

---

## 1. 计划目标

把浏览器核心工作从“接入外部 CloakBrowser”推进到“**自建 Chromium-based stealth 浏览器产品**”，并与当前应用层形成稳定分工：

1. 浏览器核心：源码 fork、patch、构建、发布
2. 当前仓库：实例管理、指纹生成、代理绑定、诊断验证、业务编排

---

## 2. 工作流总览

```text
PRD v2.0
   ↓
浏览器专项技术设计
   ↓
P1 基线构建
   ↓
P2 patch 框架
   ↓
P3 运行时接入
   ↓
P4 验证基线
   ↓
P5 发布流水线
   ↓
P6 超越 CloakBrowser 的持续迭代
```

---

## 3. 工作分期

### Phase 1：Chromium 基线与最小产物

**目标**

- 建立 Chromium 源码工作区
- 产出首个可启动的最小浏览器产物
- 输出基础 manifest 与版本元数据

**任务**

1. 建立 browser-core 独立工作区
2. 配置 `depot_tools`、`fetch`、`gclient`
3. 选择首个 Chromium stable milestone
4. 打通单平台最小编译
5. 生成基础产物目录与 manifest

**交付物**

- 可编译工作区
- 单平台浏览器产物
- 首版 manifest
- build bootstrap 文档

**验收**

- 能编译
- 能启动
- 能暴露 CDP
- 能被当前运行时发现

---

### Phase 2：Patch Framework 与 Identity Patch

**目标**

- 建立 patch overlay 组织方式
- 先打通 identity 一致性能力

**任务**

1. 定义 patch 分类目录
2. 建立 patch 开关与 build 标识
3. 实现 identity patch：
   - platform
   - locale
   - timezone
   - screen
   - hardware basics
4. 建立 patch 对应验证脚本

**交付物**

- patch overlay 目录结构
- identity patch 集合
- patch 验证脚本

**验收**

- expected vs actual 差异显著下降
- identity patch 可独立开关

---

### Phase 3：Rendering / Automation Patch

**目标**

- 打通对 anti-detect 最关键的核心能力

**任务**

1. Rendering patch：
   - Canvas
   - WebGL
   - Audio
   - Fonts / client rect
2. Automation patch：
   - webdriver
   - automation flags
   - CDP 痕迹
   - 输入行为基础特征

**交付物**

- rendering patch 集合
- automation patch 集合
- 检测站验证脚本

**验收**

- 指纹站点一致性满足基线
- 自动化痕迹显著下降

---

### Phase 4：运行时整合与业务灰度

**目标**

- 让当前应用正式切到自建浏览器产物

**任务**

1. 对接 manifest 与版本锁定
2. 支持按环境选择浏览器版本
3. 让实例管理层支持浏览器版本切换
4. 建立灰度通道：
   - alpha
   - beta
   - stable

**交付物**

- runtime integration 完整链路
- 版本切换能力
- 灰度配置能力

**验收**

- 当前应用可使用自建产物创建实例
- 灰度版本可并行存在

---

### Phase 5：构建、签名与发布流水线

**目标**

- 把浏览器核心从“能编出来”推进到“能稳定发版”

**任务**

1. 平台构建脚本
2. CI 构建 job
3. 签名 / notarization
4. checksum 与 manifest 产出
5. 回滚机制

**交付物**

- release pipeline
- signed artifacts
- 回滚手册

**验收**

- 三平台产物可重复构建
- stable 版本可签名、可回滚

---

### Phase 6：超越 CloakBrowser 的持续迭代

**目标**

- 在产品指标上形成超越

**重点方向**

1. 更强指纹一致性
2. 更细粒度 runtime knobs
3. 更好的 AI/自动化兼容性
4. 更快发布与回滚
5. 更强诊断与验证闭环

---

## 4. 依赖关系

| 阶段 | 依赖 |
|------|------|
| Phase 1 | 无 |
| Phase 2 | Phase 1 |
| Phase 3 | Phase 2 |
| Phase 4 | Phase 1, 2, 3 |
| Phase 5 | Phase 4 |
| Phase 6 | Phase 5 |

---

## 5. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| Chromium 构建成本高 | 构建慢、机器要求高 | 使用独立 build 环境与缓存 |
| patch 侵入过大 | 升级困难 | patch 分类、可开关、可验证 |
| 单平台先行导致误判 | 跨平台补丁不一致 | Phase 1 起就保留统一 artifact contract |
| 检测站结果波动 | 误判效果 | 建立多维验证，不只依赖单站 |
| 当前仓库与浏览器核心耦合过深 | 升级困难 | 严格通过 manifest/runtime contract 集成 |

---

## 6. 交付门禁

每个阶段完成时必须回答：

1. **产物是否可复现？**
2. **行为是否可验证？**
3. **退化是否可比较？**
4. **失败是否可回滚？**

只有四项都满足，阶段才算真正完成。

---

## 7. 当前优先级

### Ready Now

| 任务 | 目标 | 进入条件 |
|------|------|---------|
| `BROW-001` | 输出 browser-core 仓库/工作区规范 | Ready |
| `BROW-002` | 输出构建机准入清单 | Ready |
| `BROW-006` | 输出浏览器产物目录结构、频道与命名规范 | Ready |
| `BROW-014` | 输出检测站、采样字段与发布门禁规范 | Ready |

> 上述 4 项与当前 SQL ready 集一致，适合作为下一轮子代理并行执行入口。

### Next After Ready Set

1. `BROW-004 chromium-baseline-pin`：在 `BROW-001` 完成后固定首个 Chromium revision
2. `BROW-003 depot-tools-bootstrap`：在 `BROW-001`、`BROW-002` 完成后固化工具链
3. `BROW-005 source-fetch-bootstrap`：在 baseline 与工具链到位后编写拉取脚本
4. `BROW-007 minimal-build-args`：在 baseline 固定后定义最小构建参数

### Not Ready Yet

1. `BROW-008` 及后续真实 Chromium 编译任务
2. `BROW-017` 及后续 patch framework / identity / rendering / automation 任务
3. `BROW-029` ~ `BROW-032` 的应用整合与发布流水线任务

这些项取决于 ready set 交付物、可用磁盘、独立构建环境与构建机准备情况。

---

## 8. 结论

这份计划的目的，不是把浏览器工作拆成若干“代码任务”，而是把它变成一个**可持续迭代的浏览器内核工程项目**。  
后续所有开发都应以此计划为准，按阶段推进，而不是再回到“先接个外部浏览器顶上”的短期路径。

---

## 9. 原子任务拆分

以下任务按“足够小、足够独立、适合子代理并行执行”的原则拆分。

### 9.1 Foundation / Build Base

| ID | 任务 | 目标 | 依赖 |
|----|------|------|------|
| BROW-001 | browser-core-workspace-spec | 定义 browser-core 独立工作区、目录约定、版本文件和输出目录规范 | - |
| BROW-002 | build-host-requirements | 固化磁盘/CPU/RAM/OS/工具链要求，形成构建机准入清单 | - |
| BROW-003 | depot-tools-bootstrap | 安装并验证 `depot_tools` 与基础构建工具链 | BROW-001, BROW-002 |
| BROW-004 | chromium-baseline-pin | 选择首个 Chromium stable milestone 并固定 revision | BROW-001 |
| BROW-005 | source-fetch-bootstrap | 编写 `fetch` / `gclient sync` 初始化脚本与说明 | BROW-003, BROW-004 |
| BROW-006 | build-output-layout | 定义浏览器产物目录结构、频道目录、命名规范 | BROW-001 |
| BROW-007 | minimal-build-args | 固化首个平台最小 `gn args` 与编译目标 | BROW-004, BROW-005 |
| BROW-008 | single-platform-build-smoke | 在单平台产出首个可启动浏览器制品 | BROW-007 |

### 9.2 Artifact / Runtime Contract

| ID | 任务 | 目标 | 依赖 |
|----|------|------|------|
| BROW-009 | build-metadata-emitter | 为构建产物输出版本、revision、checksum、时间戳元数据 | BROW-006, BROW-008 |
| BROW-010 | runtime-smoke-runner | 验证浏览器能启动、退出、暴露 CDP、读取 user-data-dir | BROW-008 |
| BROW-011 | artifact-manifest-v2 | 将浏览器产物映射到统一 manifest 契约 | BROW-006, BROW-009 |
| BROW-012 | app-version-selection | 在当前应用层增加浏览器版本选择/锁定能力 | BROW-011 |
| BROW-013 | packaged-bundle-layout | 定义桌面包/服务端如何携带浏览器产物 | BROW-011 |

### 9.3 Validation / Gate Infrastructure

| ID | 任务 | 目标 | 依赖 |
|----|------|------|------|
| BROW-014 | detection-baseline-spec | 固化检测站、业务验证站、采样字段和发布门禁 | BROW-001 |
| BROW-015 | fingerprint-baseline-runner | 产出 expected vs actual 指纹基线校验脚本 | BROW-010, BROW-014 |
| BROW-016 | regression-diff-runner | 对比相邻浏览器版本的关键行为与指纹退化 | BROW-015 |

### 9.4 Patch Framework

| ID | 任务 | 目标 | 依赖 |
|----|------|------|------|
| BROW-017 | patch-overlay-layout | 建立 patch 分类目录、命名规则、开关策略 | BROW-004 |
| BROW-018 | patch-build-switches | 增加 patch train、patch enable/disable 构建开关 | BROW-017, BROW-007 |
| BROW-019 | identity-patch-platform-locale | 实现 platform / locale / timezone patch | BROW-018, BROW-008 |
| BROW-020 | identity-patch-screen-hardware | 实现 screen / hardware basics patch | BROW-018, BROW-008 |
| BROW-021 | identity-validation-run | 校验 identity patch 是否满足一致性目标 | BROW-015, BROW-019, BROW-020 |

### 9.5 Rendering / Automation Core

| ID | 任务 | 目标 | 依赖 |
|----|------|------|------|
| BROW-022 | rendering-patch-canvas | 实现 canvas patch | BROW-018, BROW-008 |
| BROW-023 | rendering-patch-webgl | 实现 WebGL / GPU patch | BROW-018, BROW-008 |
| BROW-024 | rendering-patch-audio-fonts | 实现 audio / fonts / client rect patch | BROW-018, BROW-008 |
| BROW-025 | rendering-validation-run | 校验 rendering patch 基线 | BROW-015, BROW-022, BROW-023, BROW-024 |
| BROW-026 | automation-patch-signals | 实现 webdriver / automation flags / CDP 痕迹 patch | BROW-018, BROW-008 |
| BROW-027 | automation-patch-input | 实现输入事件和基础行为 patch | BROW-018, BROW-008 |
| BROW-028 | automation-validation-run | 校验自动化痕迹与输入行为 patch | BROW-015, BROW-026, BROW-027 |

### 9.6 Integration / Release

| ID | 任务 | 目标 | 依赖 |
|----|------|------|------|
| BROW-029 | app-runtime-cutover-alpha | 让当前应用支持 alpha 通道自建浏览器产物 | BROW-012, BROW-021, BROW-025, BROW-028 |
| BROW-030 | signing-checksum-pipeline | 建立签名、checksum、产物校验流程 | BROW-009, BROW-013 |
| BROW-031 | release-channel-promotion | 建立 alpha / beta / stable 晋级流程 | BROW-016, BROW-029, BROW-030 |
| BROW-032 | rollback-procedure | 固化浏览器产物回滚和版本锁定流程 | BROW-031 |

---

## 10. 推荐执行顺序

### Batch A：先做基础设施

1. BROW-001 browser-core-workspace-spec
2. BROW-002 build-host-requirements
3. BROW-004 chromium-baseline-pin
4. BROW-006 build-output-layout
5. BROW-014 detection-baseline-spec

### Batch B：拉起可编译基线

1. BROW-003 depot-tools-bootstrap
2. BROW-005 source-fetch-bootstrap
3. BROW-007 minimal-build-args
4. BROW-008 single-platform-build-smoke

### Batch C：固定产物契约与运行时冒烟

1. BROW-009 build-metadata-emitter
2. BROW-010 runtime-smoke-runner
3. BROW-011 artifact-manifest-v2
4. BROW-012 app-version-selection
5. BROW-013 packaged-bundle-layout
6. BROW-015 fingerprint-baseline-runner
7. BROW-016 regression-diff-runner

### Batch D：搭 patch 框架

1. BROW-017 patch-overlay-layout
2. BROW-018 patch-build-switches

### Batch E：先 identity，再 rendering / automation

1. BROW-019 identity-patch-platform-locale
2. BROW-020 identity-patch-screen-hardware
3. BROW-021 identity-validation-run
4. BROW-022 rendering-patch-canvas
5. BROW-023 rendering-patch-webgl
6. BROW-024 rendering-patch-audio-fonts
7. BROW-025 rendering-validation-run
8. BROW-026 automation-patch-signals
9. BROW-027 automation-patch-input
10. BROW-028 automation-validation-run

### Batch F：最后做整合和发布

1. BROW-029 app-runtime-cutover-alpha
2. BROW-030 signing-checksum-pipeline
3. BROW-031 release-channel-promotion
4. BROW-032 rollback-procedure

---

## 11. 子代理编排建议

### 可并行的子代理任务组

| 任务组 | 适合并行的任务 |
|--------|----------------|
| Ready 基础组 | BROW-001, BROW-002, BROW-006, BROW-014 |
| Build 接入组 | BROW-003, BROW-004, BROW-005, BROW-007, BROW-008 |
| 契约与验证组 | BROW-009, BROW-010, BROW-011, BROW-012, BROW-013, BROW-015, BROW-016 |
| Patch Framework 组 | BROW-017, BROW-018 |
| Identity 组 | BROW-019, BROW-020, BROW-021 |
| Rendering 组 | BROW-022, BROW-023, BROW-024, BROW-025 |
| Automation 组 | BROW-026, BROW-027, BROW-028 |
| Release 组 | BROW-029, BROW-030, BROW-031, BROW-032 |

### 不建议并行的情况

1. 当前 ready set 完成前，不要开始真实 Chromium 拉取或 revision 固定之外的构建接入任务
2. `BROW-008` 完成前，不要开始任何真实 patch 开发
3. `BROW-018` 完成前，不要让多个 patch 子代理各自发明自己的 patch 组织方式
4. `BROW-015` 完成前，不要把 patch 实现结果当成“可验证完成”
