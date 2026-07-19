// Package tools - dde_brightness.go
//
// v0.3 Phase 2 补:屏幕亮度调节(走 gdbus 命令 + CommandExecutor 可注入)
//
// D-Bus: com.deepin.daemon.Display
//   - Brightness.SetBrightness(double brightness)   // 0.0 - 1.0
//
// 输入用 0-100 整数,内部转 0.0-1.0 ratio。
package tools

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// 亮度 D-Bus 常量
const (
	displayDest           = "com.deepin.daemon.Display"
	displayPath           = "/com/deepin/daemon/Display"
	displayInterface      = "com.deepin.daemon.Display"
	brightnessSubiface    = "Brightness"
	methodSetBrightness   = displayInterface + "." + brightnessSubiface + ".SetBrightness"
	methodGetBrightness   = displayInterface + "." + brightnessSubiface + ".GetBrightness"
)

// DdeBrightness 调节屏幕亮度。
type DdeBrightness struct{}

// NewDdeBrightness 创建 DDE 亮度工具
func NewDdeBrightness() (*DdeBrightness, error) {
	return &DdeBrightness{}, nil
}

func (d *DdeBrightness) Name() string { return "dde_brightness_set" }

func (d *DdeBrightness) Description() string {
	return "设置屏幕亮度。输入：brightness（0-100 整数）"
}

func (d *DdeBrightness) Run(ctx context.Context, input map[string]interface{}) (*Result, error) {
	start := time.Now()

	// 接受多种类型
	var brightness int
	switch v := input["brightness"].(type) {
	case int:
		brightness = v
	case int64:
		brightness = int(v)
	case float64:
		brightness = int(v)
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return &Result{
				Success: false, Error: fmt.Sprintf("invalid brightness string: %q", v), ErrorType: "invalid_input",
				Duration: time.Since(start),
			}, nil
		}
		brightness = n
	default:
		return &Result{
			Success: false, Error: fmt.Sprintf("brightness must be int or string, got %T", input["brightness"]), ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, nil
	}

	if brightness < 0 || brightness > 100 {
		return &Result{
			Success: false, Error: fmt.Sprintf("brightness out of range: %d (must be 0-100)", brightness), ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, nil
	}

	ratio := fmt.Sprintf("%f", float64(brightness)/100.0)

	_, err := dbusCall(ctx, displayDest, displayPath, methodSetBrightness, ratio)
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: fmt.Sprintf("DDE SetBrightness failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 验证：GetBrightness 读回
	currentOut, err := dbusCall(ctx, displayDest, displayPath, methodGetBrightness)
	current := -1
	if err == nil {
		// 假设返回 double（0.0-1.0），可能格式：(0.500000,)
		if f := parseDoubleFromGVariant(currentOut); f >= 0 {
			current = int(f * 100)
		}
	}
	verified := current == brightness

	modeNote := ""
	if IsMockMode() {
		modeNote = "（演示模式）"
	}

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("brightness set to %d%s", brightness, modeNote),
		Verified: verified,
		Meta: map[string]interface{}{
			"requested": brightness,
			"ratio":     ratio,
			"current":   current,
		},
		Duration: time.Since(start),
	}, nil
}

func (d *DdeBrightness) HealthCheck(ctx context.Context) error {
	if IsMockMode() {
		return nil
	}
	// GetBrightness 是只读,可以用来验证服务可用
	_, err := dbusCall(context.Background(), displayDest, displayPath, methodGetBrightness)
	if err != nil {
		return fmt.Errorf("DDE Display service unavailable: %w", err)
	}
	return nil
}

func (d *DdeBrightness) Close() error { return nil }