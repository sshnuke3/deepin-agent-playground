//go:build eino

// Package agent/eino_agent.go 提供基于 Eino ADK 的 agent 实现。
//
// 这是 Phase 2 的升级：用字节跳动 Eino 框架替换手写 agent 循环。
// 关键变化：
//   - ChatModelAgent 自动处理 ReAct 循环（model → tool calls → model）
//   - Runner 提供统一的 agent 执行入口 + 流式输出
//   - Tool 接口由 Eino 定义（通过 tools.EinoAdapter 适配我们的 Tool）
//
// Sutton option 学习的对应：
//   - ChatModelAgent 的每次 tool call = 1 个 option 执行
//   - Runner 的事件流 = option 链的轨迹（可记录为训练数据）
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/sshnuke3/daplayground/internal/metrics"
	"github.com/sshnuke3/daplayground/internal/tools"
)

// EinoAgent 是基于 Eino ADK 的 agent 实现。
//
// 它包装：
//   - ChatModel（接 Ollama DeepSeek R1）
//   - 一组 Eino 适配的 tool（通过 tools.EinoAdapter）
//   - Runner（统一入口 + 流式事件）
type EinoAgent struct {
	chatModel  model.BaseChatModel
	einoTools  []tool.BaseTool
	innerTools []tools.Tool
	runner     *adk.Runner
	mtr        *metrics.Collector
}

// NewEinoAgent 创建基于 Eino 的 agent
func NewEinoAgent(
	ctx context.Context,
	chatModel model.BaseChatModel,
	ts []tools.Tool,
	mtr *metrics.Collector,
) (*EinoAgent, error) {
	if len(ts) == 0 {
		return nil, fmt.Errorf("at least one tool required")
	}

	// 把所有 deepin Tool 包装成 Eino InvokableTool
	einoTools := make([]tool.BaseTool, 0, len(ts))
	for _, t := range ts {
		adapter, err := tools.NewEinoAdapter(t)
		if err != nil {
			return nil, fmt.Errorf("failed to adapt tool %s: %w", t.Name(), err)
		}
		einoTools = append(einoTools, adapter)
	}

	// 构造 ChatModelAgent（ReAct 循环）
	systemPrompt := buildSystemPrompt(ts)
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "deepin_agent",
		Description: "deepin 25 桌面操作助手",
		Instruction: systemPrompt,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: einoTools,
			},
		},
		MaxIterations: 10, // 最多 10 轮 tool 调用
	})
	if err != nil {
		return nil, fmt.Errorf("NewChatModelAgent failed: %w", err)
	}

	// 构造 Runner（流式事件入口）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false, // MVP 阶段不开流式
	})

	return &EinoAgent{
		chatModel:  chatModel,
		einoTools:  einoTools,
		innerTools: ts,
		runner:     runner,
		mtr:        mtr,
	}, nil
}

// Run 执行用户指令，返回所有 tool 调用的 metrics
//
// 与手写 Agent.Run 不同：
//   - 手写版：基于关键词拆任务 → 顺序执行 → 收集结果
//   - Eino 版：把任务交给 LLM + ReAct loop → LLM 自动选 tool + 拆步骤
func (a *EinoAgent) Run(ctx context.Context, userInput string) error {
	fmt.Printf("[EinoAgent] 输入: %s\n", userInput)
	fmt.Printf("[EinoAgent] 注册 %d 个 tool\n", len(a.einoTools))

	// 用 Runner 执行
	iter := a.runner.Query(ctx, userInput)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		// 处理事件：把 tool 调用结果记录到 metrics
		a.handleEvent(event)
	}

	return nil
}

// handleEvent 处理 Runner 事件，提取 tool 调用信息写入 metrics
func (a *EinoAgent) handleEvent(event *adk.AgentEvent) {
	if event == nil || event.Output == nil {
		return
	}

	// 检查是否是 tool 调用结果
	msg, err := event.Output.MessageOutput.GetMessage()
	if err != nil {
		return
	}

	// 只记录 Tool 消息
	if msg.Role != schema.Tool {
		return
	}

	// 解析 tool response 找对应 tool
	for _, t := range a.innerTools {
		// Eino ToolMessage 包含 tool name + response
		if msg.ToolName == t.Name() {
			result := parseToolResult(msg.Content)
			a.mtr.Record(t.Name(), result)
			fmt.Printf("[EinoAgent] Tool: %s → %s\n",
				t.Name(), tools.FormatResult(result))
			return
		}
	}
}

// buildSystemPrompt 给 LLM 的系统提示词
func buildSystemPrompt(ts []tools.Tool) string {
	prompt := `你是 deepin 25 桌面操作助手。基于用户指令选择合适的 tool 执行。

可用 tools:
`
	for _, t := range ts {
		prompt += fmt.Sprintf("- %s: %s\n", t.Name(), t.Description())
	}

	prompt += `
执行规则：
1. 仔细分析用户指令
2. 选择合适的 tool（可能需要多个，按合理顺序）
3. 如果信息不足（如缺少路径），优先用默认值或合理猜测
4. 完成后用中文简短总结结果
`
	return prompt
}

// parseToolResult 从 Eino ToolMessage 反解出我们的 Result
func parseToolResult(content string) *tools.Result {
	// Eino 工具返回的是 JSON 字符串
	var r tools.Result
	if err := json.Unmarshal([]byte(content), &r); err != nil {
		// 解析失败：返回一个 fallback result
		return &tools.Result{
			Success:   false,
			Error:     "failed to parse tool response: " + err.Error(),
			ErrorType: "parse_error",
			Content:   content,
		}
	}
	return &r
}