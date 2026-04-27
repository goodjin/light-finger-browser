运营facebook流程：
1. 连接 Chrome
脚本通过 CDP (调试端口 19922) 连接到已有的 Chrome：
Chrome 必须在运行中，且开启了 --remote-debugging-port=19922
如果没有找到，使用这个端口直接启动一个用 v2ray 的新 Chrome并连接，
脚本找到 Chrome 后，创建新标签页

**Chrome 独立运行机制**：
- 启动 Chrome 时使用 `disown` 将其脱离脚本进程
- 即使脚本退出，Chrome 仍保持独立运行
- 重启脚本时会自动重新连接到已存在的 Chrome
- 避免因脚本重启导致浏览器关闭

注意：要屏蔽掉所有调试标记，要避免在http请求中带上任何调试模式的标记，所有请求都要跟用户打开浏览器访问一样。

2. 登录
打开facebook前先注入本地已经保存好的会话数据，再打开facebook.com。
对于打开的页面都要注入一段js程序，用于模拟人工点击按钮输入文字等。
如果本地没有会话数据则直接打开。
判断是否有成功登录，如果没有成功登录会退出到登录界面。
使用js输入账号密码：从项目根目录下resources/accounts.txt读取，再点击登录。

3.机器人验证
登录过程中可能会碰到几类问题：
如果登录过程中碰到二次验证的情况，提示人工处理。
如果登录过程中碰到机器人认证的情况，提示人工处理。
如果登录失败，提示人工处理。

3. 运营
进入facebook首页后，通过js进行以下操作：
查看消息，是否有重要消息，有的话提示用户。
浏览首页，碰到有趣的，点开进行评论，申请加好友，但不要超过5个。
注意：要模拟人工操作，不要浏览太快，每个贴子查看时间在1~10秒之随机。点击评论，输入评论，申请加好友，时间间隔都在3秒到10秒之间。

4.重要事项
登录facebook成功后要马上记录保存会话信息，留待下次使用。
运营要保持浏览器打开状态，永远不要有关闭的动作，遇到任何情况都不允许关闭浏览器。

### 2FA 和人工处理机制
- 检测到 2FA 时，无限等待用户完成，不超时
- 用户完成后发送 `resume` 命令或创建 `2fa-complete.json` 文件
- 任何需要人工介入的场景，浏览器都保持打开，等待信号

## 命令交互系统

脚本支持通过文件系统与 Claude Code 进行异步交互，实现远程控制。

### 启动命令模式

```bash
# 终端1：启动脚本（命令模式不执行预定义操作，直接进入命令轮询）
node src/rover.js --command

# 指定交互目录（默认当前目录）
node src/rover.js --command --cmd-dir /tmp/rover-session
```

### 发送命令

```bash
# 终端2：发送各种命令（默认当前目录）
node src/rover-cmd.js goto https://www.facebook.com/messages
node src/rover-cmd.js click "John Smith"      # 按文本点击
node src/rover-cmd.js scroll 500              # 滚动500px
node src/rover-cmd.js analyze                  # 分析当前页面
node src/rover-cmd.js screenshot               # 截图
node src/rover-cmd.js status                   # 查看状态
node src/rover-cmd.js history 10              # 查看最近10条历史
node src/rover-cmd.js pause                   # 暂停
node src/rover-cmd.js resume                  # 恢复
node src/rover-cmd.js terminate               # 终止脚本

# 指定交互目录（需与 --cmd-dir 配合使用）
node src/rover-cmd.js status --dir /tmp/rover-session
node src/rover-cmd.js goto https://facebook.com --dir /tmp/rover-session
```

### 核心文件

交互目录通过 `--cmd-dir` / `--dir` 参数指定（默认当前目录 `.`）：

| 文件 | 用途 |
|------|------|
| `rover-commands.json` | 命令队列（脚本读取 Claude 写入） |
| `rover-status.json` | 状态记录（脚本写入 Claude 读取） |

