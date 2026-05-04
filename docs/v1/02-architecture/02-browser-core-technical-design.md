# 浏览器内核专项技术设计

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | Light Finger Browser |
| **设计主题** | 自建 Chromium-based stealth 浏览器 |
| **版本** | v1.0 |
| **对应 PRD** | docs/v1/01-prd.md v2.0 |
| **对应架构** | 01-browser-build-runtime.md |
| **更新日期** | 2026-04-30 |

---

## 1. 设计目标

本设计文档回答的不是“为什么要做浏览器”，而是“**这款浏览器要怎么设计出来**”。

核心目标：

1. 以 **Chromium** 为源码基线，自主构建 macOS / Windows / Linux 三端浏览器产物。
2. 以 **stealth patch overlay** 的方式组织反检测能力，而不是依赖第三方私有二进制。
3. 让当前仓库继续承担 **实例管理、指纹生成、代理绑定、运行时发现、诊断验证** 这层职责。
4. 在验证能力、发布能力和可集成性上，做出一款整体上超过 CloakBrowser 的产品。

---

## 2. 范围与边界

### 2.1 In Scope

1. Chromium 基线选择策略
2. stealth patch overlay 分类与边界
3. 浏览器构建、签名、发布架构
4. 浏览器产物与当前 Go/Wails 运行时的契约
5. 浏览器版本验证与发布门禁

### 2.2 Out of Scope

1. 任意具体业务站点自动化流程设计
2. 浏览器产品之外的上层业务逻辑
3. 直接在本仓库内 vendoring 全量 Chromium 源码树

### 2.3 仓库边界

本项目建议拆成两个协作层：

| 层 | 仓库/工作区建议 | 职责 |
|----|----------------|------|
| Browser Core Program | 独立浏览器源码工作区 | Chromium fork、patch overlay、构建、签名、浏览器产物输出 |
| Current App Repo | 当前仓库 | 指纹模型、实例管理、代理绑定、运行时发现、诊断和业务编排 |

**决策**：不把完整 Chromium 源码树直接塞进当前业务仓库。  
原因：源码体量过大，构建依赖复杂，版本演进节奏与业务层不同。

### 2.4 Browser Core Workspace 规范（BROW-001）

本节定义独立 browser-core 工作区的边界、目录布局与产物契约，作为后续 Chromium 拉取与构建的前置约束。

#### 2.4.1 Workspace Root 与边界

- **Workspace 名称**：`lfb-browser-core`（建议仓库名/工作区名）。
- **边界**：仅包含 Chromium 源码、patch overlay、构建产物与构建脚本；不包含当前仓库的指纹/实例/代理逻辑。
- **与当前仓库关系**：通过产物 manifest 与运行时契约集成，当前仓库不直接依赖或引用源码。

#### 2.4.2 顶层目录结构（必须）

| 目录 | 说明 |
|------|------|
| `src/` | Chromium 源码工作区（`fetch`/`gclient` 同步结果） |
| `patches/` | stealth patch overlay 分层目录 |
| `build/` | `gn`/`ninja` 输出与中间构建目录 |
| `artifacts/` | 浏览器发布制品与 manifest 输出 |
| `scripts/` | 拉取、构建、签名、打包等脚本入口 |
| `docs/` | browser-core 专属说明与操作文档 |

#### 2.4.3 版本与元数据文件

| 文件 | 位置 | 说明 |
|------|------|------|
| `chromium_revision` | workspace 根目录 | 固定 Chromium git revision |
| `chromium_milestone` | workspace 根目录 | 对应 Chromium milestone |
| `browser_version` | workspace 根目录 | `lfb-browser/<milestone>.<patch>.<seq>` |
| `patch_train` | workspace 根目录 | patch train 编号 |
| `metadata/build.json` | `artifacts/<browser-version>/metadata/` | 构建时间、git revision、checksum、签名状态 |
| `artifacts.json` | `artifacts/<browser-version>/` | 运行时消费的产物清单 |

#### 2.4.4 产物输出目录（高阶约定）

```text
artifacts/
  <browser-version>/
    artifacts.json
    metadata/
      build.json
    <platform>/
      <browser-binary-or-package>
```

#### 2.4.5 与当前仓库的集成契约

1. browser-core 仅输出 **manifest + 制品目录**，不输出业务代码。
2. 当前仓库通过 **manifest/runtime contract** 发现浏览器产物（见 6.3），并将 manifest 同步到运行时可读取的位置（例如 `resources/cloakbrowser/` 下的产物清单）。
3. 运行时仅依赖 manifest 与版本元数据，禁止直接引用 browser-core 源码路径或构建目录。

---

## 3. Chromium 基线策略

### 3.1 基线选择规则

首个可执行版本不以“最新实验版”为目标，而采用以下规则：

1. 基于 **Chromium Stable** 主线版本
2. 每个浏览器 release 固定一个 `chromium_milestone`
3. 默认采用“**一个内部版本只追一个 Chromium 里程碑**”策略，避免同时跨多里程碑维护

### 3.2 基线锁定与变更控制（BROW-004）

**首个稳定里程碑选择**

1. 选择当前最新的 **Stable milestone**，且已完成至少一次稳定补丁发布（避免“刚切 stable”风险）。
2. 必须满足：单平台最小编译通过、启动/退出/CDP 冒烟通过、检测基线无关键退化。
3. 若最新 milestone 无法满足上述条件，则回退到上一个 stable milestone。

**Pinning 记录与产物绑定（BROW-001 文件）**

