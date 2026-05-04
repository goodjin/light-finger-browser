# 浏览器源码构建改进方案

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | Light Finger Browser |
| **版本** | v2.1 |
| **对应 PRD** | `docs/v1/01-prd.md` v2.0 |
| **对应架构** | `docs/v1/02-architecture/01-browser-build-runtime.md` |
| **更新日期** | 2026-04-30 |

---

## 目标

围绕 PRD v2.0，把浏览器侧从“外部二进制集成”推进到“Chromium 基线 + 自研 stealth 浏览器 + 全平台产物 + 统一运行时发现”。

---

## 范围

### In Scope

1. 浏览器产物清单与平台映射。
2. 打包后应用内嵌浏览器的运行时发现。
3. Chromium 基线、stealth patch overlay 与源码构建链所需的架构拆分与阶段计划。

### Out of Scope

1. 本轮直接接入完整 Chromium 源码 fork。
2. 本轮完成全平台真实编译与签名。
3. 与当前浏览器产品目标无关的上层业务扩展。

---

## 分阶段方案

| 阶段 | 目标 | 结果 |
|------|------|------|
| P1 | 运行时承接浏览器产物 | manifest + bundled path discovery |
| P2 | 标准化浏览器产物布局 | 统一目录结构、版本命名、校验信息 |
| P3 | 接入源码构建链 | Chromium source fork、stealth patch overlay、跨平台构建 |
| P4 | 发布流水线闭环 | 签名、校验、发版、回滚 |

---

## 当前状态对齐

| 对齐项 | 当前状态 | 对应需求 |
|--------|---------|---------|
| 浏览器运行时发现 | 已落地 | FR-004 |
| 浏览器产物清单基础契约 | 已落地 | FR-005 |
| Chromium 源码基线与 patch overlay | 已设计，待执行 | FR-006 |
| 指纹诊断与验证闭环 | 已有采样与覆盖报告，但未纳入发布门禁 | FR-007 |
| 构建、签名、发布、回滚 | 已规划，待执行 | FR-008 |

---

## 已完成承接项

### T-01: 浏览器产物清单

- 新增 `resources/cloakbrowser/artifacts.json`
- 描述 macOS / Windows / Linux 的浏览器入口路径

### T-02: 运行时路径发现

- `browser_runtime.go` 新增解析顺序：
  1. env override
  2. artifact manifest
  3. bundled path fallback

### T-03: 架构文档刷新

- 新增 `docs/v1/02-architecture/01-browser-build-runtime.md`
- 更新覆盖报告，明确 PRD v2.0 新增目标的当前状态

---

## 下一批执行项

当前专项从原子任务 ready 集启动，优先推进：

1. `BROW-001`：browser-core 工作区规范
2. `BROW-002`：构建机准入要求
3. `BROW-006`：产物输出布局与命名规则
4. `BROW-014`：检测基线与 release gate 规范

> 以上四项完成后，再继续 `BROW-003`、`BROW-004`、`BROW-005`、`BROW-007`，进入真实源码拉取和最小编译链路。

---

## BROW-002 构建机准入要求（Build Host Requirements）

为满足 PRD v2.0 的“自建 Chromium-based stealth 浏览器”目标，本节定义**构建机基线**，作为拉取 Chromium 与最小编译链路的准入条件。

> 重要说明：构建机仅服务于 browser-core 源码拉取与编译，**与本仓库运行时无关**；当前应用运行仍以已打包浏览器产物与运行时发现为主。

### 1) OS 与硬件基线（推荐）

| 平台 | OS 基线 | 推荐 CPU / RAM | 推荐可用磁盘 |
|------|---------|----------------|--------------|
| macOS | macOS 13+（Ventura），支持 Intel / Apple Silicon | 16 核 / 32-64GB | 400GB+ |
| Windows | Windows 11 Pro 或 Windows 10 22H2 | 16 核 / 32-64GB | 500GB+ |
| Linux | Ubuntu 22.04 LTS 或 24.04 LTS | 16 核 / 32-64GB | 400GB+ |

最低可运行门槛（仅用于单平台最小编译验证）：
- CPU ≥ 8 核，RAM ≥ 16GB
- **可用磁盘 ≥ 250GB**（仅够一次 checkout + 单次 build）

