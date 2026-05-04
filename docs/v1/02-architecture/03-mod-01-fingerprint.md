# MOD-01: 指纹引擎模块

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目名称** | Light Finger Browser |
| **模块编号** | MOD-01 |
| **模块名称** | 指纹引擎模块 |
| **版本** | v1.0 |
| **对应PRD** | FR-002 |
| **更新日期** | 2026-04-30 |

---

## 1. 系统定位

### 1.1 在整体架构中的位置

```
┌─────────────────────────────────────────────────────┐
│      MOD-02 实例管理 / 浏览器运行时 / 诊断采样层        │
└───────────────────────┬─────────────────────────────┘
                        │ ▼ 调用
┌─────────────────────────────────────────────────────┐
│               ★ MOD-01 指纹引擎模块 ★                │
│   生成唯一指纹 / 校验一致性 / 提供运行时注入基线       │
└─────────────────────────────────────────────────────┘
```

### 1.2 核心职责

- 基于种子与国家配置生成确定性指纹
- 提供随机指纹生成能力
- 校验平台、GPU、时区、语言、屏幕等字段一致性

---

## 2. 对应 PRD

| PRD章节 | 编号 | 内容 |
|---------|-----|------|
| 功能需求 | FR-002 | 独立指纹生成 |
| 用户故事 | US-002 | 多实例并行管理 |

---

## 3. 接口定义

| 接口编号 | 方法 | 说明 |
|---------|------|------|
| API-M1-001 | Generate(seed, country) | 基于固定 seed 生成确定性指纹 |
| API-M1-002 | GenerateRandom(country) | 生成随机 seed 指纹 |
| API-M1-003 | Validate(fingerprint) | 校验指纹内部一致性 |

### 3.1 核心数据结构

```go
type Fingerprint struct {
    Seed      string
    UserAgent string
    Platform  string
    Screen    ScreenConfig
    Timezone  string
    Locale    string
    Canvas    CanvasConfig
    WebGL     WebGLConfig
    Audio     AudioConfig
    Hardware  HardwareConfig
    Network   NetworkConfig
}
```

---

## 4. 业务规则

1. 相同 `seed + country` 必须生成稳定可复现的同一份指纹。
2. `timezone` 与 `locale` 必须满足白名单映射关系。
3. `platform` 与 `gpu_vendor` 必须满足平台能力约束。
4. 指纹生成结果是实例模块与浏览器运行时的输入基线，不等于浏览器侧已完成注入。

---

## 5. 实现文件

| 文件路径 | 职责 |
|---------|------|
| `fingerprint/types.go` | 指纹数据结构与校验规则 |
| `fingerprint/generator.go` | 指纹生成逻辑 |
| `fingerprint/validator.go` | 一致性校验 |
| `fingerprint/config.go` | 国家级配置 |

---

## 6. 覆盖映射

| PRD类型 | PRD编号 | 架构元素 | 覆盖状态 |
|---------|---------|---------|---------|
| 功能需求 | FR-002 | MOD-01 | ✅ |
| 接口 | API-M1-001~003 | `fingerprint/` | ✅ |
