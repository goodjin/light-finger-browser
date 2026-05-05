# 浏览器源码构建指南

## 概述

本文档描述如何构建自建 stealth 浏览器，包括环境准备、源码获取、patch 应用、编译和产物打包。

## 前置要求

### 硬件要求

- CPU: 16核+ (推荐 32核)
- 内存: 32GB+ (推荐 64GB)
- 磁盘: 200GB+ SSD
- 网络: 稳定的互联网连接（需访问 Google 源码仓库）

### 软件依赖

| 工具 | 版本 | 用途 |
|------|------|------|
| Git | 2.0+ | 源码管理 |
| Python | 3.8+ | 构建脚本 |
| depot_tools | latest | Chromium 源码拉取 |
| ninja | 4.0+ | 编译工具 |
| gn | latest | 生成 Ninja 构建文件 |

### macOS 额外依赖

```bash
# 安装 Xcode Command Line Tools
xcode-select --install

# 安装 Homebrew (如果没有)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 安装构建依赖
brew install ninja gn git python3
```

### Linux 额外依赖

```bash
# Debian/Ubuntu
sudo apt-get install build-essential ninja-build python3 git \
    libnss3-dev libatk1.0-dev libatk-bridge2.0-dev libcups2-dev \
    libdrm-dev libxkbcommon-dev libxcomposite-dev libxdamage-dev \
    libxrandr-dev libgbm-dev libasound2-dev libpango-1.0-0

# Fedora/RHEL
sudo dnf install @development-tools ninja-build python3 git \
    nss-devel atk-devel cups-devel libdrm-devel mesa-libgbm-devel \
    alsa-lib-devel pango-devel
```

### Windows 额外依赖

- Visual Studio 2022 Community+ with "Desktop development with C++"
- Windows 10 SDK (10.0.19041.0+)
- depot_tools (添加到 PATH)

---

## 一键构建

在 `browser-core/` 目录下执行：

```bash
cd browser-core

# 给脚本加执行权限（首次）
chmod +x build.sh scripts/*.sh

# 构建当前平台
./build.sh

# 构建特定平台
./build.sh darwin-arm64
./build.sh windows-amd64
./build.sh linux-amd64

# 构建所有平台
./build.sh all
```

构建产物输出到 `browser-core/artifacts/<platform>/`。

---

## 手动构建步骤

如果一键构建脚本失败，可按以下步骤手动执行。

### 步骤 1: 拉取源码

```bash
# 创建工作目录
mkdir -p browser-core/src
cd browser-core/src

# 克隆 depot_tools
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git

# 添加到 PATH
export PATH="$PWD/depot_tools:$PATH"

# 初始化 repo
repo init -u https://chromium.googlesource.com/chromium/src.git --depth 1

# 同步源码（首次可能需要 1-2 小时）
repo sync -c -j$(nproc)
```

### 步骤 2: 记录版本

```bash
# 记录当前 revision
git rev-parse HEAD > ../chromium_revision
echo "chromium_milestone=$(git describe --tags | cut -d. -f1)" > ../chromium_milestone
```

### 步骤 3: 应用 Stealth Patches

```bash
cd ..
./scripts/apply_patches.sh \
    --src ./src \
    --overlay ./patches \
    --train 1
```

### 步骤 4: 配置构建

```bash
cd src
export PATH="$PWD/../depot_tools:$PATH"

# 生成 release 构建配置
gn gen ../out/release --args='
is_debug=false
is_official_build=true
target_cpu="x64"
target_os="mac"
mac_sdk_min=10.15
'
```

### 步骤 5: 编译

```bash
# 编译 Chrome（需要 30-60 分钟）
ninja -C ../out/release chrome
```

### 步骤 6: 打包

```bash
# 创建产物目录
mkdir -p ../artifacts/darwin-arm64

# macOS
cp -R out/release/Chromium.app ../artifacts/darwin-arm64/

# Linux
cp out/release/chrome ../artifacts/linux-amd64/chromium

# Windows
cp out/release/chrome.exe ../artifacts/windows-amd64/chromium.exe
```

---

## 产物说明

### 目录结构

```
browser-core/
├── artifacts/
│   ├── darwin-arm64/
│   │   ├── Chromium.app/
│   │   └── artifacts.json
│   ├── darwin-amd64/
│   ├── linux-amd64/
│   └── windows-amd64/
├── out/                  # 构建中间产物（可删除）
│   └── release/
└── src/                  # Chromium 源码（保留）
```

### artifacts.json

每个平台的 `artifacts.json` 包含：

```json
{
  "version": 2,
  "channel": "stable",
  "browser_version": "lfb-browser/136.1.0",
  "chromium_revision": "a12f2a5b3c7d8e9f...",
  "patch_train": 1,
  "platform": "darwin-arm64",
  "build_timestamp": "2026-05-05T12:00:00Z"
}
```

---

## patch_train 说明

`patch_train` 定义 stealth patch 的版本：

- `train = 1`：当前稳定版 patch
- 每次修改 patch 时递增 train 版本

patch 应用时，系统会自动读取 `patches/<domain>/train-N/patchset.json`。

---

## 常见问题

### Q: repo sync 失败

```bash
# 清理后重试
repo sync -c -j$(nproc) --force-sync --no-tags
```

### Q: ninja 编译 OOM

```bash
# 减少并行数
ninja -C out/release -j8 chrome
```

### Q: gn gen 找不到

```bash
# 确保 depot_tools 在 PATH 最前
export PATH="$PWD/depot_tools:$PATH"
which gn
```

### Q: macOS 构建失败权限问题

```bash
# 确认 Xcode license 已接受
sudo xcodebuild -license accept
```

---

## 与主项目集成

构建完成后，更新 `resources/browser/artifacts.json` 指向新产物：

```bash
# 复制产物到主项目
cp -r artifacts/* ../resources/browser/

# 提交更新
git add resources/browser/
git commit -m "feat: update browser artifacts to $(cat browser_version)"
```

---

## 验证构建

构建完成后，运行指纹验证：

```bash
# 启动构建的浏览器
./artifacts/darwin-arm64/Chromium.app/Contents/MacOS/Chromium \
    --user-data-dir=/tmp/test-profile \
    --headless \
    --dump-dom \
    --run-all-compositor-bailout-thresholds \
    --enable-begin-frame-control \
    --disable-frame-rate-limit \
    "data:text/html,<script>document.title=navigator.platform</script>"
```

期望输出 `document.title` 应为 `navigator.platform` 的值，即 `--platform` 参数指定的值。
