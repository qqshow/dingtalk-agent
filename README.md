# dingtalk-agent

> 让 AI Agent 读懂你的钉钉：本地解密钉钉桌面版聊天记录与日程，提供 **网页 + REST API + MCP**，
> 用于每日工作总结、互动统计、项目跟踪，并自动整理待办和待跟踪事项。

*Decrypt your own DingTalk desktop (V3) chat history and calendar locally, browse it in an embedded web UI,
and expose it to AI agents via REST and MCP for daily summaries and auto-generated to-dos.*

- **单文件，零依赖**：一个可执行文件，网页已内嵌；纯 Go 实现（含 SQLite），不需要 Python、CGO 或第三方解密工具
- **原生 V3 解密**：PBKDF2 + AES-ECB，正确处理 WAL；聊天记录和日历一次完整解密约 2 秒
- **为 Agent 设计**：结构化日报统计、按重要程度排序的消息摘要、日程，以及可写回的总结和待办
- **数据只在本机**：只读访问钉钉数据；默认只监听 `127.0.0.1`，并拦截跨站和 DNS rebinding 访问；没有任何外部网络请求

## 功能

| 模块 | 说明 |
|---|---|
| 消息 | 浏览全部单聊、群聊和项目群，支持图片（优先显示原图，可缩放）、文件、引用、富文本和 @ 高亮 |
| 工作台 | 每日统计（消息量、互动的人、参与的群和项目、会议、待我回复）、AI 日报、日程时间线、@我的消息 |
| 待办 | Agent 提取或手动添加的待办（todo）与待跟踪（follow_up）事项，支持优先级、截止日期，可跳回来源会话 |
| 日程 | 钉钉日历月视图：会议、会议室、组织者、审批待办、AI 听记、请假 |
| 搜索 | 全量关键字搜索，可限定会话和时间范围 |
| 导出 | 按会话导出 JSON（含图片和文件附件），便于交给 AI 分析 |
| 设置 | 自动同步频率、Agent 接入信息 |
| Agent 接口 | REST API 和 MCP（HTTP / stdio），提供 12 个工具和 2 个提示词模板 |

支持浅色和深色主题（默认跟随系统）。

## 环境要求

- **钉钉桌面版（新版）**：账号数据目录形如 `%APPDATA%\DingTalk\<hash>_v3`，需要至少登录过一次
- **操作系统**：Windows（已测试）。macOS / Linux 也会自动查找钉钉数据目录，但尚未测试
- 旧版 `*_v2` 数据库暂不支持

## 快速开始