1. `chromium_milestone`：记录选定 milestone（唯一活跃里程碑）。
2. `chromium_revision`：记录具体 git revision，作为源码同步与构建唯一基线。
3. `browser_version`：记录 `lfb-browser/<milestone>.<patch-train>.<release-seq>`，对外发布与运行时锁定使用。
4. `patch_train`：仅描述 patch 线，不允许隐式变更 Chromium 基线。
5. `artifacts.json` + `metadata/build.json` 必须回写上述版本/revision，确保运行时可验证。

**更新节奏与回滚预期**

1. 同一 milestone 内：安全更新优先快跟（目标 7 天内完成基线更新与验证）。
2. 跨 milestone 升级：按季度评估或风控驱动评估，必须通过完整验证与回归对比后切换。
3. 回滚：稳定通道至少保留最近两个 `browser_version` 产物，回滚通过 manifest 版本锁定完成。

**防漂移护栏**

1. `chromium_revision` 变更必须通过明确的基线变更流程（含验证门禁），禁止自动跟随 upstream。
2. `patch_train` 递增仅代表 patch 迭代，不得改变 `chromium_milestone` 与 `chromium_revision`。
3. 运行时只认 `browser_version` + manifest，禁止直接依赖源码工作区或构建目录。

### 3.3 同步策略

| 类型 | 策略 |
|------|------|
| 安全更新 | 同一 milestone 内优先快跟 |
| 功能更新 | 按季度或按风控变化评估 |
| 紧急回滚 | 保留前一稳定版本浏览器产物 |

### 3.4 版本命名

建议版本格式：

```text
lfb-browser/<chromium-milestone>.<patch-train>.<release-seq>
```

示例：

```text
lfb-browser/136.1.0
lfb-browser/136.2.0
```

其中：

- `136`：Chromium milestone
- `1`：stealth patch train
- `0`：发布序号

---

## 4. Stealth Patch Overlay 设计

### 4.1 设计原则

1. **补丁分层**：按行为域分类，不做散乱 patch 堆叠
2. **最小侵入**：每个 patch 只负责一个行为域
3. **可验证**：每类 patch 必须对应检测脚本或指纹采样结果
4. **可关闭**：关键 patch 允许通过 build flag 或 runtime flag 关闭以便诊断

### 4.2 补丁分类

| 分类 | 目标 | 典型能力 |
|------|------|----------|
| Identity Patch | 统一环境身份 | UA、platform、locale、timezone、screen、hardware 值一致性 |
| Rendering Patch | 统一渲染输出 | Canvas、WebGL、Audio、fonts、client rect、GPU 呈现 |
| Automation Patch | 降低自动化痕迹 | webdriver、automation flags、CDP 暴露、输入事件特征 |
| Network Patch | 降低网络侧异常 | WebRTC、RTT、downlink、connection type、timing 抖动 |
| Behavior Patch | 提升交互自然度 | 鼠标曲线、键盘节奏、滚动、焦点切换、空闲行为 |
| Profile Patch | 强化实例隔离 | user-data-dir、cookie/cache/storage/session 隔离 |

### 4.3 Patch Overlay 目录与命名（BROW-017）

**目录布局（强制）**

```text
patches/
  identity/
  rendering/
  automation/
  network/
  behavior/
  profile/
```

- **顶层目录即补丁域**：只能使用以上 6 类，禁止新增并行目录。
- **补丁 train 映射**：每个补丁域下按 `train-<patch-train>` 划分 patch set，`patch_train` 文件指向当前启用的 train。
- **Patch set 统一入口**：每个 train 必须包含 `patchset.json`，用于声明该 train 下的 patch 列表与默认开关状态。

```json
{
  "train": 1,
  "domain": "identity",
  "patches": ["identity/ua-platform-consistency", "identity/timezone-consistency"]
}
```

**命名规范**

1. 目录与 patch id 统一使用 `kebab-case`。
2. patch id 规则：`<domain>/<patch-name>`，例如 `identity/ua-platform-consistency`。
3. patch set 文件固定为 `patchset.json`，patch 元数据文件固定为 `metadata.json`。

**Patch 元数据结构（metadata.json）**

```json
{
  "id": "identity/ua-platform-consistency",
  "domain": "identity",
  "description": "统一 UA 与 platform 输出",
  "owner": "browser-core",
  "status": "active",
  "default_enabled": true,
  "toggle_key": "patch.identity.ua-platform-consistency",
  "files": ["patch.diff"]
}
```

- `toggle_key` 仅定义命名规范，具体 build/runtime 开关在 BROW-018 统一落地。
- `files` 记录 patch 文件名，允许 `.patch`/`.diff` 多文件。

**Patch train 与 patch set 对应关系**

- **patch_train = N** 表示启用 `patches/<domain>/train-N/patchset.json` 中列出的 patch set。
- 同一 `patch_train` 必须覆盖 6 个补丁域，形成一个完整的 stealth patch overlay 版本。

**护栏**

- 所有 stealth 修改必须归档到 `patches/` overlay 内，禁止在 `src/` 或脚本中散落 ad-hoc patch。
- 不允许绕过 overlay 直接改 Chromium 源码树，除非回写为 patch 并登记 metadata。

### 4.4 Build/Runtime Switch 策略（BROW-018）

本节定义 patch 版本选择与运行时开关的**唯一策略**，并明确与 metadata 中 `toggle_key` 的映射关系。

**Build-time Switch（patch_train）**

1. **唯一构建开关**：`patch_train` 是构建期选择 patch 版本的唯一入口。
2. **来源优先级**：默认读取 workspace 根目录 `patch_train` 文件；允许构建命令显式覆盖，但必须回写到 `build.json` 与 `artifacts.json`。
3. **作用范围**：`patch_train = N` 必须同时选中 6 个域的 `train-N/patchset.json`，保证完整的 overlay 版本一致性。

**Runtime Toggle 映射（toggle_key）**

