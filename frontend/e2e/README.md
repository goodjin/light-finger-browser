# E2E 测试

## 快速开始

### 1. 安装依赖
```bash
cd /Users/jin/github/light-finger-browser/frontend
npm install
```

### 2. 安装 Playwright 浏览器
```bash
npx playwright install --with-deps chromium
```

### 3. 运行测试

#### 仅前端测试（不需要后端）
```bash
# 启动前端开发服务器
cd /Users/jin/github/light-finger-browser/frontend
npm run dev &

# 运行测试（需要先确认 5173 端口可用）
npx playwright test
```

#### 完整测试（需要 wails dev）
```bash
# 在项目根目录启动 wails dev
cd /Users/jin/github/light-finger-browser
wails dev &

# 然后运行测试
cd frontend
npx playwright test
```

## 测试命令

```bash
# 运行所有测试
npx playwright test

# 运行特定测试文件
npx playwright test e2e/tests/fingerprint-list.spec.ts

# 调试模式
npx playwright test --debug

# UI 模式
npx playwright test --ui

# 仅列出测试
npx playwright test --list

# 查看报告
npx playwright show-report
```

## 测试文件

| 文件 | 描述 | 测试数量 |
|------|------|---------|
| `fingerprint-list.spec.ts` | 指纹列表页面测试 | 5 |
| `fingerprint-crud.spec.ts` | 指纹 CRUD 操作测试 | 6 |
| `tabs-management.spec.ts` | 标签页管理测试 | 5 |
| `cross-flow.spec.ts` | 跨流程测试 | 3 |

## 测试覆盖

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
需要完整的 wails dev 环境（Go 后端运行）：
- 新建指纹对话框
- 选择国家
- 取消/关闭对话框
- 跨流程测试

## 配置

配置文件: `e2e/playwright.config.ts`

- 测试入口: `http://localhost:5173` (Vite dev server)
- 浏览器: Chromium
- 报告: HTML Reporter
