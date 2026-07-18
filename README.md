# deepin-agent-playground

> 把 deepin 25 变成 Sutton 路线"option 学习"的真实环境
>
> **版本**：v0.3 MVP · 2026-07-18（Phase 2：Eino 接入 + 与 deepin-agent-teams 统一）
>
> **技术栈**：Go 1.22.2 · Eino v0.9.12 ADK · godbus/dbus · 玲珑包
>
> **状态**：Phase 2 已实现 · 3 个 tool + 双模式 agent（Eino / Legacy） + 完整测试

## 与 deepin-agent-teams 版本对齐

本项目与 [deepin-agent-teams](https://github.com/sshnuke3/deepin-agent-teams) v4-redesign 保持一致：

- **Go 版本**：`go 1.22.2`（与 teams 一致 · 兼容 deepin 25 默认 Go）
- **Eino**：`v0.9.12`（与 teams 一致）
- **Sonic**：`v1.15.0` + `loader v0.5.0`（与 teams 一致）

可以共享代码模式 + 同一工具链部署。

---

## 快速开始

```bash
# 默认构建（不含 Eino · sandbox 友好）
make build

# Eino 模式构建（需要 deepin 25 + Go 1.22）
make build-eino

# Dry-run demo（无需 deepin 25 环境）
make demo-dry

# Legacy 模式 demo（手写 agent 循环）
make demo-legacy

# Eino 模式 demo（需要 Ollama 跑着）
make demo-eino

# 跑测试
make test

# 玲珑包打包（需要 deepin 25 + ll-builder）
make linglong
```

## 项目结构

```
deepin-agent-playground/
├── cmd/playground/main.go              # 入口
├── internal/
│   ├── agent/core.go                   # Agent 主循环（reasoning + acting）
│   ├── tools/
│   │   ├── base.go                     # Tool 抽象基类 + Result 统一结构
│   │   ├── ll_cli.go                   # 玲珑包适配（install / run）
│   │   ├── filesystem.go               # 文件整理（机器可读反馈）
│   │   └── dde_wallpaper.go            # DDE 壁纸（D-Bus via godbus）
│   ├── metrics/collector.go            # 训练数据采集
│   └── config/config.go                # 配置加载
├── examples/demo_install_organize_wallpaper.go  # e2e demo
├── tests/                              # 测试
├── Makefile                            # 构建脚本
├── linglong.yaml                       # 玲珑包配置
├── go.mod / go.sum
└── README.md
```

## MVP 实现的 3 个 tool

| Tool | 适配层 | Sutton 价值 |
|------|--------|------------|
| `ll_cli_install` | 玲珑包 install | 真实环境反馈（exit code）|
| `filesystem_move_by_ext` | 文件操作 | before/after 状态对比 = 可验证奖励 |
| `dde_wallpaper_set` | DDE Appearance D-Bus | 跨进程状态修改 + 二次读取验证 |

## 演示场景

**用户输入**：`帮我装计算器，整理 ~/Downloads 按扩展名分类，把壁纸换成日落`

**Agent 拆解**：
- option 1: `ll_cli_install` 安装 `org.deepin.calculator`
- option 2: `filesystem_move_by_ext` 按扩展名整理下载文件
- option 3: `dde_wallpaper_set` 设置壁纸

**输出**：每步执行结果 + 整体 metrics 报告（成功率 / 平均耗时 / 错误类型分布）

---

## 详细设计

---

## 1. 项目背景

### 1.1 为什么做这件事？

Sutton（2024 图灵奖得主）在 2026/7 WAIC 主论坛指出：**大模型没有原生智能，AI 已进入"经验时代"**。

他的 OaK 路线图（FC-STOMP 5 步）里，**最难的两步**（第三步 option + 第四步世界模型）必须靠**真实环境中的长程 agent 学习**——这跟桌面操作系统天然契合。

deepin 25 + 玲珑包 + DDE 是国内桌面 OS 里**最有 RL 训练潜力**的候选：
- 玲珑包 = 应用沙箱隔离（agent 工具调用的兜底）
- DDE = 丰富的"可操作对象"（文件 / 设置 / 通知 / 窗口）
- 但**目前没有 deepin 专属的 agent 框架**

**机会窗口**：谁先做出"deepin 上的 option 训练场"，谁就占据国产 OS 在 RL 路线里的"中国版试验场"位置。

### 1.2 不做什么

- ❌ 不做"通用 LLM 客户端"（那跟 chatbox 一样，已经有现成的）
- ❌ 不做"云端推理服务"（DeepSeek 已经在做）
- ❌ 不替代 AutoGPT / LangChain（而是它们的 **deepin adapter**）
- ❌ 不碰训练模型（只做"环境 + 工具调用稳定性测量"）

---

## 2. 技术栈：Go + Eino（v0.2 升级）

### 2.1 为什么选 Go + Eino？

**Eino（cloudwego/eino）** 是字节跳动开源的 Go 大模型应用框架，2026/7 最新版 v0.3.21+，持续活跃更新：

- ✅ **生产级验证**：扣子（Coze）底层就是用 Eino
- ✅ **Lambda Chain + Graph 工作流**：完美匹配 Sutton option 学习链
- ✅ **Agent Development Kit (ADK)**：工具调用 + 多 agent 协作 + 中断恢复
- ✅ **Graph Checkpoint + Node Interrupt**：Sutton 第三步"长程 option"的天然支撑（任务断点续跑 / 中断控制）
- ✅ **Ollama 官方组件**：直接接 DeepSeek R1 本地推理
- ✅ **同语言生态对齐**：deepin 25 / 玲珑包 / dde-daemon 全是 Go 项目

**生态匹配度**：

| 组件 | 选型 | 同生态项目 |
|------|------|----------|
| LLM 框架 | `cloudwego/eino` | CloudWeGo（字节） |
| RPC（可选） | `cloudwego/kitex` | CloudWeGo |
| Ollama 客户端 | `eino-ext/components/model/ollama` | Eino 官方扩展 |
| D-Bus | `github.com/godbus/dbus/v5` | deepin 团队同款 |
| 玲珑包 | `os/exec` 调 `ll-cli` | 玲珑（deepin） |
| 部署 | `ll-builder build` 打包 .layer | 玲珑（deepin） |

**关键决策**：

- ✅ 选 **Eino 而非自己写**——避免重复造轮子 · 直接复用字节生产级基础设施
- ✅ 选 **godbus 而非 cgo dbus**——纯 Go 实现，跨平台好编译
- ❌ 不选 Python LangChain——同语言生态对齐更重要
- ❌ 不选 LangChain Go（不存在）——Eino 就是 Go 版的 LangChain + Google ADK

### 2.2 Eino 核心能力映射

| Eino 能力 | 本项目用法 | Sutton 价值 |
|----------|-----------|------------|
| ChatModel | 接 Ollama DeepSeek R1 1.5b/7b | 推理层 |
| Tool Interface | 每个 DDE / 玲珑 / 文件 tool 实现 eino Tool 接口 | option 抽象 |
| Graph | 任务拆解 + 工具选择流程 | 决策链 |
| Lambda Chain | 多 step agent 流程编排 | FC-STOMP 框架 |
| ADK Runner | agent 主循环 | reasoning + acting |
| Graph Checkpoint | 任务断点续跑 | 长程 option 必备 |
| Node Interrupt | 工具调用超时 / 错误中断 | 失败恢复 |

---

## 3. 核心架构

### 3.1 三层结构

```
┌─────────────────────────────────────┐
│ Layer 1 · Agent Core（推理层）       │
│  - Eino ChatModel（Ollama + R1）    │
│  - ADK Agent（任务拆解 + 工具选择）  │
│  - Graph 工作流（FC-STOMP 框架）     │
└─────────────────────────────────────┘
              ↓ Tool Interface
┌─────────────────────────────────────┐
│ Layer 2 · Tool Adapter（适配层）     │
│  - ll-cli 适配（应用管理）           │
│  - DDE D-Bus 适配（系统设置）        │
│  - 文件系统适配（读写/移动）          │
│  - 错误恢复 + 重试策略                │
└─────────────────────────────────────┘
              ↓ 真实环境
┌─────────────────────────────────────┐
│ Layer 3 · deepin 25 Real Env        │
│  - 玲珑包沙箱                        │
│  - DDE 桌面环境                      │
│  - 文件系统                          │
│  - 真实 exit code / 文件路径反馈     │
└─────────────────────────────────────┘
```

### 3.2 关键设计决策

**决策 1：环境反馈是机器可读的，不依赖人类评分**

- ❌ 不做"agent 做得好不好让 LLM 打分"
- ✅ 直接读 `exec.Command()` 的 exit code + stdout/stderr
- ✅ 检测文件系统变化（移动前后的路径对比）
- ✅ 检测 DDE 状态变化（壁纸是不是真的换了）

**这正是 Sutton 路线第三步 option 学习最稀缺的——可验证奖励信号。**

**决策 2：每个 tool 实现 eino Tool 接口**

```go
type Tool interface {
    Info() *schema.ToolInfo       // 工具元信息
    InvokableRun(ctx, input, opts) (output, err)  // 执行逻辑
}
```

这样所有 tool 天然被 Eino Graph / Agent 识别，**无需自己写注册机制**。

**决策 3：失败重试 + 错误分类**

- 临时错误（网络抖动 / 文件锁）→ 重试 3 次
- 权限错误 → 提示用户授权
- 未知错误 → 记录日志 + 上报 deepin 团队

**决策 4：Graph Checkpoint 做长程 option**

Eino 的 Graph Checkpoint 支持 agent 任务断点续跑——这正好对应 Sutton 第三步"option 跨会话"的能力。

---

## 4. 最小可跑 demo（MVP）

### 4.1 演示场景

**用户输入**："帮我装计算器，整理下载文件夹，把壁纸换成日落"

**Agent 拆解**：
```
Task: 装计算器 + 整理下载 + 换壁纸
  ↓ 拆成 3 个 option（Eino Graph 节点）
Option 1: install_calculator
  - tool: ll_cli_install
  - input: "org.deepin.calculator"
  - success: exit code 0 + "已安装"

Option 2: organize_downloads
  - tool: file_move
  - input: {src: ~/Downloads/, dst: ~/Documents/sorted/, by: extension}
  - success: 文件确实移动到目标目录

Option 3: change_wallpaper
  - tool: dde_wallpaper
  - input: "sunset.jpg"
  - success: 壁纸确实变了（通过 D-Bus 查回当前壁纸对比）
```

### 4.2 文件结构

```
deepin-agent-playground/
├── go.mod                              # Go 1.22+, 引入 eino + 扩展
├── go.sum
├── Makefile                            # ll-builder build 一键打包
├── README.md
├── docs/
│   ├── architecture.md                 # 架构详解
│   ├── tools-api.md                    # tool 接口规范
│   ├── eino-usage.md                   # Eino Graph / ADK 用法
│   └── metrics.md                      # 训练数据采集
├── cmd/
│   └── playground/
│       └── main.go                     # 入口
├── internal/
│   ├── agent/
│   │   ├── core.go                     # Eino ADK Agent 主循环
│   │   ├── graph.go                    # Eino Graph 定义（任务拆解）
│   │   └── prompts.go                  # Prompt 模板
│   ├── tools/
│   │   ├── base.go                     # Tool 抽象基类
│   │   ├── ll_cli.go                   # 玲珑包工具（install/run/list）
│   │   ├── dde_wallpaper.go            # DDE 壁纸（D-Bus via godbus）
│   │   ├── dde_files.go                # DDE 文件管理器（D-Bus）
│   │   ├── dde_network.go              # DDE 网络设置（D-Bus）
│   │   ├── dde_theme.go                # DDE 主题/外观（D-Bus）
│   │   └── filesystem.go               # 文件读写移动
│   ├── dbus/
│   │   └── helper.go                   # godbus 通用调用 helper
│   ├── metrics/
│   │   ├── collector.go                # 收集每次工具调用的结果
│   │   └── reporter.go                 # 报告成功率/延迟
│   └── config/
│       └── config.go                   # 配置加载（ollama endpoint / 模型）
├── tests/
│   ├── test_ll_cli.go                  # 单 tool 单元测试
│   ├── test_dde_wallpaper.go
│   └── test_e2e.go                     # 端到端场景
└── examples/
    └── demo_install_organize_wallpaper.go
```

### 4.3 第一个 PR 的范围（MVP）

**只做 3 个 tool**：
1. `ll_cli.Install(package_name)` —— 验证玲珑包适配
2. `filesystem.MoveByExt(src, dst, extensions)` —— 验证文件操作 + 环境反馈
3. `dde_wallpaper.Set(path)` —— 验证 godbus D-Bus 适配

**1 个端到端 demo**：装计算器 + 整理下载 + 换壁纸

**1 个 metrics 报表**：跑完 10 次 demo 后输出成功率

### 4.4 MVP 关键代码骨架

```go
// cmd/playground/main.go
package main

import (
    "context"
    "log"
    
    "github.com/cloudwego/eino-ext/components/model/ollama"
    "github.com/cloudwego/eino/adk"
    
    "deepin-agent-playground/internal/tools"
)

func main() {
    ctx := context.Background()
    
    // Layer 1: ChatModel（Ollama + DeepSeek R1）
    chatModel, _ := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
        BaseURL: "http://localhost:11434",
        Model:   "deepseek-r1:1.5b",
    })
    
    // Layer 2: Tool Adapters（实现 eino Tool 接口）
    llCliTool := tools.NewLlCliInstall()
    fsTool := tools.NewFilesystemMove()
    wallpaperTool := tools.NewDdeWallpaper()
    
    // Eino ADK Agent（自动推理 + 工具选择）
    agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model: chatModel,
        Tools: []adk.Tool{llCliTool, fsTool, wallpaperTool},
    })
    
    runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
    
    // Layer 3: 真实 deepin 25 环境
    iter := runner.Query(ctx, 
        "帮我装计算器 org.deepin.calculator，整理 ~/Downloads 按扩展名分类，" +
        "把壁纸换成 /usr/share/wallpapers/sunset.jpg")
    
    for {
        event, ok := iter.Next()
        if !ok {
            break
        }
        log.Println(event.Message.Content)
    }
}
```

```go
// internal/tools/ll_cli.go
package tools

import (
    "context"
    "os/exec"
    "github.com/cloudwego/eino/schema"
)

type LlCliInstall struct{}

func NewLlCliInstall() *LlCliInstall { return &LlCliInstall{} }

func (l *LlCliInstall) Info() *schema.ToolInfo {
    return &schema.ToolInfo{
        Name: "ll_cli_install",
        Desc: "通过玲珑包安装应用",
        Params: map[string]*schema.ParameterInfo{
            "package_id": {Type: schema.String, Required: true,
                Desc: "玲珑包 ID，如 org.deepin.calculator"},
        },
    }
}

func (l *LlCliInstall) InvokableRun(ctx context.Context, 
    input *schema.ToolInput, opts ...interface{}) (*schema.ToolOutput, error) {
    
    pkgID := input.Params["package_id"].(string)
    cmd := exec.CommandContext(ctx, "ll-cli", "install", pkgID)
    out, err := cmd.CombinedOutput()
    
    return &schema.ToolOutput{
        Content: string(out),
        Meta: map[string]interface{}{
            "exit_code": cmd.ProcessState.ExitCode(),
            "success":   err == nil,
        },
    }, nil
}
```

```go
// internal/tools/dde_wallpaper.go
package tools

import (
    "context"
    "github.com/godbus/dbus/v5"
    "github.com/cloudwego/eino/schema"
)

type DdeWallpaper struct{ conn *dbus.Conn }

func NewDdeWallpaper() *DdeWallpaper {
    conn, _ := dbus.SessionBus()
    return &DdeWallpaper{conn: conn}
}

func (d *DdeWallpaper) Info() *schema.ToolInfo {
    return &schema.ToolInfo{
        Name: "dde_wallpaper_set",
        Desc: "设置 DDE 桌面壁纸",
        Params: map[string]*schema.ParameterInfo{
            "path": {Type: schema.String, Required: true,
                Desc: "壁纸绝对路径"},
        },
    }
}

func (d *DdeWallpaper) InvokableRun(ctx context.Context,
    input *schema.ToolInput, opts ...interface{}) (*schema.ToolOutput, error) {
    
    path := input.Params["path"].(string)
    obj := d.conn.Object("com.deepin.daemon.Appearance",
                          "/com/deepin/daemon/Appearance")
    
    // 调用 DDE Appearance.Set
    err := obj.Call("com.deepin.daemon.Appearance.Set", 0,
                     "background", path).Err
    
    // 验证（机器可读反馈）
    currentVariant, _ := obj.GetProperty("com.deepin.daemon.Appearance.background")
    
    return &schema.ToolOutput{
        Content: "壁纸设置成功",
        Meta: map[string]interface{}{
            "success": err == nil,
            "current": currentVariant.Value(),
            "requested": path,
            "verified": currentVariant.Value() == path,
        },
    }, nil
}
```

---

## 5. 关键技术细节

### 5.1 godbus D-Bus 调用（与 deepin 同款）

deepin 团队自己用 `godbus/dbus/v5` —— 我们用同样的库，保证兼容性。

**调壁纸（验证前后对比）**：

```go
// 设置
obj.Call("com.deepin.daemon.Appearance.Set", 0, "background", "sunset.jpg")

// 读取验证
current, _ := obj.GetProperty("com.deepin.daemon.Appearance.background")
assert(current.Value() == "sunset.jpg")  // 强验证 = Sutton 可验证奖励
```

### 5.2 Eino Graph 工作流（任务拆解）

```go
// internal/agent/graph.go
func buildAgentGraph() *compose.Graph[Input, Output] {
    g := compose.NewGraph[Input, Output]()
    
    _ = g.AddLambdaNode("parse_intent", parseIntent)
    _ = g.AddLambdaNode("select_tools", selectTools)
    _ = g.AddLambdaNode("execute_tool_1", executeFirstTool)
    _ = g.AddLambdaNode("execute_tool_2", executeSecondTool)
    _ = g.AddBranch("branch", branchOnSuccess)
    
    _ = g.AddEdge("parse_intent", "select_tools")
    _ = g.AddEdge("select_tools", "branch")
    _ = g.AddBranchEdge("branch", "execute_tool_1", "execute_tool_2")
    
    return g
}
```

### 5.3 Graph Checkpoint（长程 option）

Eino 的 Graph Checkpoint 让 agent 任务可断点续跑：

```go
// 保存进度
checkpoint, _ := g.Checkpoint(ctx, state)

// 第二天恢复
state, _ = g.ResumeFromCheckpoint(ctx, checkpoint)
```

—— 这就是 Sutton 第三步"跨会话 option"在工程上的对应。

### 5.4 玲珑包打包（一键 .layer 文件）

```makefile
# Makefile
build:
	CGO_ENABLED=0 go build -o deepin-agent-playground ./cmd/playground

linglong:
	ll-builder build
	# 输出: org.deepin.agent-playground_0.1.0_amd64.layer
```

deepin 用户双击 `.layer` 文件即装。

---

## 6. 工作量估算

| 阶段 | 时间 | 产出 |
|------|------|------|
| **Phase 1 · MVP** | 1-2 天 | 3 个 tool + 1 个 e2e demo + 1 个 metrics 报表 |
| **Phase 2 · 完善** | 3-5 天 | + 7 个 DDE tool + Eino Graph 复杂工作流 + 文档 |
| **Phase 3 · 训练数据** | 5-7 天 | + 自动采集 option 数据 + 导出 RL 数据集 |
| **Phase 4 · 社区** | 1 天 | 发 deepin 论坛 + 开源仓库 + README |

**总投入**：~2 周（如果主人有空协作可以压缩到 1 周）

---

## 7. 风险与边界

### 7.1 已知风险

| 风险 | 缓解 |
|------|------|
| 玲珑包接口可能未来变化 | 用 os/exec 直接调 · 不绑 SDK |
| DDE D-Bus 接口没有官方文档 | 通过 dde-daemon 源码反推 · 给 deepin 团队提 issue |
| DeepSeek R1 1.5b 推理能力弱 | 任务拆解要简单 · 多用 CoT prompt |
| agent 误操作（如误删文件） | 工具白名单 + 高危操作二次确认 |
| Eino Graph 复杂流程调试难 | 用 Eino 自带的 graph checkpoint debug |
| godbus 与 deepin 版本兼容 | 跟 deepin 团队同步版本号 |

### 7.2 不做的事（明确边界）

- ❌ 不碰模型训练
- ❌ 不替代 sudo / 权限管理
- ❌ 不做云端同步
- ❌ 不做 UI（命令行优先 · 用 Eino 提供的 trace 工具调试）
- ❌ 不替代 deepin 官方 app

---

## 8. 参考资料

- [deepin 25 发行注记](https://www.deepin.org/zh/deepin-25-alpha-release/)
- [玲珑官方文档](https://www.linglong.space/)
- [dde-daemon 源码](https://github.com/linuxdeepin/dde-daemon)
- [DTK API 文档](https://github.com/linuxdeepin)
- [cloudwego/eino 框架](https://github.com/cloudwego/eino)
- [eino-ext 扩展组件](https://github.com/cloudwego/eino-ext)
- [godbus/dbus](https://github.com/godbus/dbus)
- [Sutton RLC 2025 演讲](https://rl-conference.cc/)
- [DeepSeek R1 技术报告](https://github.com/deepseek-ai/DeepSeek-R1)
- [Ollama 官方文档](https://ollama.com/)

---

**审核请求**：主人看完后定 1-6 + 范围（Phase 1 vs 完整 4 阶段），我再开干 🦞