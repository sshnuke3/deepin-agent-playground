// Package tools - dde_theme.go
//
// v0.3 Phase 2 补:主题切换（深色/浅色/自动）
//
// D-Bus: com.deepin.daemon.Appearance
//   - SetCurrentTheme(theme)
//   - GetCurrentTheme()
//
// 跟 teams/internal/tools/appearance.go 对齐接口，但走 godbus/dbus（playground 自己的风格）。
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

// D-Bus 服务常量（与 dde_wallpaper.go 一致）
const (
	methodSetCurrentTheme = appearanceInterface + ".SetCurrentTheme"
	methodGetCurrentTheme = appearanceInterface + ".GetCurrentTheme"
)

// DdeTheme 切换 DDE 主题。
type DdeTheme struct {
	conn *dbus.Conn
}

// NewDdeTheme 创建 DDE 主题工具
func NewDdeTheme() (*DdeTheme, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session D-Bus: %w", err)
	}
	return &DdeTheme{conn: conn}, nil
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

	// 校验合法主题名（防止无效值传 D-Bus）
	switch theme {
	case "deepin-dark", "deepin-light", "deepin-auto":
		// OK
	default:
		return &Result{
			Success: false, Error: fmt.Sprintf("invalid theme: %q", theme), ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, nil
	}

	obj := d.conn.Object(appearanceDest, appearancePath)
	err := obj.CallWithContext(ctx, methodSetCurrentTheme, 0, theme).Err
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: fmt.Sprintf("DDE SetCurrentTheme failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 验证：调 GetCurrentTheme 读回对比
	call := obj.CallWithContext(ctx, methodGetCurrentTheme, 0)
	current := ""
	if call.Err == nil && len(call.Body) > 0 {
		if s, ok := call.Body[0].(string); ok {
			current = s
		}
	}
	verified := current == theme

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("theme set to %s", theme),
		Verified: verified,
		Meta: map[string]interface{}{
			"requested": theme,
			"current":   current,
		},
		Duration: time.Since(start),
	}, nil
}

func (d *DdeTheme) HealthCheck(ctx context.Context) error {
	if d.conn == nil {
		return fmt.Errorf("D-Bus connection not initialized")
	}
	obj := d.conn.Object(appearanceDest, appearancePath)
	err := obj.CallWithContext(ctx, methodGetCurrentTheme, 0).Err
	if err != nil {
		return fmt.Errorf("DDE Appearance service unavailable: %w", err)
	}
	return nil
}

// Close 关闭 D-Bus 连接
func (d *DdeTheme) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}