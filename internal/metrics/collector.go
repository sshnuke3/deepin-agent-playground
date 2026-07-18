// Package metrics 收集 agent 运行过程中的 tool 调用结果。
//
// 这些 metrics 就是 Sutton option 学习的训练数据：
//   - success rate（option 稳定性）
//   - latency（option 耗时分布）
//   - error type 分布（option 失败模式）
package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sshnuke3/daplayground/internal/tools"
)

// Record 是单次 tool 调用的 metrics 记录
type Record struct {
	Tool      string
	Success   bool
	Verified  bool
	ExitCode  int
	ErrorType string
	Duration  time.Duration
	Timestamp time.Time
}

// Collector 线程安全地收集所有 tool 调用
type Collector struct {
	mu      sync.Mutex
	records []Record
}

// NewCollector 创建 collector
func NewCollector() *Collector {
	return &Collector{records: make([]Record, 0)}
}

// Record 记录一次调用结果
func (c *Collector) Record(toolName string, r *tools.Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, Record{
		Tool:      toolName,
		Success:   r.Success,
		Verified:  r.Verified,
		ExitCode:  r.ExitCode,
		ErrorType: r.ErrorType,
		Duration:  r.Duration,
		Timestamp: time.Now(),
	})
}

// Report 输出最终报告
func (c *Collector) Report() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.records) == 0 {
		return "no records collected"
	}

	// 按 tool 聚合
	type toolStat struct {
		total    int
		success  int
		verified int
		durations []time.Duration
		errors   map[string]int
	}
	stats := make(map[string]*toolStat)

	for _, r := range c.records {
		s, ok := stats[r.Tool]
		if !ok {
			s = &toolStat{errors: make(map[string]int)}
			stats[r.Tool] = s
		}
		s.total++
		if r.Success {
			s.success++
		}
		if r.Verified {
			s.verified++
		}
		s.durations = append(s.durations, r.Duration)
		if r.ErrorType != "" {
			s.errors[r.ErrorType]++
		}
	}

	out := "\n" + strings.Repeat("=", 60) + "\n"
	out += "📊 METRICS REPORT\n"
	out += strings.Repeat("=", 60) + "\n"
	out += fmt.Sprintf("Total tool calls: %d\n", len(c.records))
	out += fmt.Sprintf("Total success:    %d (%.1f%%)\n",
		countSuccess(c.records),
		float64(countSuccess(c.records))/float64(len(c.records))*100)
	out += fmt.Sprintf("Total verified:  %d (%.1f%%)\n",
		countVerified(c.records),
		float64(countVerified(c.records))/float64(len(c.records))*100)
	out += "\nPer-tool breakdown:\n"

	// 工具名排序（输出稳定）
	toolNames := make([]string, 0, len(stats))
	for name := range stats {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	for _, name := range toolNames {
		s := stats[name]
		avgDur := avg(s.durations)
		out += fmt.Sprintf("  • %s:\n", name)
		out += fmt.Sprintf("      calls:    %d\n", s.total)
		out += fmt.Sprintf("      success:  %d (%.1f%%)\n", s.success,
			float64(s.success)/float64(s.total)*100)
		out += fmt.Sprintf("      verified: %d (%.1f%%)\n", s.verified,
			float64(s.verified)/float64(s.total)*100)
		out += fmt.Sprintf("      avg dur:  %s\n", avgDur)
		if len(s.errors) > 0 {
			out += fmt.Sprintf("      errors:   %v\n", s.errors)
		}
	}

	out += strings.Repeat("=", 60) + "\n"
	return out
}

func countSuccess(rs []Record) int {
	n := 0
	for _, r := range rs {
		if r.Success {
			n++
		}
	}
	return n
}

func countVerified(rs []Record) int {
	n := 0
	for _, r := range rs {
		if r.Verified {
			n++
		}
	}
	return n
}

func avg(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range ds {
		sum += d
	}
	return sum / time.Duration(len(ds))
}