1. 从 [Releases](https://github.com/qqshow/dingtalk-agent/releases) 下载 `dingtalk-agent.exe`，或用 Go 安装（需要 Go 1.26+）：

   ```bash
   go install github.com/qqshow/dingtalk-agent@latest
   ```

2. 运行：

   ```powershell
   .\dingtalk-agent.exe
   ```

3. 打开 <http://localhost:8090>

首次启动时，程序会自动查找钉钉账号目录，从钉钉客户端日志中识别你的数字 UID（并用数据库文件头验证），然后完成解密。
之后默认每 30 分钟自动同步一次，可在网页的 **设置** 页修改。

### 命令行

```
dingtalk-agent [serve]                    启动网页 + REST API + MCP(HTTP)，默认命令
dingtalk-agent mcp                        以 stdio 方式运行 MCP server
dingtalk-agent sync                       立即解密一次
dingtalk-agent report -date yesterday     输出某天的结构化统计（JSON）
dingtalk-agent digest -date today         输出某天的消息摘要（Markdown），-scope involved 只看与我相关的
dingtalk-agent digest -from 2026-11-16 -to 2026-11-20    一段时间的摘要
dingtalk-agent prompt -date today         输出推荐的每日总结提示词
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-port` | `8090` | 监听端口 |
| `-host` | `127.0.0.1` | 监听地址。设为 `0.0.0.0` 会把聊天记录暴露到局域网，请谨慎 |
| `-sync-interval` | `30` | 自动同步间隔（分钟，`0` 表示关闭）。指定后会覆盖网页上的设置 |
| `-uid` | 自动识别 | 钉钉数字 UID |
| `-dingtalk-dir` | 自动查找 | 账号目录 `...\DingTalk\<hash>_v3` |
| `-data` | `<程序目录>/data` | 本地数据目录 |

以上设置也可以写在 `data/config.json`（首次运行时自动生成）：

```json
{
  "uid": "123456789",
  "dingtalk_dir": "C:\\Users\\<you>\\AppData\\Roaming\\DingTalk\\<hash>_v3",
  "port": 8090,
  "sync_interval_minutes": 30
}
```

## 接入 AI Agent

### Claude Code

二选一：

```bash
# A. 连接已在运行的服务（HTTP）
claude mcp add --transport http dingtalk http://localhost:8090/mcp

# B. 由 Claude Code 直接启动（stdio，不依赖服务是否在运行）
claude mcp add dingtalk -- /path/to/dingtalk-agent mcp
```

然后在 Claude Code 中运行提示词模板：

```
/mcp__dingtalk__daily_summary                         # 今天的日报，并自动写入待办和待跟踪事项
/mcp__dingtalk__daily_summary yesterday
/mcp__dingtalk__weekly_summary 2026-11-16 2026-11-20  # 周报
```

生成的日报和待办会保存到本地，在网页的 **工作台** 和 **待办** 页面可以看到。

**每天自动运行**：可以用系统计划任务执行
`claude -p "/mcp__dingtalk__daily_summary today" --allowedTools "mcp__dingtalk__*"`，
也可以在 Claude Code 里用 `/schedule` 创建定时任务。

其他支持 MCP 的客户端（如 Claude Desktop）同样可以接入：HTTP 地址填 `http://localhost:8090/mcp`，或用 stdio 方式运行 `dingtalk-agent mcp`。

### MCP 工具

| 工具 | 作用 |
|---|---|
| `dingtalk_daily_report` | 某天或某段时间的统计：消息量、互动的人、参与的群和项目、@我、等待我回复、日程 |
| `dingtalk_daily_digest` | 按会话分组、按重要程度排序的消息全文（Markdown），用于大模型总结 |
| `dingtalk_get_messages` | 读取单个会话在某段时间内的全部消息 |
| `dingtalk_calendar` | 日程：会议（时间、会议室、组织者、是否回复邀请、备注）、审批待办、AI 听记、请假 |
| `dingtalk_search` | 关键字搜索 |
| `dingtalk_list_conversations` | 会话列表 |
| `dingtalk_sync` | 立即同步 |
| `dingtalk_save_summary` / `dingtalk_get_summary` | 保存 / 读取日报 |
| `dingtalk_add_todos` / `dingtalk_list_todos` / `dingtalk_update_todo` | 待办和待跟踪事项；重复写入同一事项只会更新内容，不会重复创建，也不会重新打开已完成的事项 |

提示词模板（MCP prompts）：`daily_summary`、`weekly_summary`。

### REST API

适用于任意 Agent 或脚本，接口说明可访问 `GET /api/agent` 查看。

- 日期参数：`YYYY-MM-DD`、`today`、`yesterday` 或 `-N`（N 天前）
- 时间区间用 `from`、`to`（都包含在内）
- 加上 `refresh=1` 时，如果数据已超过 5 分钟，会先同步再返回

```
POST   /api/agent/sync                       立即同步
GET    /api/agent/report?date=today          结构化统计
GET    /api/agent/digest?date=today&scope=all|involved&format=markdown|json
GET    /api/agent/messages?cid=<cid>&date=today
GET    /api/agent/calendar?from=2026-11-01&to=2026-11-30&include_closed=1
GET    /api/search?q=关键字&from=&to=&cid=
GET    /api/agent/summary?date=today
PUT    /api/agent/summary                    {"date":"2026-11-19","summary":"<markdown>","focus":["..."]}
GET    /api/agent/todos?status=open|done|dropped|all&kind=todo|follow_up
POST   /api/agent/todos                      {"items":[{"kind":"todo","title":"...","priority":"high","due":"2026-11-21"}]}
PATCH  /api/agent/todos/{id}                 {"status":"done"}
DELETE /api/agent/todos/{id}
GET    /api/settings
PUT    /api/settings                         {"sync_interval_minutes":10}
```

## 统计口径

- **互动的人**：单聊中有往来的人，加上群里 @我 或被我 @ 的人；不包括机器人和系统账号（如钉钉云盘）
- **参与的群**：我在群里发过言，或被单独 @ 过（@所有人 不算）
- **项目群**：钉钉标记为项目群的群（`sceneGroupCode` / `tagCodes` 为 `project`）
- **等待我回复**：单聊最后一条消息来自对方、我之后没有回复；或在群里被单独 @ 后，我没有在该群发言
- **会议**：钉钉日历中未取消的会议；请假（考勤）单独标记，不计入会议时长
- 系统提示（如"xx 发起了群聊"）不计入"我发送的消息"

## 常见问题

**提示无法确定 UID？**
程序会从钉钉日志（`%APPDATA%\DingTalk\log\`）中查找 UID。如果日志已被清理，可以在日志里搜索 `real_uid=` 找到自己的数字 UID，
再通过 `-uid` 参数或 `data/config.json` 指定。注意：V3 账号目录名中的那串字符是哈希值，不是 UID。

**图片模糊，或标着"缩略图"？**
钉钉只会为你在客户端里点开看过的图片下载原图，其余图片本地只有 720px 的缩略图。在钉钉中点开那张图片，再点一次"同步"即可看到原图。

**同步是增量的吗？**
不是。每次都是从钉钉客户端完整解密一次（约 2 秒，不影响钉钉使用），所以同步频率设得高一些也没有问题。

**能看到会议的参会人或 AI 听记内容吗？**
不能。钉钉本地只缓存了会议的组织者；AI 听记正文保存在服务器上。部分会议的备注中带有 AI 概要，这部分会被读取。

**会修改钉钉的数据吗？**
不会。程序只读取钉钉的数据库文件，解密后的副本保存在 `data/decrypted/`，Agent 写回的内容保存在独立的 `data/agent.db`。

## 数据目录

```
data/
  config.json        账号、端口、同步频率（自动生成）
  decrypted/         解密后的快照（每次同步生成新文件，旧文件自动清理）
  agent.db           Agent 写回的日报和待办，建议备份
  exports/           导出的 JSON 和附件
  sync_state.json    同步状态
  logs/              运行日志
```

`data/` 中包含解密后的聊天记录，请妥善保管，不要提交到代码仓库或分享给他人（`.gitignore` 已默认排除）。

## 隐私与免责声明

- 本项目仅用于**在你自己的电脑上读取你自己的钉钉数据**。请勿用于获取他人数据。
- 聊天记录可能包含同事、客户的个人信息和公司机密。使用前请确认符合所在组织的数据安全规定；
  将数据交给云端 AI 服务前，请评估其中的敏感信息。
- 本项目与钉钉、阿里巴巴无任何关联。"钉钉"及相关标识归其各自所有者所有。
- 钉钉客户端更新可能导致数据格式变化、解密失效。本软件按"原样"提供，不提供任何担保。

## 开发

需要 Go 1.26 或更高版本。

```bash
git clone https://github.com/qqshow/dingtalk-agent.git
cd dingtalk-agent
go build -ldflags "-s -w" -o dingtalk-agent.exe .   # Linux/macOS 下可去掉 .exe 后缀
go test ./...
```

项目结构：

```
main.go                 命令行入口
internal/
  config/   配置、账号目录发现、UID 识别
  decrypt/  V3 数据库解密（聊天记录与日历）
  store/    快照管理、消息解析、查询、日程
  analysis/ 日报统计、消息摘要、日程汇总
  agentdb/  Agent 写回的日报和待办
  syncer/   定时同步与状态
  export/   JSON 导出与 zip 打包
  web/      HTTP 服务、REST API、内嵌前端（static/，原生 JS，无构建步骤）
  mcp/      MCP server（工具与提示词，支持 stdio / HTTP）
```

欢迎提交 Issue 和 PR。提交 Issue 时，请不要附带真实的聊天内容、UID 或截图中的个人信息。

## 致谢与许可

- V3 解密算法与 UID 识别方法移植自 [idealistichacker/dingtalk-chat-exporter](https://github.com/idealistichacker/dingtalk-chat-exporter)（MIT），详见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
- 本项目以 [MIT License](LICENSE) 发布
