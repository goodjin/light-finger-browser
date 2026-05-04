# 浏览器源码构建与运行时发现架构

## 文档信息

| 字段 | 内容 |
|------|------|
| **对应 PRD** | v2.0 自建 stealth 浏览器目标 |
| **版本** | v2.0 |
| **状态** | 生效中 |
| **更新日期** | 2026-04-30 |

---

## 1. 目标

PRD v2.0 已将浏览器侧目标调整为：

1. 以 Chromium 源码为基线，自主构建全平台 stealth 浏览器产物。
2. 完整掌控 Chromium 内核、反检测补丁、指纹注入能力与版本节奏。
3. 桌面端与服务端运行时都不再只依赖人工配置一个外部二进制路径。
4. 产品目标不是复用 CloakBrowser 私有补丁链，而是做出一款在可控性和效果上超过 CloakBrowser 的产品。

本架构文档约束浏览器侧必须拆成 **源码层、补丁层、构建层、产物层、运行时发现层、发布层** 六层，而不是继续停留在“给 Go 进程一个浏览器路径”的集成方式。

---

## 2. 当前问题

当前仓库已经具备 CloakBrowser 运行时管理能力，但仍存在以下结构缺口：

| 问题 | 当前表现 | 风险 |
|------|----------|------|
| 浏览器产物来源不受控 | 运行时主要依赖 `CLOAKBROWSER_PATH` / `BROWSER_BINARY` | 仍然依赖外部预编译包，无法完整掌控供应链 |
| 缺少产物清单 | 没有统一描述不同平台浏览器产物位置与版本 | 桌面包、容器、测试环境容易分叉 |
| 缺少 app 相对路径发现 | 打包后的 Wails 应用无法稳定发现内嵌浏览器 | 发布后仍需人工设环境变量 |
| 缺少源码构建链设计 | 还没有定义 Chromium 基线、补丁叠加、构建、签名、发布链路 | PRD v2.0 无法落地 |
| 过度绑定 CloakBrowser 命名 | 运行时与文档仍沿用 CloakBrowser 集成口径 | 容易把“竞品参考”误当成“产品基线” |

---

## 3. 目标架构

```text
Chromium Source Fork
              │
              ▼
       Stealth Patch Overlay
              │
              ▼
      Cross-Platform Build Orchestrator
              │
              ▼
      Artifact Output + Manifest Index
              │
      ┌───────┴────────┐
      ▼                ▼
Desktop Bundle     Server/CI Runtime
      │                │
      └───────┬────────┘
              ▼
     Browser Runtime Resolver
              │
              ▼
      BrowserRuntimeManager / InstanceService
```

### 3.1 分层职责

| 层 | 责任 | 本仓库当前状态 |
|----|------|----------------|
| Source Fork | 维护 Chromium 源码基线 | 未落地 |
| Patch Overlay | 维护 stealth、反检测与指纹补丁 | 未落地 |
| Build Orchestrator | 编译 macOS / Windows / Linux 产物 | 未落地 |
| Artifact Manifest | 声明平台、架构、相对路径、版本 | **本次新增设计并开始落地** |
| Runtime Resolver | 解析 env、产物清单、应用内嵌路径 | **本次执行落地** |
| Runtime Manager | 启动、停止、重启、监控实例 | 已落地 |

---

## 4. 运行时发现顺序

浏览器运行时路径解析必须遵循以下优先级：

1. **显式环境变量覆盖**
   - `CLOAKBROWSER_PATH`
   - `BROWSER_BINARY`
2. **浏览器产物清单**
   - `resources/cloakbrowser/artifacts.json`
   - 打包后的 app-relative manifest
3. **应用内嵌默认路径**
   - macOS: `Contents/Resources/.../Chromium.app/...`
   - Windows: `cloakbrowser/*.exe`
   - Linux: `cloakbrowser/*`

这样做的原因：

