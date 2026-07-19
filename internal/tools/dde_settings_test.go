// Package tools - dde_settings_test.go
//
// v0.3 Phase 2 升级:4 个 DDE tool 的单元测试(走 fakeExecutor)
//
// 测试策略（v0.3 升级后）：
//   - mock 模式（DEEPIN_DBUS=mock）：所有 dbusCall 不调 gdbus,直接返回 mockDBusResponse
//   - 注入 fakeExecutor：验证调对了 gdbus 参数名 / 接口签名 / 参数值
//   - Ubuntu/CI 上能跑（不再 skip）
//
// 验证场景：
//   1. 参数校验（缺参 / 类型错 / 越界 / 非法值）
//   2. 真 D-Bus 路径（验证调到了正确的 dest/path/method/args）
//   3. 错误处理（gdbus 失败 → Result 标记 failed）
package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// === fakeExecutor + 工具函数在 dbus_test.go 定义 ===

// === DdeTheme 测试 ===

func TestDdeTheme_InvalidTheme(t *testing.T) {
	d, _ := NewDdeTheme()

	tests := []struct {
		name      string
		theme     string
		wantError bool
	}{
		{"empty", "", true}, // missing -> error
		{"unknown", "purple", false},
		{"partial", "deepin", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := d.Run(context.Background(), map[string]interface{}{"theme": tt.theme})
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error for missing theme %q", tt.theme)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run should not return error: %v", err)
			}
			if res.Success {
				t.Errorf("expected Success=false for invalid theme %q", tt.theme)
			}
			if res.ErrorType != "invalid_input" {
				t.Errorf("expected ErrorType=invalid_input, got %q", res.ErrorType)
			}
		})
	}
}

func TestDdeTheme_RealDBus_CallsSetGtkTheme(t *testing.T) {
	// 注入 fakeExecutor 验证真 D-Bus 路径
	fake := &fakeExecutor{stdout: "()"}
	SetExecutor(fake)
	defer ResetExecutor()
	t.Setenv("DEEPIN_DBUS", "")

	d, _ := NewDdeTheme()
	res, err := d.Run(context.Background(), map[string]interface{}{"theme": "deepin-dark"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Errorf("expected Success=true, got %+v", res)
	}

	// 验证调用参数
	if len(fake.calls) == 0 {
		t.Fatal("expected at least one gdbus call")
	}
	call := fake.calls[0]

	// 第一个调用应该是 SetGtkTheme
	if call.name != "gdbus" {
		t.Errorf("expected command 'gdbus', got %q", call.name)
	}
	hasSetGtk := false
	hasDeepinDark := false
	for _, a := range call.args {
		if a == "com.deepin.daemon.Appearance.SetGtkTheme" {
			hasSetGtk = true
		}
		if a == "deepin-dark" {
			hasDeepinDark = true
		}
	}
	if !hasSetGtk {
		t.Errorf("expected args to contain SetGtkTheme, got %v", call.args)
	}
	if !hasDeepinDark {
		t.Errorf("expected args to contain 'deepin-dark', got %v", call.args)
	}
}

func TestDdeTheme_RealDBus_GdbusFailureReturnsFailed(t *testing.T) {
	// fakeExecutor 返回错误（模拟 gdbus 调用失败）
	fake := &fakeExecutor{stdout: "", stderr: "ServiceUnknown", err: errors.New("exit 1")}
	SetExecutor(fake)
	defer ResetExecutor()

	d, _ := NewDdeTheme()
	res, _ := d.Run(context.Background(), map[string]interface{}{"theme": "deepin-dark"})

	if res.Success {
		t.Error("expected Success=false when gdbus fails")
	}
	if res.ErrorType != "dbus_error" {
		t.Errorf("expected ErrorType=dbus_error, got %q", res.ErrorType)
	}
	if !strings.Contains(res.Content, "SetGtkTheme failed") {
		t.Errorf("expected error message to mention SetGtkTheme, got %q", res.Content)
	}
}

// === DdeVolume 测试 ===

func TestDdeVolume_OutOfRange(t *testing.T) {
	d, _ := NewDdeVolume()

	tests := []struct {
		name   string
		volume interface{}
	}{
		{"negative_int", -10},
		{"too_large_int", 150},
		{"negative_string", "-5"},
		{"too_large_string", "200"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := d.Run(context.Background(), map[string]interface{}{"volume": tt.volume})
			if err != nil {
				t.Fatalf("Run should not return error: %v", err)
			}
			if res.Success {
				t.Errorf("expected Success=false for volume %v", tt.volume)
			}
		})
	}
}

