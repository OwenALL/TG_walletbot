.PHONY: help build run test lint clean docker-up docker-down migrate git-init review commit commit-msg push git-status git-log

# 默认目标
help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==================== 构建 ====================

build-bot: ## 构建 Bot 服务
	go build -o bin/bot ./cmd/bot

build-api: ## 构建 API 服务
	go build -o bin/api ./cmd/api

build: build-bot build-api ## 构建所有服务

# ==================== 运行 ====================

run-bot: ## 运行 Bot 服务
	go run ./cmd/bot

run-api: ## 运行 API 服务
	go run ./cmd/api

# ==================== 测试 ====================

test: ## 运行所有测试
	go test ./... -v -count=1

test-coverage: ## 运行测试并生成覆盖率报告
	go test ./... -v -coverprofile=coverage.out -count=1
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# ==================== 代码质量 ====================

lint: ## 代码静态检查
	go vet ./...
	@echo "go vet 检查通过"

fmt: ## 格式化代码
	gofmt -w .
	@echo "代码格式化完成"

# ==================== 数据库 ====================

migrate-up: ## 执行数据库迁移 (向上)
	go run ./cmd/migrate up

migrate-down: ## 回滚数据库迁移 (向下)
	go run ./cmd/migrate down

migrate-create: ## 创建新的迁移文件 (用法: make migrate-create NAME=xxx)
	@if [ -z "$(NAME)" ]; then echo "请指定迁移名称: make migrate-create NAME=xxx"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(NAME)

# ==================== Docker ====================

docker-up: ## 启动所有 Docker 服务
	docker-compose -f deployments/docker-compose.yml up -d

docker-down: ## 停止所有 Docker 服务
	docker-compose -f deployments/docker-compose.yml down

docker-build: ## 构建 Docker 镜像
	docker-compose -f deployments/docker-compose.yml build

docker-logs: ## 查看 Docker 日志
	docker-compose -f deployments/docker-compose.yml logs -f

# ==================== 前端 ====================

web-install: ## 安装前端依赖
	cd web && npm install

web-dev: ## 启动前端开发服务器
	cd web && npm run dev

web-build: ## 构建前端生产版本
	cd web && npm run build

web-test: ## 运行前端测试
	cd web && npm run test

# ==================== 清理 ====================

clean: ## 清理构建产物
	rm -rf bin/ coverage.out coverage.html
	@echo "清理完成"

# ==================== Git 自动化 ====================

git-init: ## 初始化 Git 仓库 + 创建 GitHub 远程仓库
	@bash scripts/git-init.sh

review: ## 审查当前代码变更 (不提交)
	@bash scripts/auto-review-commit.sh review

commit: ## 自动审查 + 提交 + 推送到 main
	@bash scripts/auto-review-commit.sh commit

commit-msg: ## 使用自定义消息提交 (用法: make commit-msg MSG="feat: xxx")
	@if [ -z "$(MSG)" ]; then echo "请指定提交消息: make commit-msg MSG=\"feat: xxx\""; exit 1; fi
	@bash scripts/auto-review-commit.sh commit "$(MSG)"

push: ## 推送当前分支到远程
	@git push origin main

git-status: ## 查看 Git 状态
	@git status -sb

git-log: ## 查看最近 10 条提交记录
	@git log --oneline --graph --decorate -10
