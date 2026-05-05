# 指纹测试服务器设计

> 验证指纹生成器输出的指纹在实际浏览器运行时是否正确注入的独立服务。

## 文档信息

| 字段 | 内容 |
|------|------|
| **版本** | v1.0 |
| **更新日期** | 2026-05-05 |
| **对应 PRD** | `docs/v1/01-prd.md` |

---

## 1. 背景与目标

### 1.1 问题陈述

当前的 `fingerprint_coverage.go` 只能报告"理论覆盖"——即指纹生成器能生成哪些字段，以及哪些字段在浏览器启动时通过启动参数或 CDP override 注入。但它无法验证：

- 实际运行中的浏览器实例，特定指纹字段是否真的被网站检测到
- 生成的指纹在不同网站（Google, Facebook, Cloudflare 等）下的一致性
- Canvas/WebGL/Audio 指纹在实际页面渲染后的 hash 值

### 1.2 目标

构建一个独立的 **Fingerprint Test Server**，用于：

1. **接收** 指纹种子（seed）和目标网站 URL
2. **启动** 一个临时浏览器实例，应用该指纹
3. **访问** 目标网站，通过 CDP 采集实际指纹值（canvas hash, WebGL vendor/renderer, audio hash, etc.）
4. **对比** 采集值与指纹生成器的理论输出
5. **返回** 对比结果：pass/fail 及差异详情

---

## 2. 架构设计

### 2.1 组件关系

```
+----------------+     +-------------------+     +------------------+
| Fingerprint    | --> | Fingerprint Test  | --> | Temporary        |
| Generator      |     | Server            |     | Browser Instance |
| (existing)     |     | (new)             |     | (CDP-controlled) |
+----------------+     +-------------------+     +------------------+
                               |
                               v
                        +------------------+
                        | Fingerprint DB   |
                        | (new)            |
                        +------------------+
```

### 2.2 模块职责

| 模块 | 职责 |
|------|------|
| `fptest/server.go` | HTTP/gRPC API，接收测试请求，调度测试任务 |
| `fptest/runner.go` | 测试执行器，启动临时浏览器，执行 CDP 采集 |
| `fptest/collector.go` | 通过 CDP 采集实际指纹值（canvas, webgl, audio, etc.） |
| `fptest/comparator.go` | 对比理论指纹与实际采集值，计算差异 |
| `fptest/store.go` | 存储测试结果到 SQLite |

### 2.3 API 设计

#### POST /api/v1/fingerprint/test

请求：
```json
{
  "seed": "user-123-seed",
  "country": "US",
  "target_url": "https://browserleaks.com/canvas",
  "fingerprint_fields": ["canvas.hash", "webgl.vendor", "audio.hash"]
}
```

响应：
```json
{
  "test_id": "test-uuid",
  "status": "completed",
  "results": [
    {
      "field": "canvas.hash",
      "expected": "0xabc123",
      "actual": "0xabc123",
      "match": true
    },
    {
      "field": "webgl.vendor",
      "expected": "Google Inc (NVIDIA)",
      "actual": "Google Inc (Intel)",
      "match": false
    }
  ],
  "tested_at": "2026-05-05T12:00:00Z"
}
```

#### GET /api/v1/fingerprint/test/:id

获取指定测试的结果。

#### GET /api/v1/fingerprint/test/history

获取历史测试结果列表，支持分页和过滤。

---

## 3. CDP 采集字段映射

| 指纹字段 | CDP 方法 | 说明 |
|---------|---------|------|
| `canvas.hash` | `Page.captureScreenshot` + offscreen canvas hash | 需要执行 JS 采集 |
| `webgl.vendor` | `Page.evaluate` -> `WebGLRenderingContext.getParameter(UNMASKED_VENDOR_WEBGL)` | 需要执行 JS |
| `webgl.renderer` | `Page.evaluate` -> `WebGLRenderingContext.getParameter(UNMASKED_RENDERER_WEBGL)` | 需要执行 JS |
| `audio.hash` | `Page.evaluate` -> `OfflineAudioContext` hash | 需要执行 JS |
| `timezone` | `Emulation.getTimezoneOverride` | 通过 CDP 获取当前时区设置 |
| `locale` | `Browser.getBrowserCommandLine` | 从启动参数获取 |
| `platform` | `Browser.getBrowserCommandLine` | 从启动参数获取 |
| `user_agent` | `Network.getUserAgentOverride` | 从 CDP 获取 |

---

## 4. 临时实例生命周期

```
1. runner.Start(seed, country) -> 启动临时浏览器实例
2. instance.Navigate(target_url) -> 打开目标页面
3. instance.WaitForIdle() -> 等待页面稳定
4. collector.Collect(fields) -> 通过 CDP 采集各字段
5. comparator.Compare(generated_fp, collected) -> 对比差异
6. store.SaveResult(test_id, comparison) -> 存储结果
7. instance.Stop() -> 关闭临时浏览器
8. 返回测试结果
```

---

## 5. 存储设计

### 5.1 数据库表

```sql
CREATE TABLE fingerprint_tests (
    id TEXT PRIMARY KEY,
    seed TEXT NOT NULL,
    country TEXT NOT NULL,
    target_url TEXT NOT NULL,
    status TEXT NOT NULL, -- 'pending', 'running', 'completed', 'failed'
    created_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP
);

CREATE TABLE fingerprint_test_results (
    id TEXT PRIMARY KEY,
    test_id TEXT NOT NULL,
    field TEXT NOT NULL,
    expected TEXT,
    actual TEXT,
    match INTEGER NOT NULL, -- 0 or 1
    FOREIGN KEY (test_id) REFERENCES fingerprint_tests(id)
);
```

---

## 6. 与现有代码的关系

### 6.1 复用 `fingerprint.Generator`

`fptest/comparator.go` 调用 `fingerprint.NewGenerator().Generate(seed, country)` 获取理论指纹值，与采集的实际值对比。

### 6.2 复用 `instance.CDPClient`

`fptest/collector.go` 复用 `instance.CDPClient` 连接临时浏览器实例，通过 CDP 采集数据。

### 6.3 不依赖 `instance.Manager`

测试服务器启动自己的临时浏览器实例，不通过 `instance.Manager`。每个测试独立运行，测试完成后立即销毁。

---

## 7. 状态

**规划中** - 尚未进入实现阶段。

---

## 8. 依赖项

| 依赖 | 类型 | 说明 |
|------|------|------|
| `fingerprint/` | 内部 | 指纹生成器 |
| `instance/` | 内部 | CDP 客户端和 WebSocket 拨号器 |
| `storage/sqlite/` | 内部 | 测试结果存储 |
| `browser binary` | 外部 | 通过 `BROWSER_BINARY` 或 `SelfBuiltBrowserManager` 启动 |