- 开发环境仍然保留手工覆盖能力。
- 打包环境可以按平台自动解析浏览器产物。
- 后续把“源码构建链”接进来时，只需产出产物并刷新 manifest，不必再次改实例启动逻辑。

> 说明：`CLOAKBROWSER_PATH` 与 `resources/cloakbrowser/` 中的命名属于**兼容期遗留命名**。它们当前表示“现有运行时兼容入口”，并不代表产品基线仍是 CloakBrowser。

---

## 5. 产物清单契约

运行时发现层依赖一个平台无关的浏览器产物清单。

### 5.1 文件位置

- 仓库内开发/打包基线：`resources/cloakbrowser/artifacts.json`
- 打包后推荐位置：
  - macOS: `YourApp.app/Contents/Resources/cloakbrowser/artifacts.json`
  - Windows/Linux: `<app-dir>/resources/cloakbrowser/artifacts.json` 或 `<app-dir>/cloakbrowser/artifacts.json`

### 5.2 数据结构

```json
{
  "version": 1,
  "artifacts": [
    {
      "os": "darwin",
      "arch": "arm64",
      "path": "darwin-arm64/Chromium.app/Contents/MacOS/Chromium"
    }
  ]
}
```

### 5.3 约束

1. `path` 优先使用相对 manifest 的相对路径。
2. 同一个 `(os, arch)` 只能解析到一个发布用浏览器入口。
3. 产物路径必须是最终可执行入口，而不是目录。
4. manifest 只描述“已准备好的浏览器产物”，不承担下载逻辑。

> 说明：目录名继续沿用 `cloakbrowser/` 是为了兼容当前运行时路径与既有资源布局；后续若统一重命名，应作为独立迁移任务处理。

### 5.4 BROW-011 artifact manifest v2

manifest v2 在保持 `artifacts.json` 使用方式不变的前提下，补齐版本、通道与校验信息，并与 `metadata/` 输出对齐（BROW-009）。每个通道的 `artifacts.json` 位于 `channels/<channel>/`，结构如下：

```json
{
  "version": 2,
  "channel": "stable",
  "artifacts": [
    {
      "os": "darwin",
      "arch": "arm64",
      "path": "darwin-arm64/Chromium.app/Contents/MacOS/Chromium",
      "version": "124.0.0+rev.12345",
      "checksums": {
        "sha256": "<sha256>",
        "sha512": "<sha512>"
      }
    }
  ],
  "metadata": {
    "build": "metadata/build.json",
    "checksums": "metadata/checksums.json"
  }
}
```

**Schema 版本**

- `version=2` 表示 manifest v2；`version=1` 或缺失时视为 v1。
- v2 新增 `channel`、`version`、`checksums`、`metadata` 字段，不改变现有 `artifacts` 解析逻辑。

**字段说明**

- `channel`：当前 manifest 所属通道（`alpha`/`beta`/`stable`），需与目录名一致。
- `artifacts[].os` / `artifacts[].arch`：平台与架构标识（与 BROW-006 目录名一致）。
- `artifacts[].path`：相对 `channels/<channel>/` 的可执行入口路径。
- `artifacts[].version`：浏览器版本或 revision 标识，应与 `metadata/build.json` 保持一致。
- `artifacts[].checksums`：至少包含 `sha256`；若提供 `sha512` 用于增强校验。
- `metadata.build` / `metadata.checksums`：元数据文件相对路径，便于运行时与发布侧校验。

**选择规则**

1. 通道选择遵循 §6.4：未指定时默认 `stable`。
2. 运行时只读取所选通道目录下的 `artifacts.json`，按 `os+arch` 精确匹配。

**校验与约束**

- 每个通道内 `(os, arch)` 必须唯一。
- `path` 必须指向最终可执行入口。
- `version` 必须与 `metadata/build.json` 对齐（相同版本/revision）。
- 若 `checksums` 缺失，应以 `metadata/checksums.json` 为准；若两者同时存在，必须一致。

