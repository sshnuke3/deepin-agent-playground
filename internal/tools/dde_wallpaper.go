// Package tools - dde_wallpaper.go
//
// v0.3 Phase 2 升级:壁纸(走 gdbus 命令 + CommandExecutor 可注入)
//
// 跟 v4 teams/internal/tools/dbus.go 风格一致,playground 自己也用同一套。
//
// D-Bus: com.deepin.daemon.Appearance
//   - Set(string key, string value) 例如 Set("background", "/path/to/wallpaper.jpg")
//   - Get(string key) -> string
//
// 验证：设置后 Get 对比请求值（机器可读的成功判定）。
package tools

import (
	"context"
	"fmt"
	"os"
	"time"
)

// DDE Wallpaper / Appearance D-Bus 常量
const (
	appearanceDest      = "com.deepin.daemon.Appearance"
	appearancePath      = "/com/deepin/daemon/Appearance"
	appearanceInterface = "com.deepin.daemon.Appearance"
	methodAppearanceSet = appearanceInterface + ".Set"
	methodAppearanceGet = appearanceInterface + ".Get"
)

// DdeWallpaper 设置 / 获取 DDE 桌面壁纸。
type DdeWallpaper struct{}

// NewDdeWallpaper 创建 DDE 壁纸工具
func NewDdeWallpaper() (*DdeWallpaper, error) {
	return &DdeWallpaper{}, nil
}

func (d *DdeWallpaper) Name() string { return "dde_wallpaper_set" }

func (d *DdeWallpaper) Description() string {
	return "设置 deepin DDE 桌面壁纸。输入：path（壁纸绝对路径）"
}

func (d *DdeWallpaper) Run(ctx context.Context, input map[string]interface{}) (*Result, error) {
	start := time.Now()

	// 1. 参数校验
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return &Result{
			Success: false, Error: "missing path", ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, fmt.Errorf("path required")
	}

	// 2. 检查文件存在
	if _, err := os.Stat(path); err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "not_found",
			Content: fmt.Sprintf("wallpaper file not found: %s", path),
			Duration: time.Since(start),
		}, nil
	}

	// 3. 调用 DDE Appearance.Set("background", path)
	_, err := dbusCall(ctx, appearanceDest, appearancePath, methodAppearanceSet, "background", path)
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: fmt.Sprintf("DDE Set failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 4. 验证：Get("background") 对比请求值
	currentOut, err := dbusCall(ctx, appearanceDest, appearancePath, methodAppearanceGet, "background")
	current := ""
	if err == nil {
		current = parseGVariantString(currentOut)
	}

	verified := current == path

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("wallpaper set to %s", path),
		Verified: verified,
		Meta: map[string]interface{}{
			"requested": path,
			"current":   current,
		},
		Duration: time.Since(start),
	}, nil
}

func (d *DdeWallpaper) HealthCheck(ctx context.Context) error {
	// GetProperty 兜底：尝试 Get current theme
	_, err := dbusCall(ctx, appearanceDest, appearancePath, methodAppearanceGet, "background")
	if err != nil {
		return fmt.Errorf("DDE Appearance service unavailable: %w", err)
	}
	return nil
}

// Close 兼容接口（godbus 时代的 API,现在 noop）
func (d *DdeWallpaper) Close() error { return nil }