// Package tools - dde_network.go
//
// v0.3 Phase 2 补:WiFi 开关
//
// D-Bus: com.deepin.daemon.Network
//   - EnableWifi()
//   - DisableWifi()
//   - IsWifiEnabled() -> bool (用于验证)
//
// ⚠️ 注意：WiFi 开关会断开当前无线连接 — 慎用，测试时确保有 fallback（手机热点）。
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// 网络 D-Bus 常量
const (
	networkDest      = "com.deepin.daemon.Network"
	networkPath      = "/com/deepin/daemon/Network"
	networkInterface = "com.deepin.daemon.Network"
	methodEnableWifi  = networkInterface + ".EnableWifi"
	methodDisableWifi = networkInterface + ".DisableWifi"
	propertyWifiEnabled = networkInterface + ".WirelessEnabled" // 只读 property
)

// DdeNetwork 控制 WiFi 开关。
type DdeNetwork struct {
	conn *dbus.Conn
}

// NewDdeNetwork 创建 DDE 网络工具
func NewDdeNetwork() (*DdeNetwork, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session D-Bus: %w", err)
	}
	return &DdeNetwork{conn: conn}, nil
}

func (d *DdeNetwork) Name() string { return "dde_network_wifi_toggle" }

func (d *DdeNetwork) Description() string {
	return "开关 WiFi。输入：state（on / off）"
}

func (d *DdeNetwork) Run(ctx context.Context, input map[string]interface{}) (*Result, error) {
	start := time.Now()

	state, ok := input["state"].(string)
	if !ok || state == "" {
		return &Result{
			Success: false, Error: "missing state", ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, fmt.Errorf("state required (on / off)")
	}

	state = strings.ToLower(strings.TrimSpace(state))
	switch state {
	case "on":
		// OK
	case "off":
		// OK
	default:
		return &Result{
			Success: false, Error: fmt.Sprintf("invalid state: %q (must be on / off)", state), ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, nil
	}

	obj := d.conn.Object(networkDest, networkPath)

	method := methodDisableWifi
	if state == "on" {
		method = methodEnableWifi
	}

	err := obj.CallWithContext(ctx, method, 0).Err
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: fmt.Sprintf("DDE %s failed: %v", method, err),
			Duration: time.Since(start),
		}, nil
	}

	// 验证：读 WirelessEnabled property
	verified := false
	current := ""
	if v, err := obj.GetProperty(propertyWifiEnabled); err == nil {
		if b, ok := v.Value().(bool); ok {
			current = "off"
			if b {
				current = "on"
			}
			verified = current == state
		}
	}

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("WiFi %s", state),
		Verified: verified,
		Meta: map[string]interface{}{
			"requested": state,
			"current":   current,
			"warning":   "WiFi off 会断开无线连接，慎用",
		},
		Duration: time.Since(start),
	}, nil
}

func (d *DdeNetwork) HealthCheck(ctx context.Context) error {
	if d.conn == nil {
		return fmt.Errorf("D-Bus connection not initialized")
	}
	obj := d.conn.Object(networkDest, networkPath)
	_, err := obj.GetProperty(propertyWifiEnabled)
	if err != nil {
		return fmt.Errorf("DDE Network service unavailable: %w", err)
	}
	return nil
}

func (d *DdeNetwork) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}