### 2) 必需工具链组件

通用要求：
- `depot_tools`（含 `fetch` / `gclient` / `gn` / `ninja`）
- Git（支持大仓库与 LFS 依赖）
- Python 3（Chromium 工具链依赖）

平台要求：
- **macOS**：Xcode + Command Line Tools（含 clang / lld）
- **Windows**：Visual Studio 2022（Desktop C++）+ Windows 10/11 SDK
- **Linux**：clang / lld、build-essential、pkg-config 等基础构建依赖

### 3) 磁盘阈值（checkout / build）

- **拉取前最低可用空间**：≥ 200GB  
  （Chromium checkout + third_party 通常 100GB+，需要预留同步空间）
- **构建前稳定可用空间**：≥ 300GB  
  （构建产物 + 符号文件 + 中间产物通常额外 100-200GB）
- **推荐稳定配置**：400-500GB 可用磁盘，便于多次迭代与产物保留

### 4) 缓存策略（sccache / ccache）

- 建议为每台构建机启用 **sccache / ccache**，并预留 50-200GB 本地缓存。
- 原因：Chromium 构建耗时高，patch 迭代频繁，缓存可显著减少全量重编译时间，提升 PRD v2.0 的迭代速度与回归效率。
- 缓存与源码目录解耦，避免污染仓库与产物目录；必要时按 milestone 清理旧缓存。

---

## BROW-003 depot_tools bootstrap

为进入 Chromium 拉取与最小编译阶段，先固化 `depot_tools` 的安装、PATH 与验证流程。**必须在独立 browser-core 工作区执行**，不要在本仓库内初始化。

### 1) 安装位置（推荐）

