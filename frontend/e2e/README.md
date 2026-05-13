# E2E 测试

## 概述

Playwright E2E 测试套件，自动启动完整应用进行测试验证。

## 快速开始

### 安装依赖
```bash
cd /Users/jin/github/light-finger-browser/frontend
npm install
```

### 安装 Playwright 浏览器
```bash
npx playwright install --with-deps chromium
```

### 运行测试（完全自动化）

```bash
cd /Users/jin/github/light-finger-browser/frontend
npx playwright test
```

**自动完成以下操作：**
1. 启动 `wails dev` (完整应用)
2. 等待前端就绪
3. 运行所有测试
4. 自动清理

## 测试配置

| 配置项 | 值 |
|-------|-----|
| 测试入口 | `http://localhost:5173` |
| 浏览器 | Chromium |
| 启动超时 | 10 分钟 |
| 报告 | HTML Reporter |

## 测试文件

| 文件 | 描述 | 测试数量 |
|------|------|---------|
| `fingerprint-list.spec.ts` | 指纹列表页面 | 5 |
| `fingerprint-crud.spec.ts` | CRUD 操作 | 6 |
| `tabs-management.spec.ts` | 标签页管理 | 5 |
| `cross-flow.spec.ts` | 跨流程测试 | 3 |

## 测试结果

### 通过的测试（13）
- 页面加载和标题显示
- 新建指纹按钮显示
- 空状态/警告显示
- 指纹列表显示
- 指纹卡片信息完整性
- 展开/折叠卡片
- 标签页列表显示
- 新建标签页按钮
- 指纹详情显示
- 编辑对话框
- 复制对话框
- 删除操作

### 跳过的测试（7）
需要完整后端环境：
- 新建指纹对话框
- 选择国家
- 取消/关闭对话框
- 跨流程测试（新建到编辑/复制）

> **注意**：测试会自动检测后端是否就绪，如果后端未运行会自动跳过需要后端的测试。

## 其他命令

```bash
# 调试模式
npx playwright test --debug

# UI 模式
npx playwright test --ui

# 仅列出测试
npx playwright test --list

# 查看报告
npx playwright show-report

# 只运行前端测试（不需要后端）
# 启动前端服务器
npm run dev &
npx playwright test
```

## 故障排除

### 前端测试通过但后端测试失败
确保 `wails dev` 能成功编译 Go 代码：
```bash
cd /Users/jin/github/light-finger-browser
wails dev
```

### 端口冲突
```bash
# 清理占用端口的进程
lsof -ti :5173 | xargs kill -9
lsof -ti :34115 | xargs kill -9
lsof -ti :9222 | xargs kill -9
```

## 项目结构

```
frontend/
├── e2e/
│   ├── playwright.config.ts   # Playwright 配置
│   ├── start-app.sh          # 应用启动脚本
│   ├── README.md
│   └── tests/
│       ├── fingerprint-list.spec.ts
│       ├── fingerprint-crud.spec.ts
│       ├── tabs-management.spec.ts
│       └── cross-flow.spec.ts
```
