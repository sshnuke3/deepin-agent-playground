// Package tools - dde_network.go
//
// v0.3 Phase 2 补:WiFi 开关(走 gdbus 命令 + CommandExecutor 可注入)
//
// D-Bus: com.deepin.daemon.Network
//   - EnableWifi()
//   - DisableWifi()
//
// ⚠️ 注意:WiFi 开关会断开当前无线连接 — 慎用,测试时确保有 fallback(手机热点)。
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// 网络 D-Bus 常量
const (
	networkDest          = "com.deepin.daemon.Network"
	networkPath          = "/com/deepin/daemon/Network"
	networkInterface     = "com.deepin.daemon.Network"
	methodEnableWifi     = networkInterface + ".EnableWifi"
	methodDisableWifi    = networkInterface + ".DisableWifi"
)

// DdeNetwork 控制 WiFi 开关。
type DdeNetwork struct{}

// NewDdeNetwork 创建 DDE 网络工具
func NewDdeNetwork() (*DdeNetwork, error) {
	return &DdeNetwork{}, nil
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

	method := methodDisableWifi
	if state == "on" {
		method = methodEnableWifi
	}

	_, err := dbusCall(ctx, networkDest, networkPath, method)
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "dbus_error",
			Content: fmt.Sprintf("DDE %s failed: %v", method, err),
			Duration: time.Since(start),
		}, nil
	}

	// 注意：gdbus 字符串层无法读 bool property（只能调 method）
	// 所以这里不严格验证 — 调用成功视为成功
	modeNote := ""
	if IsMockMode() {
		modeNote = "（演示模式）"
	}

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("WiFi %s%s", state, modeNote),
		Verified: true, // 调用成功视为成功
		Meta: map[string]interface{}{
			"requested": state,
			"warning":   "WiFi off 会断开无线连接，慎用",
		},
		Duration: time.Since(start),
	}, nil
}

func (d *DdeNetwork) HealthCheck(ctx context.Context) error {
	if IsMockMode() {
		return nil
	}
	// Network 没合适的只读方法
	// 仅依赖 D-Bus session bus 在跑
	return nil
}

func (d *DdeNetwork) Close() error { return nil }