- 建议工作区根目录：`~/browser-core/`
- `depot_tools` 放置路径：`~/browser-core/depot_tools/`
- Windows 示例：`C:\browser-core\depot_tools\`

### 2) 获取 depot_tools

```bash
mkdir -p ~/browser-core
cd ~/browser-core
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
```

### 3) PATH 约定（确保优先级）

- **macOS / Linux（bash/zsh）**
  ```bash
  export PATH="$HOME/browser-core/depot_tools:$PATH"
  ```
  建议写入 `~/.zshrc` 或 `~/.bashrc`，并确保 `depot_tools` 在 PATH 前段。
- **Windows（PowerShell）**
  ```powershell
  $env:Path = "C:\browser-core\depot_tools;$env:Path"
  ```
  持久化可使用系统环境变量或 `setx PATH`。

### 4) 最小验证（只验证工具链可用）

```bash
gclient --version
gn --version
ninja --version
```

### 5) OS 备注

- **macOS**：确保已安装 Xcode + Command Line Tools，否则 `gn`/`ninja` 编译链不可用。
- **Windows**：确保 VS2022 Desktop C++ 与 Windows SDK 完成安装。
- **Linux**：确保 clang / lld 与基础构建依赖已满足。

> 注意：不要在本仓库执行 `fetch` / `gclient sync`，请在 browser-core 工作区中完成后续步骤。

---

## BROW-005 source fetch bootstrap

在 `depot_tools` 就绪后，为 Chromium 源码拉取与同步提供可复用的流程说明。**仅在 browser-core 工作区执行**，不要在本仓库内运行。

### 1) 执行位置（browser-core 工作区）

- 推荐工作区根目录：`~/browser-core/`
- 仅在该工作区内执行 `fetch` / `gclient`，**禁止在本仓库目录下执行**

### 2) 拉取与同步（示例）

```bash
cd ~/browser-core
fetch chromium
cd chromium
gclient sync -D -r <PINNED_REVISION>
```

### 3) pinned revision 说明（关联 BROW-004）

- `<PINNED_REVISION>` 来自 `BROW-004 chromium-baseline-pin` 的锁定结果。
- 必须使用 pin 结果同步，避免直接跟随 upstream HEAD。

### 4) 期望输出与最小验证

- 期望生成 `~/browser-core/chromium/src/` 源码树（或同等路径的 `src/` 目录）。
- 最小验证：确认 `src/` 目录存在且包含 `DEPS`、`BUILD.gn` 等 Chromium 基线文件。

---

## BROW-007 minimal build args

在 `BROW-004` baseline pin 与 `BROW-005` 源码拉取完成后，先固化**首个平台最小编译参数**，作为 `BROW-008` 单平台 build 的唯一入口基线。

### 1) 基线 `gn args` 假设

> 仅定义最小集，避免一次性引入过多开关；所有变更需遵守下方 guardrails。

```gn
target_os = "mac"        # 首个平台，按实际构建机选择 mac/win/linux
target_cpu = "arm64"     # 首个平台架构，常见为 arm64/x64
is_debug = true          # 首次最小编译优先 debug
is_official_build = false
proprietary_codecs = false
ffmpeg_branding = "Chromium"
```

- `target_os/target_cpu` 必须与首个平台一致，**不可跨平台混用**。
- `is_debug/is_official_build`（即 is_official）只允许在 baseline 固定后再升级为 release/official 方案。
- `proprietary_codecs` 为可选开关，默认保持关闭，避免引入专有依赖。

### 2) 最小编译目标与输出目录

- **最小目标**：`chrome`（Chromium 桌面壳体，满足可启动与 CDP 验证）
- **输出目录约定**：`out/minimal`（示例：`~/browser-core/chromium/src/out/minimal`）
- 后续若 `BROW-006` 固化了统一目录布局，以其为准，但必须保留 `minimal` 基线标识。

### 3) 验证预期（不执行）

- 产物可启动（命令行可拉起窗口或 headless）。
- 可暴露 CDP 端口（例如 `--remote-debugging-port`），可被 runtime/脚本连接。
- 可指定 `--user-data-dir` 且不报错。

### 4) Guardrails（与 baseline pin 绑定）

- 任何 `gn args` 变更必须同步更新 `BROW-004` baseline pin 记录与变更说明。
- 禁止在未更新 pin 的情况下切换 `target_os/target_cpu` 或 `is_debug`/`is_official_build`。
- 开启 `proprietary_codecs` 时需在产物元数据与 manifest 中显式标记。

---

## BROW-008 single-platform build smoke

在 `BROW-007` 基线参数就绪后，执行**单平台最小编译冒烟**，验证能产出可启动浏览器并暴露 CDP。仅记录流程与验收，不在当前仓库内执行真实编译。

### 1) 前置条件（Prerequisites）

1. `BROW-004` 已固定 `pinned revision`，且 `gclient sync -D -r <PINNED_REVISION>` 已完成。
2. `BROW-003` 已完成 `depot_tools` 安装，并确保 `gn`/`ninja` 可用。
3. `BROW-005` 已完成 Chromium 源码拉取，存在 `~/browser-core/chromium/src/` 且包含 `DEPS`、`BUILD.gn`。
4. 构建机满足 `BROW-002` 基线要求，构建前**稳定可用磁盘 ≥ 300GB**（推荐 400GB+）。
5. `BROW-007` 的 `gn args` 基线已确定，且 `target_os/target_cpu` 与构建机一致。

### 2) 最小构建命令（单平台）

> 全程在 `browser-core` 工作区执行，禁止在本仓库目录内执行。

```bash
cd ~/browser-core/chromium/src
gn gen out/minimal --args='target_os="mac" target_cpu="arm64" is_debug=true is_official_build=false proprietary_codecs=false ffmpeg_branding="Chromium"'
ninja -C out/minimal chrome
```

- `target_os/target_cpu` 根据当前构建平台调整（mac/win/linux，arm64/x64）。
- 若需修改参数，统一通过 `gn args out/minimal` 维护，保持与 `BROW-007` 对齐。

### 3) 期望输出

- `out/minimal/` 目录生成并包含产物：
  - macOS：`out/minimal/Chrome.app`
  - Linux：`out/minimal/chrome`
  - Windows：`out/minimal/chrome.exe`
- `out/minimal/args.gn` 与 `build.ninja` 生成，代表基线参数已固化。

### 4) 最小运行验证（冒烟）

1. 启动并暴露 CDP（示例）：
   - macOS：
     ```bash
     ./out/minimal/Chrome.app/Contents/MacOS/Chromium \
       --remote-debugging-port=9222 \
       --user-data-dir=$HOME/browser-core/chromium/src/out/minimal/profile
     ```
   - Linux：
     ```bash
     ./out/minimal/chrome --remote-debugging-port=9222 \
       --user-data-dir=$HOME/browser-core/chromium/src/out/minimal/profile
     ```
   - Windows（PowerShell）：
     ```powershell
     .\out\minimal\chrome.exe --remote-debugging-port=9222 `
       --user-data-dir=$env:USERPROFILE\browser-core\chromium\src\out\minimal\profile
     ```
