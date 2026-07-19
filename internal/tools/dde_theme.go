// Package tools - dde_theme.go
//
// v0.3 Phase 2 补:主题切换(走 gdbus 命令 + CommandExecutor 可注入)
//
// D-Bus: com.deepin.daemon.Appearance
//   - SetGtkTheme(string theme)
//   - GetCurrentTheme() -> string
package tools

import (
	"context"
	"fmt"
	"time"
)

// 主题相关常量
const (
	methodSetGtkTheme    = appearanceInterface + ".SetGtkTheme"
	methodGetTheme       = appearanceInterface + ".GetCurrentTheme"
	methodSetCurrentTheme = appearanceInterface + ".SetCurrentTheme"
)

// 合法主题名
var validThemes = map[string]bool{
	"deepin-dark":  true,
	"deepin-light": true,
	"deepin-auto":  true,
}

// DdeTheme 切换 DDE 主题。
type DdeTheme struct{}

// NewDdeTheme 创建 DDE 主题工具
func NewDdeTheme() (*DdeTheme, error) {
	return &DdeTheme{}, nil
}

func (d *DdeTheme) Name() string { return "dde_theme_set" }

func (d *DdeTheme) Description() string {
	return "设置 deepin DDE 主题。输入：theme（deepin-dark / deepin-light / deepin-auto）"
}

func (d *DdeTheme) Run(ctx context.Context, input map[string]interface{}) (*Result, error) {
	start := time.Now()

	theme, ok := input["theme"].(string)
	if !ok || theme == "" {
		return &Result{
			Success: false, Error: "missing theme", ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, fmt.Errorf("theme required")
	}

	if !validThemes[theme] {
		return &Result{
			Success: false, Error: fmt.Sprintf("invalid theme: %q", theme), ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, nil
	}

	// 调用 SetGtkTheme（跟 v4 teams 保持一致 — 优先用 SetGtkTheme）
	_, err := dbusCall(ctx, appearanceDest, appearancePath, methodSetGtkTheme, theme)
	if err != nil {
		// 失败时回退 SetCurrentTheme（某些 deepin 版本只有这个）
		_, err2 := dbusCall(ctx, appearanceDest, appearancePath, methodSetCurrentTheme, theme)
		if err2 != nil {
			return &Result{
				Success: false, Error: err.Error(), ErrorType: "dbus_error",
				Content: fmt.Sprintf("DDE SetGtkTheme failed: %v (also tried SetCurrentTheme: %v)", err, err2),
				Duration: time.Since(start),
			}, nil
		}
	}

	// 验证：GetCurrentTheme 读回对比
	currentOut, err := dbusCall(ctx, appearanceDest, appearancePath, methodGetTheme)
	current := ""
	if err == nil {
		current = parseGVariantString(currentOut)
	}
	verified := current == theme

	modeNote := ""
	if IsMockMode() {
		modeNote = "（演示模式）"
	}

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("theme set to %s%s", theme, modeNote),
		Verified: verified,
		Meta: map[string]interface{}{
			"requested": theme,
			"current":   current,
		},
		Duration: time.Since(start),
	}, nil
}

func (d *DdeTheme) HealthCheck(ctx context.Context) error {
	_, err := dbusCall(ctx, appearanceDest, appearancePath, methodGetTheme)
	if err != nil {
		return fmt.Errorf("DDE Appearance service unavailable: %w", err)
	}
	return nil
}

func (d *DdeTheme) Close() error { return nil }