1. **唯一命名规范**：运行时开关只允许使用 metadata 中 `toggle_key`，例如 `patch.identity.ua-platform-consistency`。
2. **配置载体无关**：开关可来自 CLI flag / runtime config / env map，但必须以 `toggle_key=bool` 映射。
3. **manifest 对齐**：构建阶段需导出该 `toggle_key` 列表，运行时只能解析清单内允许的 key。

**默认启用与安全覆盖**

1. **默认状态**：`default_enabled` 为唯一默认值；运行时未指定则继承默认。
2. **安全覆盖**：允许运行时**关闭**任何 patch 以便诊断；**开启默认关闭 patch** 仅允许在 alpha/beta 或显式 allowlist 下。
3. **审计回写**：任何非默认开关必须记录到运行时诊断/manifest 中，便于回溯。

**护栏：禁止脱离 metadata 的开关**

1. 运行时发现未知 `toggle_key` 必须拒绝或硬告警（不可静默忽略）。
2. 禁止在 build/runtime 中新增与 metadata 不一致的 ad-hoc flags。
3. patchset 与 metadata 为唯一真源，任何开关新增必须回写 `metadata.json`。

### 4.5 Patch 输入来源

| 输入源 | 提供方 | 用途 |
|--------|-------|------|
| 指纹种子 | 当前仓库 `fingerprint/` | 生成一致性环境参数 |
| 指纹字段 | 当前仓库 `fingerprint/` | 作为浏览器 patch 的 runtime 输入 |
| 代理信息 | 当前仓库 `proxy/` | 网络/地区/时区关联输入 |
| 行为模式配置 | 浏览器侧配置 | humanize / automation 行为策略 |

### 4.6 Patch 应用边界

| 层 | 是否允许 patch | 说明 |
|----|----------------|------|
| Chromium C++/平台层 | 是 | 核心 stealth/anti-detect 逻辑主要在此 |
| 启动参数层 | 是 | 允许暴露 deterministic runtime knobs |
| JS 注入层 | 仅兜底 | 不作为核心 stealth 能力主要实现方式 |

**决策**：JS 注入只能作为补救层，不能替代内核 patch。

### 4.7 Identity Patch：Platform/Locale/Timezone（BROW-019）

首批 Identity Patch 用于锁定“平台 + 语言/地区 + 时区”三要素，作为后续 screen/hardware 的前置一致性基线。

**目标字段（必须规范化）**

1. `navigator.platform`
2. `navigator.language` / `navigator.languages`
3. `Intl.DateTimeFormat().resolvedOptions().locale`
4. `Intl.DateTimeFormat().resolvedOptions().timeZone`

**Patch 边界**

- 仅覆盖 platform/locale/timezone 三类字段；不包含 screen、hardware、UA 其它字段。
- 默认与运行时指纹模型一致（以指纹种子为唯一期望值来源）。
- 任何变更必须归档为 identity patch，禁止在 `src/` 中散落 ad-hoc 修改。

**Patch 元数据与开关（遵循 BROW-017/BROW-018）**

- 建议拆为三个 patch，便于单独开关：
  - `identity/platform-consistency` → `toggle_key: patch.identity.platform-consistency`
  - `identity/locale-consistency` → `toggle_key: patch.identity.locale-consistency`
  - `identity/timezone-consistency` → `toggle_key: patch.identity.timezone-consistency`
- `patchset.json` 需在 identity/train-<N>/ 中声明以上 patch，保持与 `patch_train` 对齐。

**指纹种子与代理 Geo 关系**

1. **指纹种子优先**：platform/locale/timezone 的期望值以指纹种子为唯一真源。
2. **代理 Geo 校验**：当代理提供国家/区域信息时，必须校验 locale/timezone 与代理 Geo 一致；不一致需记录为验证失败并阻断进入稳定通道。
3. **缺省补全**：指纹种子缺失时，可使用代理 Geo 补全 locale/timezone，但需回写为运行时诊断记录，避免隐式漂移。

**验证步骤（expected vs actual）**

1. 运行时加载指纹种子与代理 Geo，生成 expected 值集（platform/locale/timezone）。
2. 浏览器内采样 actual 值：
   - `navigator.platform`
   - `navigator.language` / `navigator.languages`
   - `Intl.DateTimeFormat().resolvedOptions().locale`
   - `Intl.DateTimeFormat().resolvedOptions().timeZone`
3. 生成差异报告：字段级对比 + 代理 Geo 一致性检查。
4. expected 与 actual 必须完全一致；任一字段不一致视为 patch 失败。

**护栏**

- 禁止 JS-only patch；核心实现必须在 Chromium C++/平台层完成。
- JS 注入仅用于诊断采样或兜底验证，不作为主实现路径。

### 4.8 Identity Patch：Screen/Hardware（BROW-020）

本节定义屏幕与硬件身份的一致性补丁规范，作为 Identity Patch 的第二批落地，确保屏幕/硬件特征与指纹种子严格对齐。

**目标字段（必须规范化）**

1. `screen.width` / `screen.height`
2. `screen.availWidth` / `screen.availHeight`
3. `screen.availTop` / `screen.availLeft`
4. `window.devicePixelRatio`
5. `screen.colorDepth` / `screen.pixelDepth`
6. `navigator.hardwareConcurrency`
7. `navigator.deviceMemory`
8. GPU 基础信息（WebGL 采样）：
   - `UNMASKED_VENDOR_WEBGL`
   - `UNMASKED_RENDERER_WEBGL`

**规范化与一致性规则**

