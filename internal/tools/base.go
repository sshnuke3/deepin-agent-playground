// Package tools 提供 deepin 25 上的 tool 适配器实现。
//
// 所有 tool 实现 eino 的 Tool 接口，可被 Eino Graph / Agent 直接识别。
// 这是 Sutton 路线第三步"option 学习"的基础设施层。
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Result 是所有 tool 调用的统一返回结构。
//
// 关键设计：环境反馈是机器可读的（success / exit_code / verified），
// 不依赖人类评分——这正是 Sutton option 学习最稀缺的"可验证奖励信号"。
type Result struct {
	Success   bool                   `json:"success"`
	ExitCode  int                    `json:"exit_code"`
	Content   string                 `json:"content"`
	Error     string                 `json:"error,omitempty"`
	ErrorType string                 `json:"error_type,omitempty"`
	Verified  bool                   `json:"verified"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
	Duration  time.Duration          `json:"duration_ms"`
}

// Tool 是所有 deepin 工具适配器必须实现的接口。
//
// 简化版 Eino Tool 接口（v0.2 MVP）—— 不引 schema 依赖，
// 等后续接入 Eino Agent 时再加 schema.ToolInfo。
type Tool interface {
	// Name 返回工具名（英文 snake_case）
	Name() string

	// Description 返回工具描述（给 LLM 看的）
	Description() string

	// Run 执行工具调用
	//   ctx: 上下文（支持取消 / 超时）
	//   input: 输入参数 map[string]interface{}
	// 返回: Result + error
	Run(ctx context.Context, input map[string]interface{}) (*Result, error)

	// HealthCheck 自检状态（工具是否可用）
	HealthCheck(ctx context.Context) error
}

// ClassifyError 根据 exit code 和 stderr 分类错误
func ClassifyError(exitCode int, stderr string) string {
	if exitCode == 0 {
		return ""
	}
	low := strings.ToLower(stderr)
	switch {
	case contains(low, "permission denied", "not permitted"):
		return "permission"
	case contains(low, "not found", "no such"):
		return "not_found"
	case contains(low, "network", "timeout", "connection"):
		return "network"
	case contains(low, "busy", "locked"):
		return "busy"
	default:
		return "unknown"
	}
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) == 0 {
			continue
		}
		if indexOf(s, sub) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// FormatResult 格式化 Result 给日志/调试用
func FormatResult(r *Result) string {
	status := "✗"
	if r.Success && r.Verified {
		status = "✓"
	} else if r.Success {
		status = "?"
	}
	return fmt.Sprintf("[%s] %s exit=%d verified=%v duration=%s error=%q",
		status, r.Content, r.ExitCode, r.Verified, r.Duration, r.Error)
}

// ToSchemaToolInfo 把 Tool 转成 eino schema（后续接入 Eino Agent 时启用）
// MVP 阶段不依赖 schema 包，避免额外依赖
// func ToSchemaToolInfo(t Tool) *schema.ToolInfo {
// 	return &schema.ToolInfo{
// 		Name: t.Name(),
// 		Desc: t.Description(),
// 	}
// }