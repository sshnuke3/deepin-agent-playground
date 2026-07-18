// Package config 提供 agent 的运行时配置。
package config

import (
	"os"
	"strconv"
)

// Config 是 agent 的配置
type Config struct {
	// Ollama 服务地址
	OllamaBaseURL string
	// Ollama 模型名（如 deepseek-r1:1.5b）
	OllamaModel string
	// 日志级别（debug / info / warn / error）
	LogLevel string
	// 是否使用真实 D-Bus（false = mock mode 用于离线测试）
	UseRealDBus bool
	// 测试模式：跳过实际执行，只返回 mock 结果
	DryRun bool
}

// Default 返回默认配置
func Default() *Config {
	return &Config{
		OllamaBaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:   getEnv("OLLAMA_MODEL", "deepseek-r1:1.5b"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		UseRealDBus:   getEnvBool("USE_REAL_DBUS", true),
		DryRun:        getEnvBool("DRY_RUN", false),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}