1. **屏幕尺寸**：`width/height` 与 `availWidth/availHeight` 必须为正整数；`availWidth <= width`、`availHeight <= height` 且 `availTop/availLeft >= 0`。
2. **devicePixelRatio**：必须与指纹种子一致，保留最多 2 位小数；不同视口下不得漂移。
3. **色深**：`colorDepth` 与 `pixelDepth` 必须一致，值限定在 `24/30/32`（由指纹种子指定）。
4. **硬件并发**：`hardwareConcurrency` 必须为正整数并与指纹种子一致，不允许浏览器自适应变动。
5. **设备内存**：`deviceMemory` 必须为指纹种子值，且必须在 Chrome 允许的离散集合内（如 `0.25/0.5/1/2/4/8/16/32`）。
6. **GPU 基础信息**：WebGL1 与 WebGL2 返回的 vendor/renderer 必须与指纹种子一致，且在同一实例内保持恒定。

**Patch 边界**

- 仅覆盖屏幕与硬件字段；不改动 UA、timezone、rendering 输出等其他域。
- 必须归档为 identity patch，禁止在 `src/` 中散落 ad-hoc 修改。

**Patch 元数据与开关（遵循 BROW-017/BROW-018）**

- 建议拆为三个 patch，便于独立开关：
  - `identity/screen-consistency` → `toggle_key: patch.identity.screen-consistency`
  - `identity/hardware-consistency` → `toggle_key: patch.identity.hardware-consistency`
  - `identity/gpu-basics-consistency` → `toggle_key: patch.identity.gpu-basics-consistency`
- `patchset.json` 需在 identity/train-<N>/ 中声明以上 patch，并与 `patch_train` 对齐。

**指纹种子交互**

1. **指纹种子唯一真源**：屏幕/硬件/GPU 的 expected 值只能来自指纹种子。
2. **缺失处理**：指纹种子缺失该类字段时，必须记录为验证失败并阻断进入 stable；仅允许在 alpha/beta 通过显式 allowlist 使用默认兜底值。
3. **一致性回写**：任何兜底或 override 必须写入运行时诊断日志与 manifest 采样记录。

**验证步骤（expected vs actual）**

1. 运行时加载指纹种子，生成 expected 值集（screen/hardware/gpu）。
2. 浏览器内采样 actual 值：
   - `screen.*` / `devicePixelRatio`
   - `navigator.hardwareConcurrency` / `navigator.deviceMemory`
   - WebGL `UNMASKED_VENDOR_WEBGL` / `UNMASKED_RENDERER_WEBGL`
3. 进行字段级对比 + 交叉校验：
   - `availWidth/availHeight` 与 `width/height` 关系校验
   - `colorDepth` 与 `pixelDepth` 一致性校验
4. expected 与 actual 必须完全一致；任一字段不一致视为 patch 失败并进入阻断或降级通道。

**护栏**

- 禁止 JS-only patch；核心实现必须在 Chromium C++/平台层完成。
- JS 注入仅用于诊断采样或兜底验证，不作为主实现路径。

### 4.9 Identity Validation Run（BROW-021）

本节定义 identity patch 的专项验证运行，基于 **BROW-015 fingerprint-baseline-runner** 的采样/差异引擎，聚焦 identity 字段的一致性验证，作为 rendering / automation patch 进入验证与发布前的硬门禁，并纳入 **FR-007** 指纹诊断闭环。

**输入与触发条件**

1. `BROW-010` runtime smoke runner 已通过。
2. `BROW-015` baseline runner 可用（采样与输出协议保持一致）。
3. **expected profile**：由指纹种子生成的期望值集合，覆盖 platform/locale/timezone/screen/hardware/gpu；如有代理 Geo，需同时包含 locale/timezone 的 Geo 期望。
4. **runtime artifact**：browser binary + `artifacts.json` + `metadata/build.json`，并明确 `patch_train`、channel（alpha/beta/stable）与启动参数。

**校验范围（必须全部通过）**

1. **Platform / Locale / Timezone**
   - `navigator.platform`
   - `navigator.language` / `navigator.languages`
   - `Intl.DateTimeFormat().resolvedOptions().locale`
   - `Intl.DateTimeFormat().resolvedOptions().timeZone`
2. **Screen**
   - `screen.width/height/availWidth/availHeight/availTop/availLeft`
   - `window.devicePixelRatio`
   - `screen.colorDepth` / `screen.pixelDepth`
3. **Hardware**
   - `navigator.hardwareConcurrency`
   - `navigator.deviceMemory`
4. **GPU 基础信息**
   - WebGL `UNMASKED_VENDOR_WEBGL` / `UNMASKED_RENDERER_WEBGL`

**验证流程（expected vs actual）**

1. 使用与 `BROW-015` 一致的运行参数启动 runtime，采样 identity 字段 actual snapshot。
2. 按 `BROW-019`/`BROW-020` 规则对 expected/actual 做字段级 normalization + diff。
3. 如启用代理 Geo，需额外校验 locale/timezone 与代理 Geo 一致性；不一致视为失败。
4. 生成 identity diff 报告与 gate verdict，任何字段 mismatch 或采样缺失即判失败。

**输出工件（对齐 BROW-015 结构）**

1. `identity-validation-report.json`：版本、channel、patch_train、采样时间、gate verdict。
2. `identity-diff.json`：字段级 diff 明细（identity 子集）。
3. `identity-snapshots/`：raw snapshot 备份（actual + expected）。
4. `runner.log`：启动、采样、diff 过程日志。

**门禁与通道影响（FR-007 对齐）**

1. **Pass**：identity 字段全部匹配，且无缺失字段；允许进入 rendering/automation validation（BROW-025/BROW-028）。
2. **Fail**：任一字段 mismatch、采样失败、或 Geo 不一致即判失败；必须阻断：
   - rendering / automation patch 验证链路
   - channel promotion（beta/stable）与发布流水线
3. Fail 仅允许留在 alpha 做问题排查，不可进入稳定通道。