**兼容说明（v1）**

- v1 仅含 `version=1` 与 `artifacts[].os/arch/path`；默认通道视为 `stable`。
- v1 不要求 `version`/`checksums` 字段，运行时应保持向后兼容读取。

---

## 6. 构建产物输出布局（BROW-006）

构建产物输出目录必须与运行时发现与 manifest 契约保持一致，默认仍使用 `cloakbrowser/` 作为根目录（兼容期命名，不在本阶段更名）。

### 6.1 顶层布局（按 OS/Arch）

```text
cloakbrowser/
  channels/
    alpha/
      darwin-arm64/
      darwin-x64/
      linux-x64/
      windows-x64/
    beta/
      ...
    stable/
      ...
```

- `os-arch` 目录名必须与 manifest 中的 `os`/`arch` 组合一致，例如 `darwin-arm64`、`windows-x64`。
- `os-arch` 目录内放置**最终可执行入口**（或其所在目录），manifest 的 `path` 从 channel 根目录开始相对引用。

### 6.2 Channel 目录与命名

- 统一三条发布通道：`alpha` / `beta` / `stable`。
- 每个通道都是一份完整产物集合，可并行存在，用于灰度与回滚。
- 运行时默认指向 `stable`；具体通道由应用配置或环境选择（实现不在本阶段展开）。

### 6.3 Manifest 与元数据位置

```text
cloakbrowser/
  channels/
    stable/
      artifacts.json
      metadata/
        build.json
        checksums.json
      darwin-arm64/...
```

- `artifacts.json` 位于 `channels/<channel>/` 根目录，描述该通道的跨平台入口。
- `metadata/` 存放构建版本、revision、时间戳与校验信息，供发布/诊断与回滚使用。
- 打包后仍需把选定通道的 `artifacts.json` 与必要元数据**映射到** `resources/cloakbrowser/` 下，以保证运行时只需按统一路径读取。

### 6.4 Runtime Resolver 选择顺序

运行时解析保持现有优先级，仅在“manifest”步骤中引入通道概念：

1. 显式环境变量覆盖（`CLOAKBROWSER_PATH` / `BROWSER_BINARY`）
2. 选定通道的 `artifacts.json`（若未指定通道，则默认 `stable`）
3. 应用内嵌默认路径

---

## BROW-012 app version selection

本节用于固化 **Rule-003 浏览器版本锁定** 的 app 侧选择与锁定策略，确保在运行时切换前已有可审计的版本决策流程，并与 manifest v2 的 `channel`/`version` 字段对齐。

### 选择来源（按优先级）

1. **环境变量覆盖**：用于紧急回滚/灰度测试的临时选择入口（例如 `BROWSER_CHANNEL` / `BROWSER_VERSION`）。仅在未指定浏览器路径覆盖时生效。
2. **应用/实例配置**：应用级默认通道 + 实例级版本选择（实例创建时明确写入）。
3. **manifest v2**：根据选定通道读取 `channels/<channel>/artifacts.json`，以 `channel` 与 `artifacts[].version` 作为最终版本来源；若为 v1，则默认 `stable` 通道且版本未知。

### 版本锁定规则

- 实例创建时必须解析出 **唯一的 channel + version**，并写入实例记录；实例生命周期内不允许随 manifest 变更而隐式漂移。
- 若只指定通道，版本由该通道 manifest v2 的 `artifacts[].version` 自动解析并锁定。
- 若显式指定版本，必须在选定通道内命中对应版本；否则视为无效选择并触发回退流程。

### 回退规则

- 选择源缺失或无效时，**默认回退到 stable 通道**，并锁定 stable manifest 的版本。
- 如果 stable 通道不可用或 manifest 解析失败，应直接报错并阻断实例创建，避免隐式漂移。

### 审计与日志