2. 验证 CDP 可达：
   ```bash
   curl http://127.0.0.1:9222/json/version
   ```
   期望返回 JSON（含 `Browser` 与 `webSocketDebuggerUrl`）。
3. 验证可正常退出且 `user-data-dir` 无报错写入。

### 5) 当前环境阻塞说明

- 本仓库内**不包含** `browser-core` 工作区与 Chromium 源码树；若本机构建环境未单独准备 `~/browser-core/chromium/src/` 与 `depot_tools`，则无法执行 BROW-008 的真实编译。
- 可用磁盘、工具链版本、以及是否满足 `BROW-002` 基线**未在当前环境验证**；需在独立构建机上完成核验后再执行冒烟构建。

---

## BROW-009 build metadata emitter

为构建产物输出稳定可验证的元数据契约，作为 manifest 校验、版本追踪与发布门禁的唯一依据。**仅定义输出字段与流转规则，不涉及脚本实现。**

### 1) 输出文件与位置（保持 BROW-006 布局一致）

```text
artifacts/<browser-version>/channels/<channel>/
  artifacts.json
  metadata/
    build.json
    checksums.json
```

- `artifacts.json`：通道级产物清单（运行时读取入口）。
- `metadata/build.json`：构建元数据主文件。
- `metadata/checksums.json`：产物校验清单，与 build.json 的 `checksums` 字段一致。

### 2) 必填字段（build.json 必须完整输出）

1. `browser_version`
2. `chromium_milestone`
3. `patch_train`
4. `git_revision`
5. `build_timestamp`（UTC / ISO-8601）
6. `signing_status`（例如：unsigned/signed/notarized）
7. `checksums`（指向 `metadata/checksums.json` 的内容或引用）

`metadata/checksums.json` 至少包含：
- `algorithm: "sha256"`
- `files`：相对 `channels/<channel>/` 的文件路径 → sha256

### 3) 产出时机（build flow 约束）

1. BROW-008 编译完成并落盘到 `channels/<channel>/<platform>/` 后生成校验值。
2. 签名/公证完成后更新 `signing_status` 与 `checksums.json`。
3. 输出 `metadata/build.json`，并同步更新同通道的 `artifacts.json`。
4. 打包时将 `artifacts.json` + `metadata/` 映射到运行时可读取路径（例如 `resources/cloakbrowser/`）。

### 4) Guardrails（与 manifest/runtime contract 同步）

- `artifacts.json`、`metadata/build.json`、`metadata/checksums.json` 必须使用**同一** `browser_version` / `chromium_milestone` / `patch_train` / `git_revision`。
- 任一字段变更需**整体重写**三份文件，禁止手工局部改写导致版本不一致。
- 运行时仅依赖 manifest 与 metadata，不直接引用 browser-core 构建目录；若映射位置变化，需同步更新运行时契约文档。

---

## BROW-010 runtime smoke runner

在 `BROW-008` 产出可启动产物后，定义运行时冒烟验证流程，形成发布前的最小 gate。该流程需满足 FR-007 的诊断/验证闭环原则，并对齐 Rule-005 的运行时校验要求。

### 1) 输入与触发条件

1. 已完成 `BROW-008` 单平台最小 build 冒烟。
2. 使用 `BROW-006` 约定的产物目录与命名，且具备 `BROW-009` 产物元数据。

### 2) 核心检查项（必须全部通过）

1. **启动 / 退出验证**
   - 可在 headless 或窗口模式启动。
   - 启动后 30 秒内必须能稳定运行并可主动退出。
2. **CDP 可用性**
   - 使用 `--remote-debugging-port` 启动后，`/json/version` 返回 `webSocketDebuggerUrl`。
   - CDP 连接建立后可创建最小页面会话。
3. **user-data-dir 隔离**
   - 指定 `--user-data-dir` 到独立目录。
   - 目录内需生成 Profile/Cache 等基础数据，且退出后可完整清理。
