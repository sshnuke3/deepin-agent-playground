package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FilesystemMove 按扩展名整理文件。
//
// Sutton option 学习的关键设计：
//   - 移动前记录文件集合 (before)
//   - 执行移动
//   - 移动后再次记录 (after)
//   - 验证: after = before - moved（机器可读的成功判定）
type FilesystemMove struct{}

// NewFilesystemMove 创建文件整理工具
func NewFilesystemMove() *FilesystemMove { return &FilesystemMove{} }

func (f *FilesystemMove) Name() string { return "filesystem_move_by_ext" }

func (f *FilesystemMove) Description() string {
	return "按扩展名整理文件到目标目录。输入：src（源目录绝对路径）、dst（目标目录绝对路径）、extensions（扩展名列表如 [\"jpg\",\"png\"]）"
}

func (f *FilesystemMove) Run(ctx context.Context, input map[string]interface{}) (*Result, error) {
	start := time.Now()

	// 1. 参数校验
	src, ok := input["src"].(string)
	if !ok || src == "" {
		return &Result{
			Success: false, Error: "missing src", ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, fmt.Errorf("src required")
	}
	dst, ok := input["dst"].(string)
	if !ok || dst == "" {
		return &Result{
			Success: false, Error: "missing dst", ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, fmt.Errorf("dst required")
	}

	// 2. extensions 解析（支持 []string 或 []interface{}）
	var extensions []string
	switch v := input["extensions"].(type) {
	case []string:
		extensions = v
	case []interface{}:
		for _, e := range v {
			if s, ok := e.(string); ok {
				extensions = append(extensions, s)
			}
		}
	case string:
		// 单个扩展名（兼容）
		extensions = []string{v}
	default:
		return &Result{
			Success: false, Error: "extensions must be []string", ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, fmt.Errorf("extensions type error")
	}

	// 3. 检查源目录
	srcInfo, err := os.Stat(src)
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "not_found",
			Duration: time.Since(start),
		}, nil
	}
	if !srcInfo.IsDir() {
		return &Result{
			Success: false, Error: "src is not a directory", ErrorType: "invalid_input",
			Duration: time.Since(start),
		}, nil
	}

	// 4. 确保目标目录存在
	if err := os.MkdirAll(dst, 0755); err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "permission",
			Duration: time.Since(start),
		}, nil
	}

	// 5. 移动前快照（before）
	beforeFiles, err := listFiles(src)
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "unknown",
			Duration: time.Since(start),
		}, nil
	}

	// 6. 按扩展名筛选 + 移动
	extSet := make(map[string]bool)
	for _, e := range extensions {
		extSet[strings.ToLower(e)] = true
	}

	moved := make(map[string]string)
	for _, fname := range beforeFiles {
		// 取扩展名
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fname), "."))
		if !extSet[ext] {
			continue
		}

		from := filepath.Join(src, fname)
		to := filepath.Join(dst, fname)

		// 移动文件
		if err := os.Rename(from, to); err != nil {
			// 部分失败不算全失败，但记录
			return &Result{
				Success:   false,
				Error:     fmt.Sprintf("failed to move %s: %v", fname, err),
				ErrorType: ClassifyError(1, err.Error()),
				Content:   fmt.Sprintf("moved %d before failure", len(moved)),
				Meta: map[string]interface{}{
					"moved_count":   len(moved),
					"failure_file":  fname,
					"files_moved":   moved,
				},
				Duration: time.Since(start),
			}, nil
		}
		moved[fname] = to
	}

	// 7. 移动后快照（after）+ 验证
	afterFiles, err := listFiles(src)
	if err != nil {
		return &Result{
			Success: false, Error: err.Error(), ErrorType: "unknown",
			Duration: time.Since(start),
		}, nil
	}

	beforeSet := stringSet(beforeFiles)
	afterSet := stringSet(afterFiles)
	expectedAfter := subtract(beforeSet, stringSet(keys(moved)))

	verified := stringSetEqual(afterSet, expectedAfter)

	return &Result{
		Success:  true,
		ExitCode: 0,
		Content:  fmt.Sprintf("moved %d files to %s", len(moved), dst),
		Verified: verified,
		Meta: map[string]interface{}{
			"src":            src,
			"dst":            dst,
			"extensions":     extensions,
			"before_count":   len(beforeFiles),
			"after_count":    len(afterFiles),
			"moved_count":    len(moved),
			"files_moved":    moved,
		},
		Duration: time.Since(start),
	}, nil
}

func (f *FilesystemMove) HealthCheck(ctx context.Context) error {
	// 检查 HOME 目录可读
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(home); err != nil {
		return err
	}
	return nil
}

// === helpers ===

func listFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.Type().IsRegular() {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // 稳定排序
	return files, nil
}

func stringSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func subtract(a, b map[string]bool) map[string]bool {
	out := make(map[string]bool, len(a))
	for k := range a {
		if !b[k] {
			out[k] = true
		}
	}
	return out
}

func stringSetEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}