// Package tools - dbus.go
//
// v0.3 Phase 2 升级:D-Bus 通用调用层(走 gdbus 命令 + CommandExecutor 可注入)
//
// 跟 deepin-agent-teams v4 的 internal/tools/dbus.go 完全一致:
// - 默认真调 gdbus
// - 测试通过 Executor interface 注入 mock
// - 环境变量 DEEPIN_DBUS=mock 强制走 mock(demo / CI 用)
//
// 设计动机:跟 deepin 官方文档对齐 + 测试友好 + 零 CGO 部署
//
// 与 godbus 相比:
//   - gdbus 走进程 fork,Godbus 走 CGO/libdbus
//   - 性能 godbus 略胜,但 v0.3 场景只调几次
//   - 测试覆盖 gdbus 大胜(Executor 注入让 Ubuntu 上能验证接口签名)
package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandExecutor 抽象 exec.Command 行为，方便测试 mock
//
// 默认实现是 realExecutor（真调 exec.Command）。
// 测试时用 fakeExecutor 预设 stdout / stderr / err。
type CommandExecutor interface {
	Run(ctx context.Context, name string, args ...string) (stdout string, stderr string, err error)
}

// realExecutor 真的调用 os/exec
type realExecutor struct{}

func (realExecutor) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	return strings.TrimRight(stdoutBuf.String(), "\n"),
		strings.TrimRight(stderrBuf.String(), "\n"),
		err
}

var defaultExecutor CommandExecutor = realExecutor{}

// SetExecutor 设置全局命令执行器（测试用）
func SetExecutor(e CommandExecutor) {
	defaultExecutor = e
}

// ResetExecutor 恢复默认执行器（测试清理用）
func ResetExecutor() {
	defaultExecutor = realExecutor{}
}

// dbusMode 当前 D-Bus 运行模式
type dbusModeT int

const (
	modeAuto dbusModeT = iota
	modeMock
	modeReal
)

func (m dbusModeT) effective() dbusModeT {
	if m != modeAuto {
		return m
	}
	if os.Getenv("DEEPIN_DBUS") == "mock" {
		return modeMock
	}
	return modeReal
}

func (m dbusModeT) String() string {
	switch m.effective() {
	case modeMock:
		return "mock"
	case modeReal:
		return "real"
	default:
		return "auto(real)"
	}
}

// dbusCall 调用 D-Bus 方法（通过 gdbus 命令）
//
// 返回：gdbus 的 stdout（去掉尾换行）和错误。
//
// 实现：
//   - mock 模式（DEEPIN_DBUS=mock）：返回假成功 + 占位 stdout
//   - real 模式：调 `gdbus call --session -d <dest> -o <path> -m <method> <args>`
func dbusCall(ctx context.Context, dest, objectPath, method string, args ...string) (string, error) {
	mode := dbusModeT(0).effective()
	if mode == modeMock {
		return mockDBusResponse(dest, method, args), nil
	}
	allArgs := []string{"call", "--session", "-d", dest, "-o", objectPath, "-m", method}
	allArgs = append(allArgs, args...)
	stdout, stderr, err := defaultExecutor.Run(ctx, "gdbus", allArgs...)
	if err != nil {
		return stdout, fmt.Errorf("dbus call %s.%s failed: %w (stderr: %s)", dest, method, err, stderr)
	}
	return stdout, nil
}

// dbusCallSystem 系统总线版本
func dbusCallSystem(ctx context.Context, dest, objectPath, method string, args ...string) (string, error) {
	mode := dbusModeT(0).effective()
	if mode == modeMock {
		return mockDBusResponse(dest, method, args), nil
	}
	allArgs := []string{"call", "--system", "-d", dest, "-o", objectPath, "-m", method}
	allArgs = append(allArgs, args...)
	stdout, stderr, err := defaultExecutor.Run(ctx, "gdbus", allArgs...)
	if err != nil {
		return stdout, fmt.Errorf("system dbus call %s.%s failed: %w (stderr: %s)", dest, method, err, stderr)
	}
	return stdout, nil
}

// mockDBusResponse 假 D-Bus 响应
//
// 不同方法返回合理的占位：
//   - SetXxx → ()
//   - GetXxx → (<'mock-value',>)
//   - Enable/DisableXxx → ()
func mockDBusResponse(dest, method string, args []string) string {
	parts := strings.Split(method, ".")
	name := parts[len(parts)-1]
	switch {
	case strings.HasPrefix(name, "Set"):
		return "()"
	case strings.HasPrefix(name, "Get"):
		return "(<'mock-value',>)"
	case strings.HasPrefix(name, "Enable"), strings.HasPrefix(name, "Disable"):
		return "()"
	default:
		return "()"
	}
}

// errDBusUnavailable 在没有 deepin 服务时返回的错误
var errDBusUnavailable = errors.New("D-Bus service unavailable (likely not running on deepin 25)")

// IsMockMode 报告当前是否在 mock 模式
func IsMockMode() bool {
	return dbusModeT(0).effective() == modeMock
}