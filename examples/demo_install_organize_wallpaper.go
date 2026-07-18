// examples/demo_install_organize_wallpaper.go
//
// 演示场景：装计算器 + 整理下载 + 换壁纸
// 这是 Sutton 第三步 option 学习的典型场景。
//
// 用法：
//   go run ./examples/demo_install_organize_wallpaper.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sshnuke3/daplayground/internal/agent"
	"github.com/sshnuke3/daplayground/internal/metrics"
	"github.com/sshnuke3/daplayground/internal/tools"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	mtr := metrics.NewCollector()

	// 构建 tool 列表
	ts := []tools.Tool{
		tools.NewLlCliInstall(),
		tools.NewFilesystemMove(),
	}

	wp, err := tools.NewDdeWallpaper()
	if err != nil {
		log.Printf("⚠ DDE unavailable: %v · 跳过壁纸步骤", err)
	} else {
		ts = append(ts, wp)
		defer wp.Close()
	}

	// 输出 user input
	userInput := "帮我装计算器，整理 ~/Downloads 按扩展名分类，把壁纸换成日落"
	fmt.Println("============================================================")
	fmt.Println("🎬 demo: 装计算器 + 整理下载 + 换壁纸")
	fmt.Println("============================================================")
	fmt.Printf("输入: %s\n\n", userInput)

	// Health check
	fmt.Println("Health check:")
	for _, t := range ts {
		if err := t.HealthCheck(ctx); err != nil {
			fmt.Printf("  ✗ %s: %v\n", t.Name(), err)
		} else {
			fmt.Printf("  ✓ %s\n", t.Name())
		}
	}
	fmt.Println()

	// Run agent
	a := agent.NewAgent(ts, mtr)
	_, err = a.Run(ctx, userInput)
	if err != nil {
		log.Printf("agent error: %v", err)
	}

	// Final report
	fmt.Println(mtr.Report())
}