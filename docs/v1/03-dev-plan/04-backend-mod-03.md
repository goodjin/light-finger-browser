# 后端开发计划 - MOD-03 IP 池

## 文档信息

| 字段 | 内容 |
|------|------|
| **模块编号** | MOD-03 |
| **模块名称** | IP 池 |
| **对应架构** | docs/v1/02-architecture/03-mod-03-proxy.md |
| **优先级** | P0 |

---

## 1. 模块概述

- 提供代理获取、绑定、释放、检测与刷新能力
- 通过 Provider 适配器对接外部代理商

## 2. 接口清单

| 接口编号 | 方法 |
|---------|------|
| API-M3-001 | Acquire(country, type) |
| API-M3-002 | Release(proxyID) |
| API-M3-003 | Bind(proxyID, instanceID) |
| API-M3-004 | Unbind(proxyID) |
| API-M3-005 | HealthCheck(proxyID) |
| API-M3-006 | RefreshPool() |

## 3. 开发任务拆分

| 任务编号 | 内容 | 对应接口 |
|---------|------|---------|
| T-01 | 定义 `Proxy` / `ProxyFilter` / 状态枚举 | API-M3-001~006 |
| T-02 | 定义 `Store` 与 `ProxyProvider` 抽象 | API-M3-001~006 |
| T-03 | 实现候选排序与 `Acquire` | API-M3-001 |
| T-04 | 实现 `Release` / `Bind` / `Unbind` | API-M3-002~004 |
| T-05 | 实现健康检查与状态衰减 | API-M3-005 |
| T-06 | 实现 `RefreshPool` 与 Provider 拉取 | API-M3-006 |
| T-07 | 补充 Provider 适配器与过滤查询 | API-M3-001~006 |
| T-08 | 补充管理器与适配器单元测试 | API-M3-001~006 |

## 4. 验收清单

- 代理选择按成功率/延迟策略稳定工作
- 已绑定代理不会被二次占用
- 健康检查能正确更新状态并淘汰不可用代理