func TestDdeVolume_RealDBus_ConvertsToRatio(t *testing.T) {
	fake := &fakeExecutor{stdout: "()"}
	SetExecutor(fake)
	defer ResetExecutor()

	d, _ := NewDdeVolume()
	res, err := d.Run(context.Background(), map[string]interface{}{"volume": 30})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Errorf("expected Success=true, got %+v", res)
	}

	if len(fake.calls) == 0 {
		t.Fatal("expected at least one gdbus call")
	}
	call := fake.calls[0]

	hasSinkSetVolume := false
	hasRatio30 := false
	for _, a := range call.args {
		if a == "com.deepin.daemon.Audio.SinkSetVolume" {
			hasSinkSetVolume = true
		}
		if a == "0.300000" {
			hasRatio30 = true
		}
	}
	if !hasSinkSetVolume {
		t.Errorf("expected SinkSetVolume method, got %v", call.args)
	}
	if !hasRatio30 {
		t.Errorf("expected 0-1.0 ratio (0.300000 for 30), got %v", call.args)
	}

	// 验证 meta 里有 ratio
	if got, ok := res.Meta["ratio"].(string); !ok || got != "0.300000" {
		t.Errorf("expected meta.ratio=0.300000, got %v", res.Meta["ratio"])
	}
}

func TestDdeVolume_EdgeValues(t *testing.T) {
	// 0 和 100 也应该正确转 ratio
	fake := &fakeExecutor{stdout: "()"}
	SetExecutor(fake)
	defer ResetExecutor()

	d, _ := NewDdeVolume()

	t.Run("volume=0", func(t *testing.T) {
		fake.calls = nil
		res, _ := d.Run(context.Background(), map[string]interface{}{"volume": 0})
		if !res.Success {
			t.Errorf("expected Success=true for volume=0, got %+v", res)
		}
		// 检查传了 0.000000
		found := false
		for _, a := range fake.calls[0].args {
			if a == "0.000000" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected ratio=0.000000, got %v", fake.calls[0].args)
		}
	})

	t.Run("volume=100", func(t *testing.T) {
		fake.calls = nil
		res, _ := d.Run(context.Background(), map[string]interface{}{"volume": 100})
		if !res.Success {
			t.Errorf("expected Success=true for volume=100, got %+v", res)
		}
		found := false
		for _, a := range fake.calls[0].args {
			if a == "1.000000" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected ratio=1.000000, got %v", fake.calls[0].args)
		}
	})
}

// === DdeBrightness 测试 ===

func TestDdeBrightness_OutOfRange(t *testing.T) {
	d, _ := NewDdeBrightness()
	tests := []struct {
		name       string
		brightness interface{}
	}{
		{"negative_int", -1},
		{"too_large_int", 101},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _ := d.Run(context.Background(), map[string]interface{}{"brightness": tt.brightness})
			if res.Success {
				t.Errorf("expected Success=false for brightness %v", tt.brightness)
			}
		})
	}
}