4. **基础指纹快照**
   - 采样与 FR-007 对齐的基础字段（platform/locale/timezone/screen/userAgent）。
   - 输出与现有诊断对齐的快照（参考 [架构覆盖报告](../02-architecture/00-coverage-report.md) 的指纹诊断入口）。

### 3) 输出产物（用于门禁与追溯）

1. **运行日志**：启动、CDP 连接、退出全链路日志。
2. **状态报告**：包含版本、revision、运行时参数、验证项结果、耗时。
3. **指纹快照**：按字段输出当前值与采样时间。

### 4) 失败处理（Gate Fail）

1. 任一检查项失败即标记 gate failed。
2. 必须输出失败原因、复现参数与失败时的日志片段。
3. gate failed 产物禁止进入后续 `BROW-015`/`BROW-016` 验证链路与发布流水线。

---

## BROW-015 fingerprint baseline runner

在 `BROW-010` 运行时冒烟通过后，建立**expected vs actual 指纹基线校验**流程，满足 FR-007 的诊断闭环要求，并以 `BROW-014` 的检测站与采样字段为唯一规范来源。

### 1) 输入与触发条件

1. `BROW-010` runtime smoke runner 已通过。
2. `BROW-014` 已输出检测站清单、采样字段与允许差异阈值。
3. 指定**运行时产物**（browser binary + `artifacts.json` + `metadata/build.json`）。
4. 指定**通道**（alpha/beta/stable）与版本锁定信息。
5. 指定**expected profile**（由 BROW-014 产出的基线指纹快照/字段期望值）。

### 2) 采样与对比流程

1. 使用与 `BROW-010` 相同的 runtime 启动参数（含 `--remote-debugging-port` 与 `--user-data-dir`）。
2. 调用现有指纹诊断入口采样，覆盖 BROW-014 定义的字段集合（platform/locale/timezone/screen/GPU/automation 等）。
3. 对每个检测站生成 **actual snapshot**，并与 expected profile 做字段级 diff：
   - match / variance / violation 三种等级
   - 记录差异值、允许阈值与采样来源
4. 输出汇总 diff report（字段差异、站点差异、总体评分）。

### 3) 输出产物（用于门禁与追溯）

