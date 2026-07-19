// Package tools - gvariant.go
//
// gdbus 返回值的 GVariant 字符串解析。
//
// gdbus 单字符串返回值格式（实测 deepin 25）：
//   - (<'value',>,)        ← deepin 25 实际格式（5 字符后缀）
//   - (<'value',>)         ← v4 teams 老格式（4 字符后缀，部分版本）
//   - (<"value",>,)        ← 带引号
//   - (value,)             ← 无引号（godbus 风格）
//   - ()                   ← 空元组
//
// 调用方拿到 gdbus stdout 后用 parseGVariantString 提取 string 值。
package tools

import (
	"strconv"
	"strings"
)

// parseGVariantString 从 gdbus 返回的 GVariant 字符串中提取 string 值
//
// 适配多种格式（向后兼容 v4 teams 老格式 + deepin 25 新格式）：
//   - (<'deepin-dark',>,)  → deepin-dark  （5 后缀）
//   - (<'deepin-dark',>)   → deepin-dark  （4 后缀，v4 teams 老格式）
//   - (<"value",>,)        → value
//   - (value,)             → value
//   - ()                   → ""
//
// 不匹配返回原样（让上层处理）
func parseGVariantString(gvariant string) string {
	s := strings.TrimSpace(gvariant)
	if s == "" {
		return ""
	}

	// 嵌套单引号：(<'value',>,)  5 后缀（deepin 25）
	if strings.HasPrefix(s, "(<'") && strings.HasSuffix(s, "',>,)") {
		return s[3 : len(s)-5]
	}
	// 嵌套单引号：(<'value',>)  4 后缀（v4 teams 老格式）
	if strings.HasPrefix(s, "(<'") && strings.HasSuffix(s, "',>)") {
		return s[3 : len(s)-4]
	}
	// 嵌套双引号：(5 后缀)
	if strings.HasPrefix(s, "(<\"") && strings.HasSuffix(s, "\",>,)") {
		return s[3 : len(s)-5]
	}
	// 嵌套双引号：(4 后缀)
	if strings.HasPrefix(s, "(<\"") && strings.HasSuffix(s, "\",>)") {
		return s[3 : len(s)-4]
	}
	// 嵌套无引号（(<value,>,)）— 不常见，但容忍
	if strings.HasPrefix(s, "(<") && strings.HasSuffix(s, ">,)") {
		end := len(s) - 4 // 去掉 ",>,)" 后缀
		if end <= 2 {
			return s
		}
		return s[2:end]
	}
	// 嵌套无引号（(<value,>)）— 4 后缀变体
	if strings.HasPrefix(s, "(<") && strings.HasSuffix(s, ">)") {
		end := len(s) - 3 // 去掉 ",>)" 后缀
		if end <= 2 {
			return s
		}
		return s[2:end]
	}
	// 单层单引号：('value',)
	if strings.HasPrefix(s, "('") && strings.HasSuffix(s, "',)") {
		return s[2 : len(s)-3]
	}
	// 单层双引号：("value",)
	if strings.HasPrefix(s, "(\"") && strings.HasSuffix(s, "\",)") {
		return s[2 : len(s)-3]
	}
	// 简单形式：(value,)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ",)") && len(s) >= 3 {
		return s[1 : len(s)-2]
	}
	// 空元组：()
	if s == "()" {
		return ""
	}
	// 不是单字符串 → 原样返回
	return s
}

// parseDoubleFromGVariant 从 gdbus 返回的 GVariant 字符串中提取 double 值
//
// gdbus double 返回格式：
//   - (0.500000,)
//   - (1,)
//   - ()
//
// 失败返回 -1
func parseDoubleFromGVariant(gvariant string) float64 {
	s := strings.TrimSpace(gvariant)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ",)")
	s = strings.TrimSuffix(s, ")")

	if s == "" {
		return -1
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	return f
}