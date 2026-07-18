// daplayground 入口
//
// Phase 1 MVP：演示 3 个 tool + 1 个 e2e 场景
// Phase 2 目标：接入 Eino Graph + 完整 ADK
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
		input     = flag.String("input", "", "用户指令（如'帮我装计算器，整理下载，换壁纸'）")
		dryRun    = flag.Bool("dry-run", false, "测试模式：跳过真实工具调用")
		timeout   = flag.Duration("timeout", 5*time.Minute, "总超时")
	)
	flag.Parse()

	cfg := config.Default()
	if *dryRun {
		cfg.DryRun = true
	}

	if *input == "" {
		// 默认 demo 输入
		*input = "帮我装计算器，整理 ~/Downloads 按扩展名分类，把壁纸换成日落"
		fmt.Printf("[main] 未指定 --input，使用默认 demo：\n  %s\n\n", *input)
	}

	fmt.Println("🚀 daplayground · Phase 1 MVP")
	fmt.Printf("   输入: %s\n", *input)
	fmt.Printf("   DryRun: %v\n", cfg.DryRun)
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

	// 4. 创建 + 运行 agent
	a := agent.NewAgent(toolList, mtr)
	results, err := a.Run(ctx, *input)
	if err != nil {
		log.Printf("[main] agent run error: %v", err)
	}

	// 5. 输出最终报告
	fmt.Println(mtr.Report())

	// 6. Exit code: 全部成功才算 0
	if len(results) > 0 {
		for _, r := range results {
			if !r.Success {
				os.Exit(1)
			}
		}
	}
}

// buildTools 初始化所有 tool
func buildTools(cfg *config.Config) ([]tools.Tool, error) {
	ts := []tools.Tool{
		tools.NewLlCliInstall(),
		tools.NewLlCliRun(),
		tools.NewFilesystemMove(),
	}

	// DDE Wallpaper 需要 D-Bus 连接
	if cfg.UseRealDBus && !cfg.DryRun {
		wp, err := tools.NewDdeWallpaper()
		if err != nil {
			return nil, fmt.Errorf("DDE wallpaper init failed: %w", err)
		}
		ts = append(ts, wp)
	} else {
		// Dry-run 或无 D-Bus：跳过 DDE tool
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