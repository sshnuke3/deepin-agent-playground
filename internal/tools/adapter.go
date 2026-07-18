//go:build eino

// Package tools/adapter 提供我们的 Tool 接口到 Eino InvokableTool 的桥接。
//
// 这层适配让所有 deepin 工具既能：
//   - 被手写 agent 循环直接调用（tools.Tool.Run）
//   - 被 Eino ADK ChatModelAgent 注册（tool.InvokableTool.InvokableRun）
//
// 桥接逻辑：
//   - Info() → schema.ToolInfo（含 Name / Desc / 参数 schema）
//   - InvokableRun(argsJSON) → 解码 JSON → 调用 Run() → 编码 Result 回 string
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// EinoAdapter 把我们的 Tool 包装成 Eino 的 InvokableTool
type EinoAdapter struct {
	inner    Tool
	toolInfo *schema.ToolInfo
}

// NewEinoAdapter 创建 Eino 适配器
func NewEinoAdapter(t Tool) (*EinoAdapter, error) {
	params, err := buildParamsSchema(t)
	if err != nil {
		return nil, fmt.Errorf("failed to build params schema for %s: %w", t.Name(), err)
	}

	return &EinoAdapter{
		inner: t,
		toolInfo: &schema.ToolInfo{
			Name:        t.Name(),
			Desc:        t.Description(),
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		},
	}, nil
}

// Info 返回 Eino 标准的 tool 元信息
func (a *EinoAdapter) Info(_ context.Context) (*schema.ToolInfo, error) {
	return a.toolInfo, nil
}

// InvokableRun 接收 JSON 参数，调用内部 Tool.Run，返回 JSON 字符串结果
func (a *EinoAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	// 1. 解码 JSON 参数
	var input map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("invalid JSON arguments: %w", err)
	}

	// 2. 调用内部 Tool
	result, err := a.inner.Run(ctx, input)
	if err != nil {
		// 返回错误信息给 LLM（不 panic）
		errJSON, _ := json.Marshal(map[string]interface{}{
			"error":      err.Error(),
			"error_type": "execution_error",
		})
		return string(errJSON), nil
	}

	// 3. 编码 Result 回 JSON 字符串
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(out), nil
}

// Inner 返回被包装的内部 Tool（用于混合场景：手动调用 + Eino Agent）
func (a *EinoAdapter) Inner() Tool {
	return a.inner
}

// buildParamsSchema 为不同 tool 构造对应的 ParameterInfo 映射
//
// 使用 Eino 原生的 schema.NewParamsOneOfByParams（不需要 jsonschema 包）
func buildParamsSchema(t Tool) (map[string]*schema.ParameterInfo, error) {
	switch t.Name() {
	case "ll_cli_install":
		return map[string]*schema.ParameterInfo{
			"package_id": {
				Type:     schema.String,
				Desc:     "玲珑包 ID，如 org.deepin.calculator",
				Required: true,
			},
		}, nil

	case "ll_cli_run":
		return map[string]*schema.ParameterInfo{
			"package_id": {
				Type:     schema.String,
				Desc:     "玲珑包 ID",
				Required: true,
			},
		}, nil

	case "filesystem_move_by_ext":
		return map[string]*schema.ParameterInfo{
			"src": {
				Type:     schema.String,
				Desc:     "源目录绝对路径",
				Required: true,
			},
			"dst": {
				Type:     schema.String,
				Desc:     "目标目录绝对路径",
				Required: true,
			},
			"extensions": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
				Desc:     "要整理的扩展名列表，如 [\"jpg\", \"png\"]",
				Required: true,
			},
		}, nil

	case "dde_wallpaper_set":
		return map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "壁纸绝对路径",
				Required: true,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", t.Name())
	}
}