// Package tools - dde_settings_test.go
//
// v0.3 Phase 2 补:4 个新 tool 的单元测试
//
// 只测参数校验逻辑（不真打 D-Bus）：
//   - 缺参数
//   - 类型错误
//   - 值越界
//   - 非法字符串
//
// D-Bus 真调用要在 deepin 25 真机验证（见 DEEPIN_25_TESTING.md）。
package tools

import (
	"context"
	"testing"
)

func TestDdeTheme_InvalidTheme(t *testing.T) {
	d, err := NewDdeTheme()
	if err != nil {
		t.Skipf("D-Bus not available (probably not on deepin 25): %v", err)
	}
	defer d.Close()

	// 区分 missing (返回 error) 和 invalid value (返回 Result)
	tests := []struct {
		name      string
		theme     string
		wantError bool
	}{
		{"empty", "", true},  // 缺参 → 返回 error
		{"unknown", "purple", false}, // 非法值 → Result+nil
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
				t.Fatalf("Run should not return error for invalid input, got: %v", err)
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

func TestDdeTheme_MissingTheme(t *testing.T) {
	d, err := NewDdeTheme()
	if err != nil {
		t.Skipf("D-Bus not available: %v", err)
	}
	defer d.Close()

	res, err := d.Run(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing theme")
	}
	if res.Success {
		t.Error("expected Success=false")
	}
	if res.ErrorType != "invalid_input" {
		t.Errorf("expected ErrorType=invalid_input, got %q", res.ErrorType)
	}
}

func TestDdeVolume_OutOfRange(t *testing.T) {
	d, err := NewDdeVolume()
	if err != nil {
		t.Skipf("D-Bus not available: %v", err)
	}
	defer d.Close()

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
			if res.ErrorType != "invalid_input" {
				t.Errorf("expected ErrorType=invalid_input, got %q", res.ErrorType)
			}
		})
	}
}

func TestDdeVolume_TypeCoercion(t *testing.T) {
	d, err := NewDdeVolume()
	if err != nil {
		t.Skipf("D-Bus not available: %v", err)
	}
	defer d.Close()

	// 只测字符串解析失败的情况（其他类型 跳过 D-Bus 真调用）
	t.Run("invalid_string", func(t *testing.T) {
		res, err := d.Run(context.Background(), map[string]interface{}{"volume": "abc"})
		if err != nil {
			t.Fatalf("Run should not return error: %v", err)
		}
		if res.Success {
			t.Error("expected Success=false for non-numeric string")
		}
		if res.ErrorType != "invalid_input" {
			t.Errorf("expected ErrorType=invalid_input, got %q", res.ErrorType)
		}
	})
}

func TestDdeBrightness_OutOfRange(t *testing.T) {
	d, err := NewDdeBrightness()
	if err != nil {
		t.Skipf("D-Bus not available: %v", err)
	}
	defer d.Close()

	tests := []struct {
		name       string
		brightness interface{}
	}{
		{"negative_int", -1},
		{"too_large_int", 101},
		{"negative_string", "-50"},
		{"too_large_string", "999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := d.Run(context.Background(), map[string]interface{}{"brightness": tt.brightness})
			if err != nil {
				t.Fatalf("Run should not return error: %v", err)
			}
			if res.Success {
				t.Errorf("expected Success=false for brightness %v", tt.brightness)
			}
		})
	}
}

func TestDdeNetwork_InvalidState(t *testing.T) {
	d, err := NewDdeNetwork()
	if err != nil {
		t.Skipf("D-Bus not available: %v", err)
	}
	defer d.Close()

	tests := []string{"", "enable", "disable", "true", "1", "yes"}
	for _, state := range tests {
		t.Run("state="+state, func(t *testing.T) {
			res, err := d.Run(context.Background(), map[string]interface{}{"state": state})
			if err != nil && state != "" {
				// 空字符串会返回 error（state required）
				t.Fatalf("Run should not return error for non-empty state: %v", err)
			}
			if state == "" {
				// 空字符串 -> error
				if err == nil {
					t.Error("expected error for empty state")
				}
				return
			}
			if res.Success {
				t.Errorf("expected Success=false for state %q", state)
			}
			if res.ErrorType != "invalid_input" {
				t.Errorf("expected ErrorType=invalid_input, got %q", res.ErrorType)
			}
		})
	}
}

func TestDdeNetwork_StateCaseInsensitive(t *testing.T) {
	d, err := NewDdeNetwork()
	if err != nil {
		t.Skipf("D-Bus not available: %v", err)
	}
	defer d.Close()

	// ON / Off / OFF 大小写都应该接受（走到 D-Bus 阶段），所以测试只校验格式解析。
	// 在非 deepin 25 环境下 D-Bus 会失败，但不会因为大小写报 invalid_input。
	for _, state := range []string{"ON", "Off", "OFF", "on", "off"} {
		t.Run("state="+state, func(t *testing.T) {
			res, _ := d.Run(context.Background(), map[string]interface{}{"state": state})
			// 大小写归一化后是 on/off,所以不该报 invalid_input
			if res.ErrorType == "invalid_input" {
				t.Errorf("state %q should be normalized (case insensitive), got invalid_input", state)
			}
		})
	}
}

func TestAllTools_HaveNameAndDescription(t *testing.T) {
	// 编译期 sanity check：所有 tool 都该实现 Tool 接口并有名字 + 描述
	tools := []Tool{
		&DdeTheme{},
		&DdeVolume{},
		&DdeBrightness{},
		&DdeNetwork{},
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