1. **baseline-report.json**：版本、通道、revision、采样时间、结果汇总。
2. **baseline-diff.json**：expected vs actual 对比结果与差异明细。
3. **fingerprint-snapshots/**：与诊断入口一致的 raw snapshot 备份。
4. **runner.log**：启动、采样、diff 过程日志。

### 4) Pass/Fail 与发布门禁

1. **Pass**：所有必需字段满足 BROW-014 阈值，且无 critical 级别 violation。
2. **Fail**：任一检测站缺失、关键字段 violation、或采样失败即判失败。
3. Fail 结果必须阻断：
   - `BROW-016` regression diff runner
   - release channel promotion / 发布流水线

---

## BROW-016 regression diff runner

在 `BROW-015` 基线校验通过后，定义**相邻浏览器版本的回归对比**流程，确保版本升级不会引入指纹/行为退化，并满足 FR-007 的发布门禁要求。

### 1) 输入与触发条件

1. `BROW-015` 已对比完成，且 **current** 与 **previous stable** 均通过基线校验。
2. 输入两套 **BROW-015 产物**（分别来自 current 与 previous stable）：
   - `baseline-report.json`
   - `baseline-diff.json`
   - `fingerprint-snapshots/`
   - `runner.log`
3. 指定通道与版本：
   - current：待发布版本（alpha/beta/stable 任一通道）
   - previous stable：上一稳定版本（同平台、同检测站集合）

### 2) Diff 流程（field-level / site-level）

1. **字段级 diff（field-level）**
   - 对齐 BROW-014 字段集合，逐字段对比 current vs previous stable 的差异幅度。
   - 输出：字段级 regression 标记（improve / neutral / regression / critical）。
2. **站点级 diff（site-level）**
   - 以检测站为单位汇总字段结果，计算每站点的退化比例。
   - 输出：站点级 regression 评分与 P0/P1/P2 分类。
3. **汇总评分**
   - 统计整体 regression rate（退化字段 / 总字段）。
   - 标记新增 P0（previous stable 无、current 新增）。

### 3) 输出产物（对齐 BROW-015 结果链路）

1. **regression-diff.json**：字段级 + 站点级 diff 明细、回归比例与新旧版本对齐信息。
2. **regression-summary.json**：聚合评分、P0/P1/P2 统计、最终 gate verdict。
3. **runner.log**：diff 过程日志（与 BROW-015 runner.log 风格保持一致）。

### 4) Pass/Fail 与发布门禁（FR-007）

1. **Pass**：
   - overall regression rate **≤ 2%**
   - **no new P0** regression
2. **Fail**：
   - regression rate > 2%，或出现任一新 P0
3. Fail 结果必须阻断：
   - release channel promotion / 发布流水线
   - `BROW-031` release-channel-promotion

---

## BROW-030 signing checksum pipeline

建立签名与校验的产物流转契约，确保三平台产物在进入发布门禁前完成签名、校验与元数据更新。**仅定义流程与输出，不包含 CI 脚本或执行细节**。需与 `BROW-009` 元数据契约与 Rule-005 发布门禁对齐。

### 1) 输入与前置条件

1. 已完成 `BROW-008` 构建，产物落盘并满足 `BROW-006` 目录布局。
2. `BROW-009` 已输出 `metadata/build.json` 与 `metadata/checksums.json`（初始为 unsigned）。
3. 已指定通道与版本（alpha/beta/stable）以及对应产物清单 `artifacts.json`。

### 2) 平台签名阶段（Signing Stages）

1. **macOS（notarization）**
   - `codesign` → notarization 提交 → notarization 通过后 `staple`。
   - `signing_status` 必须更新为 `notarized`，并记录签名完成时间。
2. **Windows（code signing）**
   - 对 `exe/msi`（或 zip 内的二进制）执行签名与时间戳。
   - 通过 `signtool verify` 后，`signing_status` 更新为 `signed`。
3. **Linux（packaging）**
   - 产出标准包（tar.gz / deb / rpm 之一或多种），按 `BROW-006` 命名。
   - 若执行包签名（如 detached signature），`signing_status` 置为 `signed`；未签名则保持 `unsigned`，并不得进入 stable。

### 3) Checksum 生成与校验

1. **生成时机**：平台签名完成后，对最终产物重新生成 sha256。
2. **一致性校验**：
   - `metadata/checksums.json` 的 `files` 必须与 `channels/<channel>/` 实际文件完全一致。
   - `build.json` 内 `checksums` 与 `metadata/checksums.json` 内容一致（或引用一致）。
3. **校验步骤**：对 `checksums.json` 中所有文件重新计算 sha256，逐一比对；任一不一致即判失败。

### 4) 输出与元数据更新（对齐 BROW-009）

1. 更新 `metadata/build.json`：
   - `signing_status`（unsigned/signed/notarized）
   - `build_timestamp` 保持原始构建时间，另补充签名完成时间字段（若已有约定则复用）
2. 重新生成 `metadata/checksums.json`（以签名后产物为准）。
3. 若签名或打包导致文件路径变化，必须同步更新 `artifacts.json` 并保持与 `build.json`/`checksums.json` 版本字段一致。

### 5) Gate Fail 处理（Rule-005）

1. 任一平台签名失败、checksum 校验失败或元数据不一致，立即标记 gate failed。
2. gate failed 产物不得进入 `BROW-031` release promotion，也不得进入 stable。
3. 必须输出失败原因、失败步骤与日志片段，供回滚与重试使用。

---

## 后续执行项

1. 增加带版本号与校验信息的产物元数据。
2. 设计 Chromium 源码 fork 与 stealth patch overlay 管理方式。
3. 增加跨平台构建脚本与 CI 任务。
4. 增加桌面包与服务端共享的浏览器产物发布规范。

## 当前阻塞与前置条件

1. 真实 Chromium 拉取/编译前，必须先产出 browser-core 工作区和构建机基线规范。
2. 构建、签名、发布阶段依赖独立 build host 与稳定的产物输出布局。
3. FR-007 当前仍是“部分落地”，后续需要把采样、差异分析与发布门禁连成闭环。

---

> 详细分阶段执行计划见：`10-browser-core-program.md`
