package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

// DDE Wallpaper / Appearance D-Bus 服务常量
const (
	appearanceDest      = "com.deepin.daemon.Appearance"
	appearancePath      = "/com/deepin/daemon/Appearance"
	appearanceInterface = "com.deepin.daemon.Appearance"
	propertyBackground  = appearanceInterface + ".background"
	methodSet           = appearanceInterface + ".Set"
	methodGet           = appearanceInterface + ".Get"
)

// DdeWallpaper 设置 / 获取 DDE 桌面壁纸。
//
// 通过 D-Bus 调用 com.deepin.daemon.Appearance 服务（与 deepin 团队同款库）。
// 验证：设置后读取当前壁纸，对比请求值——机器可读的成功判定。
type DdeWallpaper struct {
	conn *dbus.Conn
}

// NewDdeWallpaper 创建 DDE 壁纸工具
func NewDdeWallpaper() (*DdeWallpaper, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session D-Bus: %w", err)
	}
	return &DdeWallpaper{conn: conn}, nil
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

	// 3. 调用 DDE Appearance.Set
	obj := d.conn.Object(appearanceDest, appearancePath)
	err := obj.CallWithContext(ctx, methodSet, 0, "background", path).Err
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: fmt.Sprintf("DDE Set failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 4. 验证：读取当前壁纸对比
	currentVariant, err := obj.GetProperty(propertyBackground)
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: "DDE Get failed",
			Meta: map[string]interface{}{
				"requested": path,
			},
			Duration: time.Since(start),
		}, nil
	}

	current, _ := currentVariant.Value().(string)
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
	// 尝试获取当前壁纸（只读操作）
	if d.conn == nil {
		return fmt.Errorf("D-Bus connection not initialized")
	}
	obj := d.conn.Object(appearanceDest, appearancePath)
	_, err := obj.GetProperty(propertyBackground)
	if err != nil {
		return fmt.Errorf("DDE Appearance service unavailable: %w", err)
	}
	return nil
}

// Close 关闭 D-Bus 连接
func (d *DdeWallpaper) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}