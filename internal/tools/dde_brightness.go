// Package tools - dde_brightness.go
//
// v0.3 Phase 2 补:屏幕亮度调节
//
// D-Bus: com.deepin.daemon.Display
//   - SetBrightness(double brightness)   // brightness: 0.0 - 1.0
//
// 输入用 0-100 整数，内部转 0.0-1.0 ratio。
package tools

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/godbus/dbus/v5"
)

// 亮度 D-Bus 常量
// 注意：deepin 不同版本服务名可能不同（com.deepin.daemon.Display 或 .Power.Display）。
// 先写 Display，跟 dde_wallpaper 保持一致的 dest 风格；真机跑挂时再改 Power。
const (
	displayDest      = "com.deepin.daemon.Display"
	displayPath      = "/com/deepin/daemon/Display"
	displayInterface = "com.deepin.daemon.Display"
	methodSetBrightness = displayInterface + ".SetBrightness"
	propertyBrightness = displayInterface + ".Brightness" // 只读 property
)

// DdeBrightness 调节屏幕亮度。
type DdeBrightness struct {
	conn *dbus.Conn
}

// NewDdeBrightness 创建 DDE 亮度工具
func NewDdeBrightness() (*DdeBrightness, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session D-Bus: %w", err)
	}
	return &DdeBrightness{conn: conn}, nil
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

	ratio := float64(brightness) / 100.0

	obj := d.conn.Object(displayDest, displayPath)
	err := obj.CallWithContext(ctx, methodSetBrightness, 0, ratio).Err
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: fmt.Sprintf("DDE SetBrightness failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 验证：读 Brightness property
	verified := false
	current := -1
	if v, err := obj.GetProperty(propertyBrightness); err == nil {
		// Brightness 通常返回 double (0.0-1.0)
		if f, ok := v.Value().(float64); ok {
			current = int(f * 100)
			verified = current == brightness
		}
	}

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("brightness set to %d", brightness),
		Verified: verified,
		Meta: map[string]interface{}{
			"requested": brightness,
			"current":   current,
		},
		Duration: time.Since(start),
	}, nil
}

func (d *DdeBrightness) HealthCheck(ctx context.Context) error {
	if d.conn == nil {
		return fmt.Errorf("D-Bus connection not initialized")
	}
	obj := d.conn.Object(displayDest, displayPath)
	_, err := obj.GetProperty(propertyBrightness)
	if err != nil {
		return fmt.Errorf("DDE Display service unavailable: %w", err)
	}
	return nil
}

func (d *DdeBrightness) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}