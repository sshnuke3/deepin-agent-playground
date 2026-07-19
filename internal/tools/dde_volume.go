// Package tools - dde_volume.go
//
// v0.3 Phase 2 补:音量调节
//
// D-Bus: com.deepin.daemon.Audio
//   - SinkSetVolume(double volume)   // volume: 0.0 - 1.0
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

// 音量 D-Bus 常量
const (
	audioDest      = "com.deepin.daemon.Audio"
	audioPath      = "/com/deepin/daemon/Audio"
	audioInterface = "com.deepin.daemon.Audio"
	methodSinkSetVolume = audioInterface + ".SinkSetVolume"
	propertySinkVolume = audioInterface + ".sinkVolume" // 只读 property 用于验证
)

// DdeVolume 控制系统音量。
type DdeVolume struct {
	conn *dbus.Conn
}

// NewDdeVolume 创建 DDE 音量工具
func NewDdeVolume() (*DdeVolume, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session D-Bus: %w", err)
	}
	return &DdeVolume{conn: conn}, nil
}

func (d *DdeVolume) Name() string { return "dde_volume_set" }

func (d *DdeVolume) Description() string {
	return "设置系统音量。输入：volume（0-100 整数）"
}

func (d *DdeVolume) Run(ctx context.Context, input map[string]interface{}) (*Result, error) {
	start := time.Now()

	// 接受两种类型：int / string（因为 LLM 可能传字符串）
	var volume int
	switch v := input["volume"].(type) {
	case int:
		volume = v
	case int64:
		volume = int(v)
	case float64:
		volume = int(v)
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return &Result{
				Success: false, Error: fmt.Sprintf("invalid volume string: %q", v), ErrorType: "invalid_input",
				Duration: time.Since(start),
			}, nil
		}
		volume = n
	default:
		return &Result{
			Success: false, Error: fmt.Sprintf("volume must be int or string, got %T", input["volume"]), ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, nil
	}

	if volume < 0 || volume > 100 {
		return &Result{
			Success: false, Error: fmt.Sprintf("volume out of range: %d (must be 0-100)", volume), ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, nil
	}

	// 0-100 → 0.0-1.0
	ratio := float64(volume) / 100.0

	obj := d.conn.Object(audioDest, audioPath)
	err := obj.CallWithContext(ctx, methodSinkSetVolume, 0, ratio).Err
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: fmt.Sprintf("DDE SinkSetVolume failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// 验证：读 sinkVolume property
	verified := false
	current := -1
	if v, err := obj.GetProperty(propertySinkVolume); err == nil {
		// Audio 的 sinkVolume 返回 uint16（0-65535），需要再转 ratio 再 *100
		if u, ok := v.Value().(uint16); ok {
			current = int(float64(u) / 65535.0 * 100)
			verified = current == volume
		}
	}

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("volume set to %d", volume),
		Verified: verified,
		Meta: map[string]interface{}{
			"requested": volume,
			"current":   current,
		},
		Duration: time.Since(start),
	}, nil
}

func (d *DdeVolume) HealthCheck(ctx context.Context) error {
	if d.conn == nil {
		return fmt.Errorf("D-Bus connection not initialized")
	}
	obj := d.conn.Object(audioDest, audioPath)
	// 读 sinkVolume property 验证服务可用
	_, err := obj.GetProperty(propertySinkVolume)
	if err != nil {
		return fmt.Errorf("DDE Audio service unavailable: %w", err)
	}
	return nil
}

func (d *DdeVolume) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}