### 状态文件结构

```json
{
  "timestamp": "2026-04-23T...",
  "currentUrl": "https://www.facebook.com/...",
  "lastAction": {
    "commandId": "cmd-xxx",
    "action": "goto",
    "target": "https://www.facebook.com/messages",
    "result": "success",
    "error": null,
    "duration_ms": 2340,
    "pageSnapshot": { ... },
    "domFile": "dom-cmd-xxx-12345.json",
    "snapshotFile": "snapshot-cmd-xxx-12345.json",
    "screenshot": "screenshots/.../cmd-xxx.png"
  },
  "history": [
    { "commandId": "cmd-xxx", "action": "goto", "result": "success", ... }
  ],
  "browser": { "connected": true, "port": 19922 },
  "control": { "running": true, "paused": false }
}
```

### analyze 命令新增信息

执行 `analyze` 命令时，会保存以下文件到 `screenshots/` 目录：

| 字段 | 说明 |
|------|------|
| `domFile` | DOM 树 JSON 文件（简化版，最多5层） |
| `snapshotFile` | 页面快照摘要（URL、标题、栏目统计、cookies前10个、localStorage keys） |
| `pageSnapshot` | 内嵌的完整分析结果 |

### pageSnapshot 结构

```json
{
  "pageInfo": { "title": "Facebook", "url": "https://www.facebook.com/" },
  "sections": [
    { "text": "消息", "href": "https://...", "type": "message", "isVisible": true }
  ],
  "domTree": { "tag": "body", "children": [...] },
  "cookies": ["datr=xxx", "c_user=xxx", ...],
  "localStorage": { "key1": "value1", ... },
  "sessionStorage": { ... }
}
```

### 命令格式

```json
{
  "id": "cmd-xxx",
  "action": "goto",
  "target": "https://www.facebook.com/messages",
  "priority": 1,
  "timestamp": "2026-04-23T..."
}
```

### 支持的命令

| 命令 | 参数 | 说明 |
|------|------|------|
| `goto` | url | 导航到 URL |
| `click` | selector | 点击元素（文本或CSS选择器） |
| `type` | text | 输入文本 |
| `scroll` | distance | 滚动页面（px） |
| `wait` | ms | 等待（默认3000ms） |
| `screenshot` | name | 截图 |
| `analyze` | - | 分析当前页面 |
| `browse` | - | 执行浏览会话 |
| `pause` | reason | 暂停 |
| `resume` | - | 恢复 |
| `terminate` | - | 终止脚本 |

### 指数退避轮询

脚本采用指数退避算法轮询命令：

```
有命令 → 执行 → 立即检查下一条
              ↓
        没命令 → 等待1s → 没命令 → 等待2s → 没命令 → 等待4s...
              ↓                              ↓
        重置回1s                        最多退避到30s
```

连续3次没命令开始退避，最多退避到30秒。有新命令立即重置回1秒。

### 交互流程

```
┌─────────────────────────────────────────────────────┐
│                    Claude Code                        │
│                         │                            │
│                  写命令到 rover-commands.json         │
│                         │                            │
└─────────────────────────┼─────────────────────────────┘
                          │ 定时轮询（指数退避）
                          ↓
┌─────────────────────────────────────────────────────┐
│                   rover-core.js                       │
│                    (Commander)                        │
│                         │                            │
│               读取命令 → 执行 → 写状态                 │
│                         │                            │
│               读取 rover-status.json                  │
└─────────────────────────────────────────────────────┘
```

### 优先级

命令按 `priority` 排序（数字越小优先级越高）：
- `goto`: 1（最高）
- `click`/`analyze`: 2
- `scroll`: 3
- `wait`/`screenshot`: 5

再强调一次重要事项：
登录facebook成功后要马上记录保存会话信息，留待下次使用。
运营要保持浏览器打开状态，永远不要有关闭的动作，遇到任何情况都不允许关闭浏览器。