**失败处理**

- 必须输出失败原因、复现参数与对应字段差异。
- 缺失 expected profile 或指纹字段时直接判失败，禁止使用隐式默认值。
- 失败结果需回写运行时诊断与发布门禁记录，作为 FR-007 闭环证据。


---

### 4.10 Rendering Patch：Canvas（BROW-022）

Canvas 是 Rendering Patch 的核心指纹面，必须纳入 patch overlay 管理并与 FR-007 验证闭环对齐。

**目标 API（必须覆盖）**

1. `HTMLCanvasElement.toDataURL`
2. `HTMLCanvasElement.toBlob`
3. `CanvasRenderingContext2D.getImageData`
4. `OffscreenCanvas.convertToBlob`
5. `OffscreenCanvas.transferToImageBitmap`

**Patch 行为目标**

1. **确定性**：同一指纹 seed + 相同绘制输入必须输出一致指纹结果（跨会话可复现）。
2. **可控扰动**：仅在像素读取/导出路径注入 seed-based 变换，不影响页面视觉渲染。
3. **一致性基线**：2D/bitmap 采样 hash 与 expected 值对齐，避免随机噪声导致漂移。

**Patch 元数据与开关（遵循 BROW-017/BROW-018）**

- patch id：`rendering/canvas-consistency`
- toggle_key：`patch.rendering.canvas-consistency`
- `patchset.json` 需在 `rendering/train-<N>/` 中声明以上 patch，确保与 `patch_train` 对齐。

**验证步骤（FR-007 对齐）**

1. 由指纹种子生成 expected canvas hash（2D + bitmap），作为基线目标值。
2. 运行时连续采样 3 次（同一绘制脚本），通过 `toDataURL` / `getImageData` / `toBlob` 计算实际 hash。
3. 判定标准：
   - **稳定性**：跨 3 次采样稳定率 ≥ 95%。
   - **一致性**：actual 与 expected 不出现关键漂移（字段级 diff = 0）。
4. 输出 diff 报告与门禁结果，纳入 FR-007 指纹诊断闭环与发布门禁。

**护栏**

- 禁止 JS-only patch；核心实现必须在 Chromium C++/平台层完成。
- JS 注入仅用于诊断采样或兜底验证，不作为主实现路径。
- 不允许绕过 `toggle_key` 临时开关或通过脚本修改 Canvas API 行为。

---

### 4.11 Rendering Patch：WebGL（BROW-023）

WebGL 是 Rendering Patch 的关键指纹面，必须在 patch overlay 中定义可控范围，并与 FR-007 验证闭环对齐。

**目标 API（必须覆盖）**

1. `WebGLRenderingContext.getParameter` / `WebGL2RenderingContext.getParameter`
2. `WebGLRenderingContext.getSupportedExtensions`
3. `WebGLRenderingContext.getExtension`
4. `WebGLRenderingContext.getShaderPrecisionFormat` / `WebGL2RenderingContext.getShaderPrecisionFormat`
5. `WebGLRenderingContext.getContextAttributes`（需保证与种子一致的可见属性）

**Patch 行为目标**

1. **确定性**：同一指纹 seed + 相同上下文配置返回一致结果（跨会话可复现）。
2. **种子驱动**：vendor/renderer、extension 列表、precision/limit 值以指纹种子为唯一来源，不得漂移或被设备自适应覆盖。
3. **跨上下文一致**：WebGL1/WebGL2 与多实例 context 结果一致；同一 GPU 指纹在所有采样点保持稳定。

**Patch 元数据与开关（遵循 BROW-017/BROW-018）**

- patch id：`rendering/webgl-consistency`
- toggle_key：`patch.rendering.webgl-consistency`
- `patchset.json` 需在 `rendering/train-<N>/` 中声明以上 patch，确保与 `patch_train` 对齐。

**验证步骤（FR-007 对齐）**

1. 由指纹种子生成 expected WebGL 配置（vendor/renderer、extensions、shader precision、关键 limits）。
2. 运行时分别创建 WebGL1/WebGL2 context，连续采样 3 次并记录：
   - `getParameter`（含 vendor/renderer 与关键 limits）
   - `getSupportedExtensions` / `getExtension`
   - `getShaderPrecisionFormat`
3. 判定标准：
   - **稳定性**：跨 3 次采样稳定率 ≥ 95%。
   - **一致性**：actual 与 expected 完全一致；WebGL1/WebGL2 结果无漂移。
4. 输出 diff 报告与门禁结果，纳入 FR-007 指纹诊断闭环与发布门禁。

**护栏**

- 禁止 JS-only patch；核心实现必须在 Chromium C++/GPU/ANGLE 层完成。
- JS 注入仅用于诊断采样或兜底验证，不作为主实现路径。
- 不允许通过运行时脚本修改 WebGL API 行为或绕过 `toggle_key`。

---

### 4.12 Rendering Patch：Audio/Fonts（BROW-024）

Audio 与字体渲染是高风险指纹面，必须纳入 patch overlay 管理，并与 FR-007 验证闭环对齐。

**目标 API（必须覆盖）**

1. `AudioContext` / `OfflineAudioContext`（含 `startRendering`）
2. `AudioBuffer.getChannelData`
3. `AnalyserNode.getFloatFrequencyData` / `getByteFrequencyData`
4. `CanvasRenderingContext2D.measureText` / `TextMetrics`（字体度量）
5. `FontFaceSet.check` / `document.fonts`（字体可用性与枚举）

**Patch 行为目标**

1. **确定性**：同一指纹 seed + 相同音频图与文本输入必须输出一致结果（跨会话可复现）。
2. **种子驱动**：音频指纹扰动与字体度量偏移仅由 seed 驱动，禁止设备自适应或随机噪声。
3. **功能隔离**：patch 仅影响采样/读数路径，不改变可听输出或可见渲染效果。
4. **跨上下文一致**：`AudioContext` 与 `OfflineAudioContext`、Canvas/DOM 字体度量结果保持一致。

