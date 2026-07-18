package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemMove_Basic(t *testing.T) {
	// 创建临时测试目录
	src, err := os.MkdirTemp("", "test-src-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(src)

	dst, err := os.MkdirTemp("", "test-dst-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dst)

	// 在 src 创建测试文件
	testFiles := map[string]string{
		"photo1.jpg": "fake jpg content",
		"photo2.png": "fake png content",
		"doc.pdf":    "fake pdf content",
		"note.txt":   "fake txt content",
	}
	for name, content := range testFiles {
		path := filepath.Join(src, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 执行 move
	tool := NewFilesystemMove()
	input := map[string]interface{}{
		"src":        src,
		"dst":        dst,
		"extensions": []string{"jpg", "png"},
	}

	result, err := tool.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got false: %s", result.Error)
	}
	if !result.Verified {
		t.Errorf("expected verified=true")
	}

	// 验证结果
	movedCount, _ := result.Meta["moved_count"].(int)
	if movedCount != 2 {
		t.Errorf("expected 2 files moved, got %d", movedCount)
	}

	// src 应该剩 2 个文件（pdf + txt）
	remaining, _ := os.ReadDir(src)
	if len(remaining) != 2 {
		t.Errorf("expected 2 files remaining in src, got %d", len(remaining))
	}

	// dst 应该有 2 个文件
	arrived, _ := os.ReadDir(dst)
	if len(arrived) != 2 {
		t.Errorf("expected 2 files in dst, got %d", len(arrived))
	}
}

func TestFilesystemMove_InvalidSrc(t *testing.T) {
	tool := NewFilesystemMove()
	input := map[string]interface{}{
		"src":        "/nonexistent/path",
		"dst":        "/tmp",
		"extensions": []string{"jpg"},
	}

	result, _ := tool.Run(context.Background(), input)

	if result.Success {
		t.Error("expected failure for nonexistent src")
	}
	if result.ErrorType != "not_found" {
		t.Errorf("expected error_type=not_found, got %q", result.ErrorType)
	}
}

func TestFilesystemMove_MissingParam(t *testing.T) {
	tool := NewFilesystemMove()
	input := map[string]interface{}{
		"src": "/tmp",
		// 缺少 dst
	}

	result, _ := tool.Run(context.Background(), input)

	if result.Success {
		t.Error("expected failure for missing dst")
	}
	if result.ErrorType != "invalid_input" {
		t.Errorf("expected error_type=invalid_input, got %q", result.ErrorType)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		code     int
		stderr   string
		expected string
	}{
		{0, "", ""},
		{1, "permission denied", "permission"},
		{1, "Permission denied", "permission"},
		{2, "file not found", "not_found"},
		{3, "connection timeout", "network"},
		{1, "device busy", "busy"},
		{99, "unknown error xyz", "unknown"},
	}
	for _, tt := range tests {
		got := ClassifyError(tt.code, tt.stderr)
		if got != tt.expected {
			t.Errorf("ClassifyError(%d, %q) = %q, want %q",
				tt.code, tt.stderr, got, tt.expected)
		}
	}
}