# daplayground Makefile
# 支持：原生构建 + 玲珑包打包 + Eino / Legacy 双模式

# 默认构建（不含 Eino，避免 sonic loader 链接问题）
.PHONY: build
build:
	CGO_ENABLED=0 go build -o bin/playground ./cmd/playground
	CGO_ENABLED=0 go build -o bin/demo ./examples
	@echo "✓ built: bin/playground (4.6M) + bin/demo (4.5M)"

# Eino 模式构建（需要 deepin 25 + Go 1.22 + sonic v1.13）
.PHONY: build-eino
build-eino:
	CGO_ENABLED=0 go build -tags eino -o bin/playground-eino ./cmd/playground
	@echo "✓ built: bin/playground-eino (Eino ADK 模式)"

# 构建所有 example
.PHONY: build-examples
build-examples: build
	@echo "✓ examples built"

# 运行测试
.PHONY: test
test:
	go test -v ./internal/tools/...

# 静态检查
.PHONY: vet
vet:
	go vet ./...

# 格式化
.PHONY: fmt
fmt:
	go fmt ./...

# 依赖管理
.PHONY: tidy
tidy:
	go mod tidy

# 玲珑包打包（需要先 build）
.PHONY: linglong
linglong: build
	@echo "→ 打包玲珑包 ..."
	ll-builder build
	@echo "✓ 玲珑包构建完成"

# 清理
.PHONY: clean
clean:
	rm -rf bin/
	rm -f *.layer

# 完整流程：build + test + linglong
.PHONY: all
all: tidy fmt vet test build linglong
	@echo "✓ all done"

# Dry-run 模式（不需要 deepin 25 环境）
.PHONY: demo-dry
demo-dry: build
	./bin/playground --dry-run --input "帮我装计算器，整理 ~/Downloads，把壁纸换成日落"

# Eino 模式 demo（需要 Ollama 跑着）
.PHONY: demo-eino
demo-eino: build-eino
	./bin/playground-eino --input "帮我装计算器，整理 ~/Downloads，把壁纸换成日落"

# Legacy 模式 demo（手写 agent 循环）
.PHONY: demo-legacy
demo-legacy: build
	./bin/playground --legacy --input "帮我装计算器，整理 ~/Downloads，把壁纸换成日落"