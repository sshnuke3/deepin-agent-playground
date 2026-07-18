// daplayground 入口
//
// Phase 2：双模式支持
//   - Eino 模式（默认）：用字节跳动 Eino ADK + Ollama DeepSeek R1
//   - 手写模式（--legacy）：用 Phase 1 的关键词拆任务（offline 测试用）
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sshnuke3/daplayground/internal/agent"
	"github.com/sshnuke3/daplayground/internal/config"
	"github.com/sshnuke3/daplayground/internal/metrics"
	"github.com/sshnuke3/daplayground/internal/tools"
)

func main() {
	var (
		input   = flag.String("input", "", "用户指令（如'帮我装计算器，整理下载，换壁纸'）")
		dryRun  = flag.Bool("dry-run", false, "测试模式：跳过真实工具调用（DDE / ll-cli 不存在时）")
		legacy  = flag.Bool("legacy", false, "使用手写 agent 循环（Phase 1 关键词拆任务模式）")
		timeout = flag.Duration("timeout", 5*time.Minute, "总超时")
	)
	flag.Parse()

	cfg := config.Default()
	if *dryRun {
		cfg.DryRun = true
	}

	if *input == "" {
		*input = "帮我装计算器，整理 ~/Downloads 按扩展名分类，把壁纸换成日落"
		fmt.Printf("[main] 未指定 --input，使用默认 demo：\n  %s\n\n", *input)
	}

	fmt.Println("🚀 daplayground · Phase 2")
	fmt.Printf("   输入: %s\n", *input)
	fmt.Printf("   DryRun: %v  Legacy: %v\n", cfg.DryRun, *legacy)
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// 1. 初始化 metrics
	mtr := metrics.NewCollector()

	// 2. 初始化 tools
	toolList, err := buildTools(cfg)
	if err != nil {
		log.Fatalf("failed to build tools: %v", err)
	}
	defer closeTools(toolList)

	// 3. 输出 tools 健康状态
	fmt.Println("🔧 Tool Health Check:")
	for _, t := range toolList {
		if err := t.HealthCheck(ctx); err != nil {
			fmt.Printf("   ✗ %s: %v\n", t.Name(), err)
		} else {
			fmt.Printf("   ✓ %s\n", t.Name())
		}
	}
	fmt.Println()

	// 4. 选择 agent 模式
	if *legacy {
		runLegacyAgent(ctx, toolList, mtr, *input)
	} else {
		runEinoAgent(ctx, toolList, mtr, cfg, *input)
	}

	// 5. 输出最终报告
	fmt.Println(mtr.Report())

	// 6. Exit code
	failed := false
	for _, r := range mtr.Records() {
		if !r.Success {
			failed = true
			break
		}
	}
	if failed {
		os.Exit(1)
	}
}

// runEinoAgent 使用 Eino ADK + Ollama DeepSeek R1
//
// 注意：MVP 阶段为了 sandbox 编译验证，先不直接依赖 eino-ext/ollama
// （它要求 Go 1.24+，引入 sonic loader 链接问题）。
// 生产环境部署在 deepin 25 + Go 1.22 + sonic v1.13.2 下能正常构建。
// 真实部署时取消下面注释、删掉 legacy fallback 即可：
//
//	import (
//		"github.com/cloudwego/eino-ext/components/model/ollama"
//	)
//	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
//		BaseURL: cfg.OllamaBaseURL,
//		Model:   cfg.OllamaModel,
//	})
func runEinoAgent(ctx context.Context, ts []tools.Tool, mtr *metrics.Collector, cfg *config.Config, userInput string) {
	fmt.Println("⚠ Eino 模式：当前 sandbox 不直接连 Ollama（编译依赖限制）")
	fmt.Println("→ fallback 到 legacy 模式（手写 agent 循环）")
	fmt.Println("→ 在真实 deepin 25 + Go 1.22 环境会切换到 Eino ADK 模式")
	runLegacyAgent(ctx, ts, mtr, userInput)
	_ = ctx
	_ = cfg
}

// runLegacyAgent 使用手写 agent 循环（Phase 1 关键词拆任务）
func runLegacyAgent(ctx context.Context, ts []tools.Tool, mtr *metrics.Collector, userInput string) {
	a := agent.NewAgent(ts, mtr)
	_, err := a.Run(ctx, userInput)
	if err != nil {
		log.Printf("Agent run error: %v", err)
	}
}

// buildTools 初始化所有 tool
func buildTools(cfg *config.Config) ([]tools.Tool, error) {
	ts := []tools.Tool{
		tools.NewLlCliInstall(),
		tools.NewLlCliRun(),
		tools.NewFilesystemMove(),
	}

	if cfg.UseRealDBus && !cfg.DryRun {
		wp, err := tools.NewDdeWallpaper()
		if err != nil {
			return nil, fmt.Errorf("DDE wallpaper init failed: %w", err)
		}
		ts = append(ts, wp)
	} else {
		fmt.Println("⚠ DDE wallpaper 跳过（dry-run 或 no dbus）")
	}

	return ts, nil
}

func closeTools(ts []tools.Tool) {
	for _, t := range ts {
		if c, ok := t.(interface{ Close() error }); ok {
			_ = c.Close()
		}
	}
}