**Patch 元数据与开关（遵循 BROW-017/BROW-018）**

- patch id：`rendering/audio-fonts-consistency`
- toggle_key：`patch.rendering.audio-fonts-consistency`
- `patchset.json` 需在 `rendering/train-<N>/` 中声明以上 patch，确保与 `patch_train` 对齐。

**验证步骤（FR-007 对齐）**

1. 由指纹种子生成 expected audio hash 与字体度量基线（含 canvas/DOM 文本样本）。
2. 运行时连续采样 3 次：
   - `OfflineAudioContext` 渲染音频图并计算 hash。
   - `AudioContext` 采样分析节点输出并计算 hash。
   - 字体度量（`measureText` + DOM 文本测量）输出指标集合。
3. 判定标准：
   - **稳定性**：跨 3 次采样稳定率 ≥ 95%。
   - **一致性**：actual 与 expected 无字段级 diff；AudioContext/OfflineAudioContext 与多字体采样无漂移。
4. 输出 diff 报告与门禁结果，纳入 FR-007 指纹诊断闭环与发布门禁。

**护栏**

- 禁止 JS-only patch；核心实现必须在 Chromium C++/Blink/Audio 栈完成。
- JS 注入仅用于诊断采样或兜底验证，不作为主实现路径。
- 不允许通过脚本修改 Audio/Font API 行为或绕过 `toggle_key`。

---

### 4.13 Rendering Validation Run（BROW-025）

本节定义 rendering patch 的专项验证运行，基于 **BROW-015 fingerprint-baseline-runner** 的采样/差异引擎与 **BROW-016 regression-diff-runner** 的回归对比能力，覆盖 Canvas/WebGL/Audio-Fonts 的一致性与回归风险，作为进入 automation 验证与发布门禁前的硬门禁，并纳入 **FR-007** 指纹诊断闭环。

**输入与触发条件**

1. `BROW-021` identity validation 已通过。
2. `BROW-015` baseline runner 与 `BROW-016` regression diff runner 可用（采样与输出协议保持一致）。
3. **expected rendering baselines**：由指纹种子生成的渲染期望值集合：
   - Canvas：2D/bitmap hash。
   - WebGL：vendor/renderer、extensions、precision/limit、context attributes。
   - Audio/Fonts：audio hash 与字体度量/可用性基线。
4. **runtime artifact**：browser binary + `artifacts.json` + `metadata/build.json`，并明确 `patch_train`、channel（alpha/beta/stable）与启动参数。
5. **previous stable baselines（用于回归对比）**：用于 BROW-016 的上一稳定版本 baseline 产物；alpha 可选，beta/stable 必需。

**校验范围（必须全部通过）**

1. **Canvas（BROW-022）**：`toDataURL`/`getImageData`/`toBlob` 采样，稳定率 ≥ 95%，actual vs expected diff=0。
2. **WebGL（BROW-023）**：WebGL1/WebGL2 参数、extensions、precision/limits 采样，稳定率 ≥ 95%，actual vs expected diff=0。
3. **Audio/Fonts（BROW-024）**：audio hash、AudioContext/OfflineAudioContext 输出、字体度量与可用性采样，稳定率 ≥ 95%，actual vs expected diff=0。

**验证流程（expected vs actual + regression）**

1. 使用 `BROW-015` 采样引擎生成 rendering actual snapshot，并对 expected 做 normalization + diff。
2. 当提供 previous stable baselines 时，使用 `BROW-016` 对 rendering 子集做回归对比，输出 regression verdict。
3. 任一渲染域 mismatch、采样缺失、或 regression 超阈值即判失败。

**输出工件（对齐 BROW-015/BROW-016 结构）**

1. `rendering-validation-report.json`：版本、channel、patch_train、采样时间、gate verdict。
2. `rendering-diff.json`：Canvas/WebGL/Audio-Fonts 字段级 diff 明细。
3. `rendering-snapshots/`：raw snapshot 备份（actual + expected）。
4. `regression-diff.json` / `regression-summary.json`：rendering 子集回归对比结果（若触发 BROW-016）。
5. `runner.log`：启动、采样、diff、regression 过程日志。

**门禁与通道影响（FR-007 对齐）**

1. **Pass**：三类渲染域全部匹配，且 regression verdict 为 Pass（若适用）；允许进入 `BROW-028` automation validation，并进入 channel promotion。
2. **Fail**：任一渲染域 mismatch、采样失败、缺失 baseline、或 regression fail 即判失败；必须阻断：
   - automation validation 及后续发布流水线
   - channel promotion（beta/stable）
3. Fail 仅允许留在 alpha 做问题排查，不可进入稳定通道。

**失败处理**

- 必须输出失败原因、复现参数、patch_train 与对应字段差异。
- 缺失 expected baselines 或 regression 对比基线时直接判失败，禁止使用隐式默认值。
- 失败结果需回写运行时诊断与发布门禁记录，作为 FR-007 闭环证据。

---

### 4.14 Automation Patch：Signals（BROW-026）

Automation Patch 需要对自动化信号进行明确补丁范围定义，并纳入 FR-007 验证闭环，避免“仅靠脚本遮蔽”的脆弱策略。

**目标信号（必须覆盖）**

1. `navigator.webdriver` / `Navigator.webdriver` 输出
2. Automation flags 与暴露痕迹：
   - `--enable-automation`/automation infobar
   - `ChromeAutomationExtension` 等自动化扩展痕迹
   - `--disable-blink-features=AutomationControlled` 等行为标记
