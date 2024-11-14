# Makefile

# 项目名称
PROJECT_NAME := kube-ui

# Go 编译器
GO := go

# 输出目录
OUTPUT_DIR := bin

# 默认目标
.PHONY: all
all: build-windows build-linux build-mac

# 构建 Windows 可执行文件
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GO) build -o $(OUTPUT_DIR)/$(PROJECT_NAME).exe

build-mac:
	@echo "Building for Mac..."
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(OUTPUT_DIR)/$(PROJECT_NAME)-mac

# 构建 Linux 可执行文件
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GO) build -o $(OUTPUT_DIR)/$(PROJECT_NAME)

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf $(OUTPUT_DIR)