- 每次实例创建/启动时记录：选择来源、解析出的 `channel`/`version`、manifest schema 版本、命中的 `os/arch/path`。
- 若发生回退或版本不匹配，必须输出告警日志并写入审计事件，保留原始输入与回退原因。
- manifest v2 可用时同步记录 `checksums`/`metadata.build`，便于后续追溯与回滚。

---

## BROW-013 packaged bundle layout

本节定义桌面包/服务端分发时的浏览器产物布局，确保与 §6 的构建输出与 manifest v2 契约一致。

### macOS App Bundle

```text
YourApp.app/
  Contents/
    Resources/
      cloakbrowser/
        artifacts.json
        channels/
          stable/
            artifacts.json
            metadata/
              build.json
              checksums.json
            darwin-arm64/
              Chromium.app/Contents/MacOS/Chromium
```

- `cloakbrowser/` 为打包根目录，保持与 §6 的 `channels/<channel>/` 结构一致。
- `artifacts.json` 为选定通道的 manifest v2 副本，满足 app-relative 发现入口。
- `channels/<channel>/` 只需要当前平台/架构产物，其余平台不进入桌面包。

### Windows/Linux 桌面与服务端分发

```text
<app-dir>/resources/
  cloakbrowser/
    artifacts.json
    channels/<channel>/...

<install-root>/cloakbrowser/...
```

- Windows/Linux 默认放在 `<app-dir>/resources/cloakbrowser/`；服务端/CI 发行可直接使用 `<install-root>/cloakbrowser/`。
- 路径规则与 macOS 保持一致，确保运行时按统一相对路径解析。

### Manifest v2 与元数据映射

- `artifacts.json` 内容不做字段改写，保留 `channel` 与相对路径约定。
- 运行时解析 `artifacts[].path` 与 `metadata.*` 时，以 `cloakbrowser/channels/<channel>/` 作为基准目录，即使 manifest 位于 `cloakbrowser/artifacts.json`。
- 打包时必须同时带上所选通道的 `metadata/`，与 manifest 字段一一对应。

### 通道选择对打包内容的影响

1. 打包阶段必须显式选择通道（默认 `stable`），并仅携带该通道的浏览器产物与元数据。
2. 若需多通道共存，应完整保留多个 `channels/<channel>/` 子目录，并保证 `artifacts.json` 指向当前激活通道。
3. 切换通道即切换包内可解析的浏览器版本，不允许混用不同通道的二进制与 metadata。

---

## 7. 与当前运行时的衔接

### 7.1 已执行

本次代码执行已经把浏览器解析逻辑从“只认环境变量”扩展为：

- 先读显式 env override。
- 再读 `artifacts.json`。
- 再尝试 app-relative bundled path。

这意味着桌面包后续只要把浏览器产物与 manifest 一起带进安装包，`InstanceService -> CloakBrowserManager -> Client` 这一链路就能直接复用。

### 7.2 暂未执行

以下仍是后续阶段任务：

1. 真正的 Chromium 源码 fork 与 stealth patch overlay。
2. 跨平台编译流水线。
3. 产物签名、校验、发布渠道。
4. 浏览器版本回滚与灰度策略。

---

## 8. 检测基线规范（BROW-014）

本节用于固化 **FR-007 指纹诊断与验证闭环** 与 **Rule-005 发布门禁** 的检测基线输入、采样字段、门禁判定与输出工件，作为后续验证脚本与发布流水线的唯一基准。

### 8.1 基线检测来源

1. **公共检测站（Detection Sites）**
   - 指纹检测站：聚焦 UA / platform / locale / timezone / screen / hardware 识别。
   - 渲染检测站：聚焦 Canvas / WebGL / Audio 指纹稳定性。
   - 自动化检测站：聚焦 webdriver / automation flags / CDP 痕迹。
2. **业务验证站（Business Targets）**
   - 关键业务链路、风控关键路径、登陆/注册/支付等关键流程验证。
   - 标记为 “P0 关键站点”，作为发布门禁硬指标。