3. CDP 相关暴露：
   - `--remote-debugging-port/pipe` 引入的可探测标记
   - `window.cdc_*`/driver 注入的 CDP artifacts

**Patch 行为目标**

1. **Stealth**：自动化信号在浏览器内不可被脚本或检测站直接识别。
2. **确定性**：同一 patch train + runtime flags 下结果一致，不做随机化或环境自适应漂移。
3. **最小影响**：仅修复检测面，不改变正常功能链路与业务脚本行为。

**Patch 元数据与开关（遵循 BROW-017/BROW-018）**

- patch id：`automation/automation-signals`
- toggle_key：`patch.automation.automation-signals`
- `patchset.json` 需在 `automation/train-<N>/` 中声明以上 patch，确保与 `patch_train` 对齐。

**验证步骤（FR-007 对齐）**

1. 运行时采样自动化信号基线（webdriver/flags/CDP artifacts）。
2. 同一实例内连续采样 3 次，记录每次结果并做 diff。
3. 至少在 2 个检测站执行 automation 信号检测与行为校验。
4. 判定标准：
   - **稳定性**：跨 3 次采样输出一致，diff=0。
   - **可用性**：检测站不暴露 automation/driver/CDP 相关异常提示。
5. 输出 automation diff 报告与门禁结果，纳入 FR-007 指纹诊断闭环与发布门禁。

**护栏**

- 禁止 JS-only patch；核心实现必须在 Chromium C++/平台层完成。
- JS 注入仅用于诊断采样或兜底验证，不作为主实现路径。
- 不允许绕过 `toggle_key` 临时隐藏或恢复 automation 信号。

---

### 4.15 Automation Patch：Input Behavior（BROW-027）

输入行为是自动化痕迹的核心信号之一，需在 patch overlay 中明确定义可控范围，并与 FR-007 验证闭环对齐。

**目标行为（必须覆盖）**

1. **鼠标移动**：速度/加速度曲线、曲率变化、停顿与微抖动。
2. **鼠标点击**：`mousedown`→`mouseup` 间隔、双击间隔、点击前后停顿。
3. **滚动节奏**：滚轮/触控板 delta 级别、连续滚动 cadence、惯性衰减。
4. **键盘输入**：按键 dwell time、按键 flight time、连续输入节奏。

**Patch 行为目标**

1. **Humanized**：输入节奏具备自然抖动与停顿，避免固定间隔/线性轨迹等自动化特征。
2. **Deterministic per seed**：相同指纹 seed + 相同输入脚本应输出一致事件序列与时间分布（跨会话可复现）。
3. **可控范围**：所有节奏与抖动参数必须来自行为 profile 配置，禁止 runtime 随机漂移。

**Patch 元数据与开关（遵循 BROW-017/BROW-018）**

- patch id：`automation/input-behavior`
- toggle_key：`patch.automation.input-behavior`
- `patchset.json` 需在 `automation/train-<N>/` 中声明以上 patch，确保与 `patch_train` 对齐。

**验证步骤（FR-007 对齐）**

1. 由指纹 seed 生成 expected 输入 profile（鼠标/键盘/滚动 cadence 参数）。
2. `BROW-028 automation-validation-run` 启动标准化输入脚本，连续采样 3 次并记录 event trace：
   - mouse move/click：路径、速度、加速度、停顿分布
   - scroll：delta 序列、间隔、衰减曲线
   - keyboard：dwell/flight time 分布
3. 判定标准：
   - **稳定性**：跨 3 次采样稳定率 ≥ 95%。
   - **一致性**：actual 与 expected 的 cadence 分布差异在允许阈值内。
   - **反自动化**：不存在固定间隔、零抖动或单一速度曲线等明显自动化特征。
4. 输出 diff 报告与门禁结果，纳入 FR-007 指纹诊断闭环与发布门禁。

**护栏**

- 禁止 JS-only patch；核心实现必须在 Chromium 输入管线完成。
- JS 注入仅用于诊断采样或兜底验证，不允许改写事件时间戳或序列。
- 不允许绕过 `toggle_key` 在脚本层临时调整输入节奏。

---

### 4.16 Automation Validation Run（BROW-028）

本节定义 automation patch 的专项验证运行，基于 **BROW-015 fingerprint-baseline-runner** 的采样/差异引擎，并输出可供 **BROW-016 regression-diff-runner** 对比的快照，作为 automation patch 进入发布前的硬门禁，纳入 **FR-007** 指纹诊断闭环。

**输入与触发条件**

1. `BROW-021` identity validation 已通过，避免身份域不一致导致误判。
2. `BROW-015` baseline runner 可用（采样与输出协议保持一致）。
3. **expected automation baselines**：
   - automation signals baseline：webdriver/flags/CDP artifacts 的期望结果集。
   - input behavior baseline：由指纹 seed 生成的 mouse/keyboard/scroll cadence profile。
4. **runtime artifact**：browser binary + `artifacts.json` + `metadata/build.json`，并明确 `patch_train`、channel（alpha/beta/stable）与启动参数。

**校验范围（必须全部通过）**

1. **Automation Signals**：
   - `navigator.webdriver` / automation flags / CDP artifacts 是否符合 baseline。
   - 至少 2 个检测站无 automation/driver/CDP 异常提示。
2. **Input Behavior**：
   - event trace 与 expected profile 的 cadence 分布差异在阈值内。
   - 跨 3 次采样稳定率 ≥ 95%，且无固定间隔、零抖动等自动化特征。

**验证流程（expected vs actual）**

1. 使用与 `BROW-015` 一致的运行参数启动 runtime。
2. 执行 automation signals 采样与检测站校验，产出 diff。
3. `BROW-028` 启动标准化输入脚本，连续采样 3 次 event trace。
4. 汇总 automation signals 与 input behavior 的 diff，生成 gate verdict。

