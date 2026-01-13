# CYP-Docker-Registry Makefile
# 构建和管理脚本

.PHONY: all build build-server build-cli build-web clean test lint docker help

# 变量
VERSION := $(shell cat VERSION)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# 默认目标
all: build

# 构建所有组件
build: build-server build-cli build-web
	@echo "Build complete!"

# 构建服务器
build-server:
	@echo "Building server..."
	go build $(LDFLAGS) -o bin/cyp-docker-registry ./cmd/server

# 构建 CLI 工具
build-cli:
	@echo "Building CLI..."
	go build $(LDFLAGS) -o bin/cyp-docker-registry-cli ./cmd/cli

# 构建前端
build-web:
	@echo "Building web frontend..."
	cd web && npm install && npm run build

# 清理构建产物
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf web/dist/
	go clean

# 运行测试
test:
	@echo "Running tests..."
	go test -v ./...
	cd web && npm test

# 代码检查
lint:
	@echo "Running linters..."
	go vet ./...
	golangci-lint run
	cd web && npm run lint

# 格式化代码
fmt:
	@echo "Formatting code..."
	go fmt ./...
	cd web && npm run format

# 依赖管理
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	cd web && npm install

# Docker 构建
docker:
	@echo "Building Docker image..."
	docker build -t cyp-docker-registry:$(VERSION) .
	docker tag cyp-docker-registry:$(VERSION) cyp-docker-registry:latest

# Docker Compose 启动
up:
	docker-compose up -d

# Docker Compose 停止
down:
	docker-compose down

# 开发模式运行
dev:
	@echo "Starting development server..."
	go run ./cmd/server &
	cd web && npm run dev

# 生成 API 文档
docs:
	@echo "Generating API documentation..."
	swag init -g cmd/server/main.go -o docs/swagger

# 数据库迁移
migrate:
	@echo "Running database migrations..."
	go run ./cmd/migrate

# 安装到系统
install: build
	@echo "Installing..."
	cp bin/cyp-docker-registry /usr/local/bin/
	cp bin/cyp-docker-registry-cli /usr/local/bin/
	mkdir -p /etc/cyp-docker-registry
	cp configs/config.yaml.example /etc/cyp-docker-registry/config.yaml

# 卸载
uninstall:
	@echo "Uninstalling..."
	rm -f /usr/local/bin/cyp-docker-registry
	rm -f /usr/local/bin/cyp-docker-registry-cli

# 跨平台构建
build-all:
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/cyp-docker-registry-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/cyp-docker-registry-linux-arm64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/cyp-docker-registry-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/cyp-docker-registry-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/cyp-docker-registry-windows-amd64.exe ./cmd/server

# 发布
release: clean build-all docker
	@echo "Creating release $(VERSION)..."
	mkdir -p release/$(VERSION)
	cp bin/* release/$(VERSION)/
	cd release/$(VERSION) && sha256sum * > checksums.txt
	@echo "Release $(VERSION) created!"

# 版本管理 (使用 UVM)
version-patch:
	@echo "Bumping patch version..."
	@node tools/uvm/bin/uvm.js patch
	@node scripts/sync-version.js

version-minor:
	@echo "Bumping minor version..."
	@node tools/uvm/bin/uvm.js minor
	@node scripts/sync-version.js

version-major:
	@echo "Bumping major version..."
	@node tools/uvm/bin/uvm.js major
	@node scripts/sync-version.js

version-info:
	@node tools/uvm/bin/uvm.js info

version-history:
	@node tools/uvm/bin/uvm.js history

version-sync:
	@node scripts/sync-version.js

# 帮助
help:
	@echo "CYP-Docker-Registry Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          Build all components (default)"
	@echo "  build        Build server, CLI, and web"
	@echo "  build-server Build server binary"
	@echo "  build-cli    Build CLI binary"
	@echo "  build-web    Build web frontend"
	@echo "  clean        Clean build artifacts"
	@echo "  test         Run tests"
	@echo "  lint         Run linters"
	@echo "  fmt          Format code"
	@echo "  deps         Install dependencies"
	@echo "  docker       Build Docker image"
	@echo "  up           Start with Docker Compose"
	@echo "  down         Stop Docker Compose"
	@echo "  dev          Start development server"
	@echo "  install      Install to system"
	@echo "  uninstall    Uninstall from system"
	@echo "  build-all    Build for all platforms"
	@echo "  release      Create release"
	@echo "  help         Show this help"
	@echo ""
	@echo "Version Management (UVM):"
	@echo "  version-patch  Bump patch version (1.0.0 -> 1.0.1)"
	@echo "  version-minor  Bump minor version (1.0.0 -> 1.1.0)"
	@echo "  version-major  Bump major version (1.0.0 -> 2.0.0)"
	@echo "  version-info   Show version info"
	@echo "  version-history Generate version history"
