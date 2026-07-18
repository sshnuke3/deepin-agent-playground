// Package agent 提供基于 Eino 的 agent 主循环。
//
// MVP v0.2：自己实现简单的 reasoning + acting 循环（不依赖完整 Eino ADK）。
// 后续接入 Eino Graph Checkpoint 做长程 option。
package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sshnuke3/daplayground/internal/metrics"
	"github.com/sshnuke3/daplayground/internal/tools"
)

// Agent 是最小可跑的 agent 主循环。
//
// 流程：
//   1. 把用户指令拆成 options（基于 prompt template + 关键词匹配）
//   2. 逐个执行 option，记录结果
//   3. 输出 metrics 报告
//
// 为什么不用 Eino ADK 完整版：MVP 阶段避免依赖复杂度，先验证 tool adapter 层。
// 后续接入：把 Reason 换成 ChatModel.Generate，把 Execute 换成 Eino Graph。
type Agent struct {
	tools  map[string]tools.Tool
	mtr    *metrics.Collector
}

// NewAgent 创建 agent 实例
func NewAgent(ts []tools.Tool, mtr *metrics.Collector) *Agent {
	a := &Agent{
		tools: make(map[string]tools.Tool),
		mtr:   mtr,
	}
	for _, t := range ts {
		a.tools[t.Name()] = t
	}
	return a
}

// Run 接收用户指令，执行 agent 主循环，返回执行结果列表
func (a *Agent) Run(ctx context.Context, userInput string) ([]*tools.Result, error) {
	// Step 1: 拆任务（简单关键词匹配 · MVP）
	options := a.parseIntent(userInput)
	if len(options) == 0 {
		return nil, fmt.Errorf("no actionable option detected in input: %q", userInput)
	}

	// Step 2: 逐个执行
	results := make([]*tools.Result, 0, len(options))
	for i, opt := range options {
		fmt.Printf("\n[Agent] 执行 option %d/%d: %s (%s)\n",
			i+1, len(options), opt.Tool, opt.Reason)

		tool, ok := a.tools[opt.Tool]
		if !ok {
			errResult := &tools.Result{
				Success:   false,
				Error:     fmt.Sprintf("tool not found: %s", opt.Tool),
				ErrorType: "not_found",
			}
			results = append(results, errResult)
			a.mtr.Record(opt.Tool, errResult)
			continue
		}

		// Health check before run
		if err := tool.HealthCheck(ctx); err != nil {
			errResult := &tools.Result{
				Success:   false,
				Error:     fmt.Sprintf("health check failed: %v", err),
				ErrorType: "unavailable",
			}
			results = append(results, errResult)
			a.mtr.Record(opt.Tool, errResult)
			continue
		}

		result, err := tool.Run(ctx, opt.Input)
		if err != nil {
			result = &tools.Result{
				Success: false,
				Error:   err.Error(),
			}
		}
		results = append(results, result)
		a.mtr.Record(opt.Tool, result)

		fmt.Printf("[Agent] 结果: %s\n", tools.FormatResult(result))

		// 失败：记录但继续（agent 不应该因单点失败全停）
		if !result.Success {
			fmt.Printf("[Agent] ⚠ option failed but continuing\n")
		}
	}

	return results, nil
}

// Option 是 agent 拆解出的一个执行单元
type Option struct {
	Tool   string
	Input  map[string]interface{}
	Reason string
}

// parseIntent 简单关键词匹配，把用户输入拆成 options
//
// MVP 实现：基于关键词触发固定 tool
// 后续接入：换成 Eino ChatModel + structured output
func (a *Agent) parseIntent(input string) []Option {
	input = strings.TrimSpace(input)
	var opts []Option
	low := strings.ToLower(input)

	// 关键词 1: 装/安装 + 包名
	if containsAny(low, "装", "install", "安装") {
		pkg := extractPackageID(input)
		if pkg != "" {
			opts = append(opts, Option{
				Tool:   "ll_cli_install",
				Input:  map[string]interface{}{"package_id": pkg},
				Reason: "用户要求安装应用",
			})
		}
	}

	// 关键词 2: 整理/分类/移动 + 目录
	if containsAny(low, "整理", "分类", "organize", "sort", "下载") {
		opts = append(opts, Option{
			Tool: "filesystem_move_by_ext",
			Input: map[string]interface{}{
				"src":        "~/Downloads",
				"dst":        "~/Documents/sorted",
				"extensions": []string{"jpg", "png", "pdf", "doc", "docx"},
			},
			Reason: "用户要求整理下载文件",
		})
	}

	// 关键词 3: 壁纸/wallpaper
	if containsAny(low, "壁纸", "wallpaper", "背景") {
		path := extractWallpaperPath(input)
		if path != "" {
			opts = append(opts, Option{
				Tool:   "dde_wallpaper_set",
				Input:  map[string]interface{}{"path": path},
				Reason: "用户要求更换壁纸",
			})
		}
	}

	return opts
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func extractPackageID(input string) string {
	// 简单提取：找形如 org.xxx.xxx 的字符串
	// 或包含"装 计算器"提取"计算器"
	imports := []string{
		"org.deepin.calculator", "org.deepin.editor", "org.deepin.files",
		"org.deepin.terminal", "org.deepin.screen-capture",
	}
	for _, id := range imports {
		if strings.Contains(input, id) {
			return id
		}
	}
	// 中文关键词映射
	cnMap := map[string]string{
		"计算器": "org.deepin.calculator",
		"编辑器": "org.deepin.editor",
		"文件管理器": "org.deepin.files",
		"终端": "org.deepin.terminal",
		"截图": "org.deepin.screen-capture",
	}
	for cn, id := range cnMap {
		if strings.Contains(input, cn) {
			return id
		}
	}
	return ""
}

func extractWallpaperPath(input string) string {
	// 简单实现：找 /usr/share/wallpapers 或 ~/Pictures 下的常见壁纸
	candidates := []string{
		"/usr/share/wallpapers/deepin/desktop.jpg",
		"/usr/share/wallpapers/deepin/castle.jpg",
		"~/Pictures/wallpaper.jpg",
		"/tmp/sunset.jpg",
	}
	for _, c := range candidates {
		if strings.Contains(input, c) {
			return c
		}
	}
	// 默认路径
	if strings.Contains(input, "日落") || strings.Contains(input, "sunset") {
		return "/tmp/sunset.jpg"
	}
	return "/usr/share/wallpapers/deepin/desktop.jpg"
}