.PHONY: build run test clean lint fmt help

# 变量
APP_NAME := copilot
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := bin
CMD_DIR := .

# Go 相关
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# 环境变量
export GOSUMDB := sum.golang.org

# 构建标志
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

# 默认目标
.DEFAULT_GOAL := help

## build: 构建应用
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)"

## run: 运行应用
run:
	@echo "Running $(APP_NAME)..."
	$(GOCMD) run ./$(CMD_DIR)

## test: 运行测试
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "Test coverage:"
	$(GOCMD) tool cover -func=coverage.out | tail -1

## test-coverage: 生成测试覆盖率报告
test-coverage:
	@echo "Generating coverage report..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: 运行代码检查
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

## fmt: 格式化代码
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## tidy: 整理依赖
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

## clean: 清理构建产物
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## install: 安装到 $GOPATH/bin
install:
	@echo "Installing $(APP_NAME)..."
	$(GOCMD) install ./$(CMD_DIR)

## deps: 安装依赖
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download

## help: 显示帮助信息
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