**输出工件（对齐 BROW-015/BROW-016 结构）**

1. `automation-validation-report.json`：版本、channel、patch_train、采样时间、gate verdict。
2. `automation-signal-diff.json`：automation signals diff 明细。
3. `input-trace/`：3 次采样 event trace + profile 参数快照。
4. `automation-snapshots/`：expected/actual baseline 备份，供 BROW-016 做版本回归对比。
5. `runner.log`：启动、采样、diff 过程日志。

**门禁与通道影响（FR-007 对齐）**

1. **Pass**：automation signals 与 input behavior 全部满足基线；允许进入 beta/stable 发布门禁与 `BROW-016` 回归对比。
2. **Fail**：任一项 mismatch、采样失败或检测站异常提示即判失败；必须阻断：
   - release channel promotion（beta/stable）
   - `BROW-029` app runtime cutover
3. Fail 仅允许留在 alpha 做问题排查，不可进入稳定通道。

**失败处理**

- 必须输出失败原因、复现参数与对应 diff。
- 缺失 expected baseline 或输入 profile 时直接判失败，禁止默认兜底。
- 失败结果需回写运行时诊断与发布门禁记录，作为 FR-007 闭环证据。

---

## 5. 构建系统设计

### 5.1 构建工具链

| 组件 | 作用 |
|------|------|
| `depot_tools` | Chromium 源码拉取与工具链 |
| `gclient` / `fetch` | 源码同步 |
| `gn` | 构建配置生成 |
| `ninja` | 编译 |
| 平台签名工具 | macOS 签名/notarization、Windows 签名、Linux 包装 |

### 5.2 构建目标矩阵

| 平台 | 架构 | 产物 |
|------|------|------|
| macOS | arm64 | `.app` |
| macOS | amd64 | `.app` |
| Windows | amd64 | `.exe` + runtime files |
| Linux | amd64 | binary + package layout |

### 5.3 产物目录规范

构建产物目录必须与 **BROW-006** 的运行时发现与通道布局保持一致，browser-core 内部输出与运行时消费之间只做**路径映射**，不改变契约。

```text
artifacts/
  <browser-version>/
    channels/
      stable/
        artifacts.json
        metadata/
          build.json
          checksums.json
        darwin-arm64/
          Chromium.app/...
        darwin-amd64/
          Chromium.app/...
        windows-x64/
          chrome.exe
        linux-x64/
          chrome
```

### 5.4 构建输出元数据

每次发布必须输出（与 `metadata/build.json`/`checksums.json` 对齐）：

1. `browser_version`
2. `chromium_milestone`
3. `patch_train`
4. `git_revision`
5. `sha256`
6. `build_timestamp`
7. `signing_status`

---

## 6. 运行时集成契约

### 6.1 当前仓库负责

1. 指纹生成
2. 代理绑定
3. 实例生命周期管理
4. 浏览器产物发现
5. 运行时诊断与实际指纹采样

### 6.2 浏览器核心负责

1. 真正的 stealth patch 生效
2. 统一浏览器内核行为
3. 接收 deterministic runtime flags
4. 产出可签名、可分发浏览器制品

### 6.3 Browser Runtime Contract

浏览器产物必须满足以下接口契约：

| 能力 | 要求 |
|------|------|
| 启动方式 | 支持命令行启动 |
| CDP | 暴露稳定 CDP 入口 |
| User Data Dir | 支持显式指定 |
| Proxy | 支持显式代理输入 |
| Fingerprint Flags | 支持 deterministic runtime knobs |
| Exit Code | 正常退出与异常退出可区分 |

---

## 7. 发布与回滚设计

### 7.1 发布通道

| 通道 | 用途 |
|------|------|
| alpha | patch 开发验证 |
| beta | 检测站与业务灰度 |
| stable | 正式生产 |

### 7.2 发布门禁

浏览器进入 stable 前，必须通过：

1. 编译成功
2. 启动冒烟通过
3. CDP 冒烟通过
4. 指纹一致性检查通过
5. 检测站基线通过
6. 与上一稳定版本对比无关键退化

### 7.3 回滚策略

1. 保留最近两个稳定产物
2. manifest 支持固定回滚版本
3. 当前应用层允许按版本锁定浏览器产物

---

## 8. 验证体系设计

### 8.1 验证层级

| 层级 | 验证内容 |
|------|----------|
| Build Validation | 编译、打包、签名是否成功 |
| Runtime Validation | 启动、退出、CDP、user-data-dir、代理 |
| Fingerprint Validation | expected vs actual 是否一致 |
| Detection Validation | 检测站、业务目标站表现 |
| Regression Validation | 与上一版本对比退化检查 |

### 8.2 必须纳入门禁的指标

1. 指纹字段覆盖率
2. 指纹一致性差异数
3. 浏览器启动成功率
4. 人机行为链路稳定性
5. 检测站通过率

---

## 9. 实施顺序决策

### 第一批必须先做

1. Chromium 基线工作区
2. 最小可编译浏览器产物
3. 产物 manifest
4. 运行时接入与冒烟

### 第二批跟进

1. Identity Patch
2. Rendering Patch
3. Automation Patch

### 第三批跟进

1. Network Patch
2. Behavior Patch
3. 签名、发布、灰度、回滚

---

## 10. 结论

浏览器专项设计的关键，不是“把一堆 patch 塞进 Chromium”，而是建立一套 **可演进、可验证、可发布、可回滚** 的浏览器产品工程体系。  
当前仓库已经具备运行时接入层，后续浏览器核心工作必须围绕这份设计推进，而不能重新回到“依赖外部 stealth 二进制”的路线。
