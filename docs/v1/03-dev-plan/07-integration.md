# 集成测试计划

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目编号** | TMOS |
| **计划名称** | 集成测试计划 |
| **对应架构** | docs/v1/02-architecture/ |
| **优先级** | P0 |
| **预估工时** | 3 天 |

---

## 1. 测试范围

### 1.1 测试目标

验证所有模块之间的集成正确性，确保端到端流程正常运行。

### 1.2 测试边界

**包含**:
- 前后端接口集成
- 模块间依赖调用
- 数据库操作
- 外部服务 Mock

**不包含**:
- 单元测试覆盖的内容 (已在各模块测试中覆盖)
- 纯前端逻辑 (前端集成测试)
- 外部 TikTok API (需要真实账号)

---

## 2. 测试环境

### 2.1 环境配置

```yaml
# docker-compose.test.yml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: tmos_test
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
    ports:
      - "5432:5432"
    volumes:
      - ./migrations:/docker-entrypoint-initdb.d

  redis:
    image: redis:7
    ports:
      - "6379:6379"

  mcp-server:
    build: ./cmd/mcp
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://test:test@postgres:5432/tmos_test
      REDIS_URL: redis://redis:6379

  api-server:
    build: ./cmd/api
    ports:
      - "8081:8081"
    environment:
      DATABASE_URL: postgres://test:test@postgres:5432/tmos_test
      REDIS_URL: redis://6379
```

### 2.2 测试数据

```sql
-- 初始化测试数据
INSERT INTO tiktok_accounts (id, username, email, status) VALUES
  ('test-account-001', 'test_user_001', 'test001@example.com', 'active'),
  ('test-account-002', 'test_user_002', 'test002@example.com', 'active');

INSERT INTO browser_instances (id, status, account_id) VALUES
  ('test-instance-001', 'running', 'test-account-001');
```

---

## 3. 测试场景

### 3.1 账号注册流程集成测试

| 场景编号 | 场景名称 | 前置条件 | 测试步骤 | 预期结果 |
|---------|---------|---------|---------|---------|
| INT-001 | 完整注册流程 | 数据库可用 | 1. 调用 RegisterByGoogle 2. 验证账号创建 3. 验证资源分配 | 注册成功，账号状态为 active |
| INT-002 | 注册失败清理 | 数据库可用 | 1. 模拟注册失败 2. 验证资源释放 | IP、实例、手机号已释放 |
| INT-003 | 批量注册 | 数据库可用 | 1. 提交 10 个注册请求 2. 验证并发控制 | 最多 10 个并发执行 |

### 3.2 IP 池模块集成测试

| 场景编号 | 场景名称 | 前置条件 | 测试步骤 | 预期结果 |
|---------|---------|---------|---------|---------|
| INT-004 | IP 分配与绑定 | 数据库可用，代理服务可用 | 1. 调用 Acquire 2. 绑定到实例 3. 验证绑定关系 | IP 绑定成功 |
| INT-005 | IP 健康检查 | 有已分配的 IP | 1. 触发健康检查 2. 验证状态更新 | 失败 IP 状态更新为 dead |
| INT-006 | IP 池耗尽 | 模拟 IP 池为空 | 1. 调用 Acquire 2. 验证错误返回 | 返回 ErrNoAvailableProxy |

### 3.3 实例管理模块集成测试

| 场景编号 | 场景名称 | 前置条件 | 测试步骤 | 预期结果 |
|---------|---------|---------|---------|---------|
| INT-007 | 创建实例 | 指纹、IP 可用 | 1. 调用 Create 2. 验证进程启动 3. 验证 CDP 连接 | 实例状态为 running |
| INT-008 | 销毁实例 | 有运行中的实例 | 1. 调用 Destroy 2. 验证进程停止 3. 验证资源释放 | 端口、目录已释放 |
| INT-009 | 并发创建 | 数据库可用 | 1. 同时创建 20 个实例 2. 验证无端口冲突 | 20 个实例创建成功 |

### 3.4 验证码模块集成测试

| 场景编号 | 场景名称 | 前置条件 | 测试步骤 | 预期结果 |
|---------|---------|---------|---------|---------|
| INT-010 | 获取号码与验证码 | SMS-Activate API 可用 (Mock) | 1. 调用 GetNumber 2. 调用 WaitForCode | 成功获取号码和验证码 |
| INT-011 | 换绑手机号 | 有活跃账号 | 1. 调用 BindPhone 2. 验证账号更新 | 账号手机号已更新 |

### 3.5 AI 自动化模块集成测试

| 场景编号 | 场景名称 | 前置条件 | 测试步骤 | 预期结果 |
|---------|---------|---------|---------|---------|
| INT-012 | MCP 工具调用 | MCP 服务端运行 | 1. 发送 create_post 请求 2. 验证响应 | 返回 task_id |
| INT-013 | AI 内容生成 | OpenAI API 可用 (Mock) | 1. 调用 generate_content 2. 验证内容生成 | 返回 title, description, tags |
| INT-014 | 批量发布 | 有内容和一个账号 | 1. 调用 batch_publish 2. 验证发布结果 | 发布成功 |

