# deepin-agent-playground Makefile
# 支持：原生构建 + 玲珑包打包

# Go 二进制构建
.PHONY: build
build:
	CGO_ENABLED=0 go build -o bin/deepin-agent-playground ./cmd/playground
	@echo "✓ built: bin/deepin-agent-playground"

# 构建所有 example
.PHONY: build-examples
build-examples:
	CGO_ENABLED=0 go build -o bin/demo-install-organize-wallpaper ./examples/demo_install_organize_wallpaper.go
	@echo "✓ built: bin/demo-install-organize-wallpaper"

# 运行测试
.PHONY: test
test:
	go test -v -race ./...

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
	./bin/deepin-agent-playground --dry-run --input "帮我装计算器，整理 ~/Downloads，把壁纸换成日落"