3. **内部诊断脚本（Internal Scripts）**
   - 指纹诊断/覆盖率采样（现有诊断命令与覆盖报告脚本）。
   - Runtime 冒烟采样（启动、CDP、user-data-dir、代理等基础能力）。

### 8.2 必采样字段

以下字段必须在所有采样源中统一采集，并进入 expected vs actual 比对：

- **UA**：`navigator.userAgent` + UA-CH（brands/fullVersionList）。
- **Platform**：`navigator.platform` / OS 标识与硬件架构。
- **Locale**：`navigator.language(s)` / `Intl.Locale`。
- **Timezone**：`Intl.DateTimeFormat().resolvedOptions().timeZone` + offset。
- **Screen**：`width/height/availWidth/availHeight/devicePixelRatio/colorDepth`。
- **WebGL**：vendor/renderer/unmasked vendor/renderer。
- **Canvas**：2D/bitmap fingerprint hash。
- **Audio**：audio fingerprint hash。

### 8.3 输出工件

检测基线必须产出以下工件，用于审计、回归与门禁判定：

1. **原始采样快照**：按站点/脚本输出采样 JSON（包含浏览器版本、渠道、时间戳）。
2. **规范化指纹快照**：归一化字段后的 baseline 视图（expected/actual）。
3. **差异报告**：字段级 diff（identity / rendering / automation 分组）。
4. **门禁汇总**：通过率、缺失字段数、关键失败项、最终 gate verdict。

### 8.4 门禁判定（Pass/Fail）

门禁判定以 “稳定进入 stable” 为基线，满足以下阈值才可通过：

1. **采样完整性**：必采样字段缺失率 **= 0%**。
2. **Identity 一致性**（UA / platform / locale / timezone / screen）：期望与实际 **完全一致**。
3. **Rendering 一致性**（WebGL / Canvas / Audio）：跨 3 次采样稳定率 **≥ 95%**，且与期望值不出现关键漂移。
4. **检测站通过率**：公共检测站整体通过率 **≥ 95%**，且任一类别不得低于 **90%**。
5. **业务验证站**：P0 关键站点通过率 **= 100%**。
6. **回归退化**：相对上一 stable 版本，通过率下降 **≤ 2%**，且无新增 P0 失败项。

### 8.5 与发布门禁衔接

检测基线结果必须与指纹一致性、runtime 冒烟、签名校验一并进入发布门禁。若任一门禁项失败，则 **禁止进入 stable**（Rule-005）。该 gate verdict 作为发布流水线与回滚策略的硬依赖，并与 manifest 版本元数据一同归档，形成 FR-007 的闭环证据。

---

## 9. 改进方案

### Phase 1：运行时可接入化

- 定义浏览器产物清单。
- 支持 app-relative runtime discovery。
- 保留 env override 作为开发与调试入口。

### Phase 2：构建产物标准化

- 为 macOS / Windows / Linux 产出统一目录结构。
- 约定产物命名、版本号、sha256、签名元数据。
- 让桌面包与服务端都消费同一套 artifact contract。

### Phase 3：源码构建链闭环

- 建立 Chromium 自维护 source fork。
- 管理 stealth / anti-detect / fingerprint patch overlay。
- 建立跨平台构建流水线与失败回滚流程。

### Phase 4：发布与验证闭环

- 引入签名 / notarization / checksum 校验。
- 引入冒烟验证：启动、指纹校验、CDP 健康检查。
- 引入版本发布记录与回滚策略。

---

## 10. 结论

PRD v2.0 的关键不是“拿到 CloakBrowser 源码”，而是把浏览器从一个外部依赖改造成一个**以 Chromium 为基线、自研 stealth 能力的受控产品子系统**。  
本次架构更新先把运行时入口改成可承接“源码构建产物”的形态，为后续真正的 Chromium 自建浏览器链路留出稳定接口。

> 详细实施细节见：`02-browser-core-technical-design.md`