### 3.6 端到端场景测试

| 场景编号 | 场景名称 | 前置条件 | 测试步骤 | 预期结果 |
|---------|---------|---------|---------|---------|
| INT-015 | 完整账号注册流程 | 数据库、外部服务可用 | 1. 分配指纹 2. 分配 IP 3. 获取手机号 4. 注册 TikTok 5. 接收验证码 6. 保存账号 | 账号创建成功 |
| INT-016 | 完整内容发布流程 | 有活跃账号和视频 | 1. AI 生成内容 2. 批量发布到账号 3. 验证发布结果 | 发布成功 |

---

## 4. 开发任务拆分

| 任务编号 | 任务名称 | 涉及文件 | 预估工时 | 依赖 |
|---------|---------|---------|---------|------|
| T-01 | 测试环境配置 | docker-compose.test.yml | 0.5 天 | - |
| T-02 | 测试数据准备 | migrations/, fixtures/ | 0.25 天 | T-01 |
| T-03 | 账号注册流程测试 | integration/account_test.go | 0.5 天 | T-02 |
| T-04 | IP 池模块测试 | integration/proxy_test.go | 0.25 天 | T-02 |
| T-05 | 实例管理模块测试 | integration/instance_test.go | 0.5 天 | T-02 |
| T-06 | 验证码模块测试 | integration/phone_test.go | 0.25 天 | T-02 |
| T-07 | AI 自动化模块测试 | integration/mcp_test.go | 0.5 天 | T-02 |
| T-08 | 端到端场景测试 | integration/e2e_test.go | 0.5 天 | T-03~07 |
| T-09 | 测试报告生成 | - | 0.25 天 | T-08 |

---

## 5. 详细任务定义

### T-01: 测试环境配置

**任务概述**: 配置集成测试环境

**输出**:
- `docker-compose.test.yml`
- `Makefile` (测试目标)
- `.env.test`

**实现要求**:
- PostgreSQL 15 + Redis 7
- 测试数据库初始化脚本
- 测试用环境变量配置

**验收标准**:
- [ ] **环境启动**: `docker-compose up -f docker-compose.test.yml` 成功
- [ ] **数据库就绪**: 测试数据库可连接
- [ ] **测试隔离**: 测试之间数据隔离

---

### T-02: 测试数据准备

**任务概述**: 准备测试数据和 fixtures

**输出**:
- `tests/fixtures/` - 测试数据 fixtures
- `migrations/` - 数据库迁移脚本

**实现要求**:
- 每个模块的测试数据 fixtures
- 测试前数据清理脚本
- 测试后数据回滚

---

### T-03: 账号注册流程测试

**任务概述**: 测试账号注册完整流程

**输出**:
- `integration/account_test.go`

**测试用例**:
```go
func TestCompleteRegistrationFlow(t *testing.T) {
    // 1. 分配指纹和 IP
    fp, _ := fingerprint.GenerateRandom("US")
    proxy, _ := proxy.Acquire(ctx, "US", proxy.ProxyTypeResidential)

    // 2. 创建实例
    instance, _ := instance.Create(ctx, &instance.Config{
        Fingerprint: fp,
        Proxy: proxy,
    })

    // 3. 获取手机号
    phone, _ := phone.GetNumber(ctx, "US", "tiktok")

    // 4. 执行注册
    account, _ := account.RegisterByGoogle(ctx, &RegisterRequest{
        Email: "test@example.com",
        Country: "US",
    })

    // 5. 验证
    assert.Equal(t, AccountStatusActive, account.Status)
}
```

---

### T-04 ~ T-07: 各模块集成测试

**任务概述**: 测试各模块的集成

**输出**:
- `integration/proxy_test.go`
- `integration/instance_test.go`
- `integration/phone_test.go`
- `integration/mcp_test.go`

---

### T-08: 端到端场景测试

**任务概述**: 测试完整端到端流程

**输出**:
- `integration/e2e_test.go`

---

### T-09: 测试报告生成

**任务概述**: 生成测试报告

**输出**:
- `test-report/` - 测试报告输出目录

---

## 6. 测试覆盖率目标

| 模块 | 覆盖率目标 | 实际覆盖率 |
|-----|-----------|-----------|
| MOD-01 指纹引擎 | ≥ 80% | - |
| MOD-02 实例管理 | ≥ 80% | - |
| MOD-03 IP 池 | ≥ 80% | - |
| MOD-04 验证码 | ≥ 80% | - |
| MOD-05 账号注册 | ≥ 80% | - |
| MOD-06 AI 自动化 | ≥ 80% | - |

---

## 7. 验收清单

- [ ] 所有集成测试通过
- [ ] 测试覆盖率 ≥ 70%
- [ ] 无测试遗留缺陷
- [ ] 测试报告已生成
