# Chromium 浏览器编译指南

> 本文档详细说明基于 Chromium 源码从头编译浏览器的方法，涵盖术语解释、原理说明和完整步骤。

## 目录

1. [概述](#概述)
2. [核心概念与术语](#核心概念与术语)
3. [编译前准备](#编译前准备)
4. [源码获取](#源码获取)
5. [构建配置](#构建配置)
6. [编译过程](#编译过程)
7. [常见问题](#常见问题)
8. [进阶话题](#进阶话题)

---

## 概述

### Chromium 架构简介

Chromium 是 Google Chrome 的开源基础，采用多进程架构：

```
┌─────────────────────────────────────────────────────────┐
│                    Browser Process                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │    Main     │  │   Renderer  │  │   GPU       │   │
│  │   Thread    │  │   Process    │  │   Process   │   │
│  └─────────────┘  └─────────────┘  └─────────────┘   │
│         │                │                │             │
│         └────────────────┼────────────────┘             │
│                          │                              │
│                   ┌──────▼──────┐                       │
│                   │   Mojo IPC  │                       │
│                   └─────────────┘                       │
└─────────────────────────────────────────────────────────┘
```

**关键组件**：

| 组件 | 仓库 | 说明 |
|------|------|------|
| **Blink** | third_party/blink | 渲染引擎 (HTML/CSS 解析) |
| **V8** | v8 | JavaScript 引擎 |
| **Skia** | third_party/skia | 2D 图形库 |
| **Mojo** | mojo | 进程间通信框架 |
| **Content** | content | 浏览器核心接口层 |

---

## 核心概念与术语

### 1. depot_tools

Google 官方的 Chromium 开发工具集，包含：

| 工具 | 用途 |
|------|------|
| `gclient` | 管理多仓库依赖 |
| `gn` | 生成 Ninja 构建文件 |
| `ninja` | 执行构建 |
| `roll-dep` | 滚依赖版本 |

```bash
# 安装 depot_tools
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
export PATH="$PWD/depot_tools:$PATH"
```

### 2. GN (Generate Ninja)

Meta-build 系统，用于生成 `build.ninja` 文件：

```bash
# GN 配置示例
gn gen out/Release --args='is_debug=false target_os="mac" target_cpu="arm64"'
```

**核心 GN 概念**：

| 概念 | 说明 |
|------|------|
| `BUILD.gn` | 构建定义文件 |
| `args.gn` | 构建参数文件 |
| `target` | 构建目标 (可执行文件、库等) |
| `deps` | 依赖的其他 targets |

### 3. Ninja

高速构建系统，读取 `build.ninja` 并执行编译：

```bash
# 单线程构建
ninja -C out/Release chrome

# 多线程构建 (推荐)
ninja -C out/Release -j$(nproc) chrome
```

### 4. gclient

Git 仓库管理工具，用于同步 DEPS 依赖：

```bash
# 配置
gclient config <repo_url>

# 同步 (下载所有 DEPS)
gclient sync

# 同步但不运行 hooks
gclient sync --nohooks
```

### 5. DEPS

`DEPS` 文件是 Python 格式的依赖定义：

```python
# src/DEPS 示例
vars = {
    'build_with_chromium': True,
    'checkout_android': False,
}

deps = {
    'src/third_party/blink':
        Var('chromium_git') + '/blink.git' + '@' + Var('blink_revision'),
    'src/v8': ...,
}
```

### 6. 子模块 (Git Submodules)

嵌套的 Git 仓库，用于管理大型第三方依赖：

```bash
# 初始化子模块
git submodule update --init

# 递归初始化所有子模块
git submodule update --init --recursive

# 查看子模块状态
git submodule status
```

### 7. Git Shallow Clone

只下载最新的 commit，减少仓库大小：

```bash
# 浅克隆 (只下载最新代码)
git clone --depth=1 https://chromium.googlesource.com/chromium/src.git

# 完全克隆 (下载所有历史)
git clone https://chromium.googlesource.com/chromium/src.git
```

### 8. GN Args (构建参数)

控制编译行为的变量：

| 参数 | 值 | 说明 |
|------|-----|------|
| `is_debug` | `false` | Release 构建 |
| `is_official_build` | `true` | 官方优化构建 |
| `is_component_build` | `false` | 静态链接 |
| `target_os` | `"mac"` | 目标操作系统 |
| `target_cpu` | `"arm64"` | 目标 CPU 架构 |
| `proprietary_codecs` | `true` | 专有编解码器 |

### 9. Chrome Content Browser Client

浏览器入口点之一，负责：

- 添加命令行开关
- 配置功能开关
- 处理进程模型

```cpp
// 添加 stealth 开关示例
void ChromeContentBrowserClient::AppendExtraCommandLineSwitches(
    CommandLine* command_line, int child_process_id) {
  static const char* const kStealthSwitches[] = {
      "platform", "locale", "timezone",
  };
  command_line->CopySwitchesFrom(browser_command_line, kStealthSwitches);
}
```

---

## 编译前准备

### 系统要求

| 项目 | macOS (Apple Silicon) | Linux (x86_64) |
|------|---------------------|-----------------|
| CPU | Apple Silicon (M1/M2/M3) | 8+ 核心 |
| 内存 | 16GB+ 推荐 | 16GB+ 推荐 |
| 磁盘 | 150GB+ SSD | 150GB+ SSD |
| Xcode | 15.0+ | - |
| Python | 3.8+ | 3.8+ |
| Git | 2.0+ | 2.0+ |

### 安装依赖

#### macOS

```bash
# 安装 Xcode Command Line Tools
xcode-select --install

# 安装 Homebrew (如果需要)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 安装必要工具
brew install python3 ninja

# 或通过 depot_tools 自带
```

#### Linux (Ubuntu/Debian)

```bash
sudo apt-get update
sudo apt-get install python3 git ninja-build clang
```

### 安装 depot_tools

```bash
# 下载 depot_tools
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git ~/depot_tools

# 添加到 PATH
echo 'export PATH="$HOME/depot_tools:$PATH"' >> ~/.zshrc
source ~/.zshrc

# 验证安装
gclient --version
```

---

## 源码获取

### 方式一：完整克隆 (推荐)

```bash
# 创建工作目录
mkdir chromium && cd chromium

# 配置 gclient
gclient config https://chromium.googlesource.com/chromium/src.git

# 同步源码 (可能需要 30-60 分钟)
gclient sync

# 或者浅克隆后选择性同步
git clone --depth=1 https://chromium.googlesource.com/chromium/src.git src
cd src
git submodule update --init --depth=1
```

### 方式二：选择性同步 (节省流量)

```bash
# 只同步 Mac 需要的组件
gclient sync --with_branch_heads --no-history

# 或设置 .gclient 限制范围
cat > .gclient << 'EOF'
solutions = [
  {
    "name": "src",
    "url": "https://chromium.googlesource.com/chromium/src.git",
    "managed": False,
    "custom_deps": {},
    "custom_vars": {
      "checkout_configuration": "minimal",
    },
  },
]
EOF

gclient sync
```

### 方式三：使用 fetch 脚本

```bash
# 如果项目提供了 fetch 脚本
./scripts/fetch_chromium.sh
```

### 验证源码

```bash
cd src

# 检查 git 状态
git status

# 确认版本
git log --oneline -1
cat chrome/VERSION
```

---

## 构建配置

### 创建构建目录

```bash
mkdir -p out/Release
```

### 配置 GN 参数

**创建 args.gn 文件**：

```bash
# src/args.gn
cat > args.gn << 'EOF'
# 基础配置
is_debug = false
is_official_build = true
is_component_build = false

# 性能优化
chrome_pgo_phase = 0
use_thin_lto = false
treat_warnings_as_errors = false

# 编解码器
proprietary_codecs = true
ffmpeg_branding = "Chrome"

# 目标平台 (macOS ARM64)
target_os = "mac"
target_cpu = "arm64"
mac_sdk_min = "10.15"

# 禁用不需要的功能
enable_nacl = false
fieldtrial_testing_like_official_build = true
EOF
```

### 生成构建文件

```bash
# 方法一：使用 args 文件
gn gen out/Release --args-file=args.gn

# 方法二：直接传递参数
gn gen out/Release --args='is_debug=false target_os="mac" target_cpu="arm64"'

# 验证生成成功
gn ls out/Release --type=executable | head
```

### GN 常用命令

```bash
# 列出所有构建目标
gn ls out/Release

# 列出特定类型目标
gn ls out/Release --type=executable
gn ls out/Release --type=shared_library

# 查看构建配置
gn args out/Release --list

# 清理构建
gn clean out/Release
```

---

## 编译过程

### 单线程编译 (调试用)

```bash
cd src
ninja -C out/Release -j1 chrome
```

### 多线程编译 (推荐)

```bash
# 使用所有 CPU 核心
ninja -C out/Release -j$(sysctl -n hw.ncpu) chrome

# 或指定核心数
ninja -C out/Release -j16 chrome
```

### 编译特定目标

```bash
# 只编译 chrome (浏览器)
ninja -C out/Release chrome

# 编译 chrome + 安装包
ninja -C out/Release chrome chrome/installer/linux/package.desktop

# 只编译 base 库
ninja -C out/Release base
```

### 监控编译进度

```bash
# 实时查看编译输出
ninja -C out/Release -j16 chrome 2>&1 | tee build.log

# 后台编译并监控
ninja -C out/Release -j16 chrome &
watch -n 1 'ls -la out/Release/*.a 2>/dev/null | wc -l'
```

### 编译完成验证

```bash
# 检查产出物
ls -la out/Release/Chromium.app/Contents/MacOS/Chromium

# 检查版本
./out/Release/Chromium.app/Contents/MacOS/Chromium --version

# 检查二进制大小
du -sh out/Release/Chromium.app
```

---

## 常见问题

### Q1: gclient sync 失败，提示需要认证

```bash
# 确保已登录 Google 账号
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git

# 尝试重新认证
gclient config https://chromium.googlesource.com/chromium/src.git
gclient sync --force
```

### Q2: 编译错误 "missing .gni file"

```bash
# 子模块未初始化
git submodule update --init

# 或重新同步
gclient sync --with_branch_heads
```

### Q3: 磁盘空间不足

```bash
# 清理构建产物
rm -rf out/Release

# 清理 git 历史 (如果之前 unshallowed)
git repack -ad

# 清理旧构建
ninja -C out/Release -t clean
```

### Q4: GN gen 失败 "Unable to load"

通常是缺少 gclient sync 生成的配置文件：

```bash
# 确保运行过完整 gclient sync
gclient sync --nohooks
```

### Q5: 编译超时/网络问题

```bash
# 使用国内镜像 (如果有)
export GIT_REPO_CACHE=/path/to/cache

# 或设置代理
export https_proxy=http://127.0.0.1:7890
export http_proxy=http://127.0.0.1:7890
```

---

## 进阶话题

### 1. 添加自定义补丁

```bash
# 创建补丁
git diff src/third_party/blink/renderer/core/frame/navigator.cc > my_patch.patch

# 应用补丁
git apply my_patch.patch

# 验证应用成功
git diff --stat
```

### 2. 交叉编译

```bash
# 在 Linux 上交叉编译 Windows 版本
gn gen out/Release-win --args='target_os="win" target_cpu="x64"'
ninja -C out/Release-win chrome.exe
```

### 3. Component Build (快速开发)

```bash
# 组件构建 - 编译更快，但二进制更大
gn gen out/Debug --args='is_debug=true is_component_build=true'
ninja -C out/Debug chrome
```

### 4. 构建变体

| 构建类型 | 用途 | 特点 |
|----------|------|------|
| Debug | 开发调试 | 包含调试符号，编译快 |
| Release | 生产环境 | 优化编译，体积小 |
| Official | 内部发布 | 最高优化级别 |

### 5. 理解构建依赖图

```bash
# 查看目标依赖
gn desc out/Release //chrome:chrome

# 查看所有依赖
ninja -C out/Release -t rules | head
```

---

## 参考资源

- [Chromium Build Documentation](https://www.chromium.org/developers/how-tos/build-instructions/)
- [GN Quick Start](https://gn.googlesource.com/gn/+/main/docs/quick_start.md)
- [depot_tools Tutorial](https://commondatastorage.googleapis.com/chromium-closer-logs.appspot.com/data/crew/2017/05/11/build174-syhudschtxqosbjrmjfpm/build_logs/0001_stderr.txt)
- [Chromium DevTools](https://developer.chrome.com/docs/devtools/)

---

*最后更新: 2026-05-06*
