package tools

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// LlCliInstall 通过玲珑包安装应用。
//
// 适配 deepin 25 玲珑包 CLI（ll-cli install）。
// 真实环境反馈来自 ll-cli 的 exit code + stderr（机器可读）。
type LlCliInstall struct{}

// NewLlCliInstall 创建玲珑包安装工具
func NewLlCliInstall() *LlCliInstall {
	return &LlCliInstall{}
}

func (l *LlCliInstall) Name() string {
	return "ll_cli_install"
}

func (l *LlCliInstall) Description() string {
	return "通过玲珑包（linglong）安装 deepin 应用。输入：package_id（如 org.deepin.calculator）"
}

func (l *LlCliInstall) Run(ctx context.Context, input map[string]interface{}) (*Result, error) {
	start := time.Now()

	// 1. 参数校验
	pkgID, ok := input["package_id"].(string)
	if !ok || pkgID == "" {
		return &Result{
			Success:   false,
			Error:     "missing or invalid package_id parameter",
			ErrorType: "invalid_input",
			Duration:  time.Since(start),
		}, fmt.Errorf("package_id required")
	}

	// 2. 设置超时（玲珑包安装可能耗时 30s+）
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	// 3. 执行 ll-cli install
	cmd := exec.CommandContext(ctx, "ll-cli", "install", pkgID)
	out, err := cmd.CombinedOutput()

	result := &Result{
		ExitCode: cmd.ProcessState.ExitCode(),
		Content:  string(out),
		Duration: time.Since(start),
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.ErrorType = ClassifyError(result.ExitCode, string(out))
		return result, nil // 返回 result + nil error，让上层根据 Success 判断
	}

	result.Success = true
	result.Verified = true // exit code 0 + 无 error = 已验证成功
	return result, nil
}

func (l *LlCliInstall) HealthCheck(ctx context.Context) error {
	// 检查 ll-cli 是否存在 + 版本
	cmd := exec.CommandContext(ctx, "ll-cli", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ll-cli not available: %w", err)
	}
	return nil
}

// LlCliRun 运行玲珑包应用
type LlCliRun struct{}

func NewLlCliRun() *LlCliRun { return &LlCliRun{} }

func (l *LlCliRun) Name() string { return "ll_cli_run" }

func (l *LlCliRun) Description() string {
	return "运行玲珑包应用（在前台）。输入：package_id（如 org.deepin.calculator）"
}

func (l *LlCliRun) Run(ctx context.Context, input map[string]interface{}) (*Result, error) {
	start := time.Now()

	pkgID, ok := input["package_id"].(string)
	if !ok || pkgID == "" {
		return &Result{
			Success:   false,
			Error:     "missing package_id",
			ErrorType: "invalid_input",
			Duration:  time.Since(start),
		}, fmt.Errorf("package_id required")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ll-cli", "run", pkgID)
	out, err := cmd.CombinedOutput()

	result := &Result{
		ExitCode: cmd.ProcessState.ExitCode(),
		Content:  string(out),
		Duration: time.Since(start),
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.ErrorType = ClassifyError(result.ExitCode, string(out))
		return result, nil
	}

	result.Success = true
	return result, nil
}

func (l *LlCliRun) HealthCheck(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "ll-cli", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ll-cli not available: %w", err)
	}
	return nil
}