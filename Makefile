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
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -o $(OUTPUT_DIR)/$(PROJECT_NAME).exe

build-mac:
	@echo "Building for Mac..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -o $(OUTPUT_DIR)/$(PROJECT_NAME)-mac

# 构建 Linux 可执行文件
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(OUTPUT_DIR)/$(PROJECT_NAME)

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf $(OUTPUT_DIR)

# 如果是mac系统 复制产物到mac bin目录
# 如果是linux系统 复制产物到linux bin目录
.PHONY: install
install:
	@echo "installing..."
ifeq ($(shell uname), Darwin)
	cp $(OUTPUT_DIR)/$(PROJECT_NAME)-mac /usr/local/bin/$(PROJECT_NAME)
else ifeq ($(shell uname), Linux)
	cp $(OUTPUT_DIR)/$(PROJECT_NAME) /usr/local/bin/$(PROJECT_NAME)
endif