func TestDdeBrightness_RealDBus_ConvertsToRatio(t *testing.T) {
	fake := &fakeExecutor{stdout: "()"}
	SetExecutor(fake)
	defer ResetExecutor()

	d, _ := NewDdeBrightness()
	res, _ := d.Run(context.Background(), map[string]interface{}{"brightness": 80})

	if !res.Success {
		t.Errorf("expected Success=true, got %+v", res)
	}

	hasSetBrightness := false
	hasRatio80 := false
	for _, a := range fake.calls[0].args {
		if a == "com.deepin.daemon.Display.Brightness.SetBrightness" {
			hasSetBrightness = true
		}
		if a == "0.800000" {
			hasRatio80 = true
		}
	}
	if !hasSetBrightness {
		t.Errorf("expected Brightness.SetBrightness method, got %v", fake.calls[0].args)
	}
	if !hasRatio80 {
		t.Errorf("expected 0.800000 ratio for 80, got %v", fake.calls[0].args)
	}
}

// === DdeNetwork 测试 ===

func TestDdeNetwork_InvalidState(t *testing.T) {
	d, _ := NewDdeNetwork()
	for _, state := range []string{"enable", "disable", "true", "1", "yes"} {
		t.Run("state="+state, func(t *testing.T) {
			res, _ := d.Run(context.Background(), map[string]interface{}{"state": state})
			if res.Success {
				t.Errorf("expected Success=false for state %q", state)
			}
		})
	}
}

func TestDdeNetwork_RealDBus_ChoosesCorrectMethod(t *testing.T) {
	fake := &fakeExecutor{stdout: "()"}
	SetExecutor(fake)
	defer ResetExecutor()

	d, _ := NewDdeNetwork()

	t.Run("state=on calls EnableWifi", func(t *testing.T) {
		fake.calls = nil
		res, _ := d.Run(context.Background(), map[string]interface{}{"state": "on"})
		if !res.Success {
			t.Errorf("expected Success=true, got %+v", res)
		}
		hasEnable := false
		for _, a := range fake.calls[0].args {
			if a == "com.deepin.daemon.Network.EnableWifi" {
				hasEnable = true
			}
		}
		if !hasEnable {
			t.Errorf("expected EnableWifi for state=on, got %v", fake.calls[0].args)
		}
	})

	t.Run("state=off calls DisableWifi", func(t *testing.T) {
		fake.calls = nil
		res, _ := d.Run(context.Background(), map[string]interface{}{"state": "off"})
		if !res.Success {
			t.Errorf("expected Success=true, got %+v", res)
		}
		hasDisable := false
		for _, a := range fake.calls[0].args {
			if a == "com.deepin.daemon.Network.DisableWifi" {
				hasDisable = true
			}
		}
		if !hasDisable {
			t.Errorf("expected DisableWifi for state=off, got %v", fake.calls[0].args)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		fake.calls = nil
		d.Run(context.Background(), map[string]interface{}{"state": "ON"})
		hasEnable := false
		for _, a := range fake.calls[0].args {
			if a == "com.deepin.daemon.Network.EnableWifi" {
				hasEnable = true
			}
		}
		if !hasEnable {
			t.Errorf("expected case-insensitive ON to call EnableWifi, got %v", fake.calls[0].args)
		}
	})
}

// === 公共 sanity 测试 ===

func TestAllTools_HaveNameAndDescription(t *testing.T) {
	tools := []Tool{
		&DdeTheme{},
		&DdeVolume{},
		&DdeBrightness{},
		&DdeNetwork{},
		&DdeWallpaper{},
	}
	for _, tool := range tools {
		if tool.Name() == "" {
			t.Errorf("%T: Name() should not be empty", tool)
		}
		if tool.Description() == "" {
			t.Errorf("%T: Description() should not be empty", tool)
		}
	}
}

// === 复制 v4 teams 的 gvariant 解析测试 ===

func TestParseGVariantString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"(<'deepin-dark',>,)", "deepin-dark"},
		{"(<value,>,)", "value"},
		{"(<'path/to/file.jpg',>,)", "path/to/file.jpg"},
		{"()", ""},
		{"", ""},
		{"not-gvariant-format", "not-gvariant-format"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseGVariantString(tt.input)
			if got != tt.want {
				t.Errorf("parseGVariantString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}