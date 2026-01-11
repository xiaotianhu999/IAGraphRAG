.PHONY: help build run test clean docker-build-app docker-build-docreader docker-build-frontend docker-build-all docker-run migrate-up migrate-down docker-restart docker-stop start-all stop-all start-ollama stop-ollama build-images build-images-app build-images-docreader build-images-frontend clean-images check-env list-containers pull-images show-platform dev-start dev-stop dev-restart dev-logs dev-status dev-app dev-frontend docs install-swagger

# Show help
help:
	@echo "aiplusall-kb Makefile 帮助"
	@echo ""
	@echo "基础命令:"
	@echo "  build             构建应用"
	@echo "  run               运行应用"
	@echo "  test              运行测试"
	@echo "  clean             清理构建文件"
	@echo ""
	@echo "Docker 命令:"
	@echo "  docker-build-app       构建应用 Docker 镜像 (aiplusall/aiplusall-kb-app)"
	@echo "  docker-build-docreader 构建文档读取器镜像 (aiplusall/aiplusall-kb-docreader)"
	@echo "  docker-build-frontend  构建前端镜像 (aiplusall/aiplusall-kb-ui)"
	@echo "  docker-build-all       构建所有 Docker 镜像"
	@echo "  docker-run            运行 Docker 容器"
	@echo "  docker-stop           停止 Docker 容器"
	@echo "  docker-restart        重启 Docker 容器"
	@echo ""
	@echo "服务管理:"
	@echo "  start-all         启动所有服务"
	@echo "  stop-all          停止所有服务"
	@echo "  start-ollama      仅启动 Ollama 服务"
	@echo ""
	@echo "镜像构建:"
	@echo "  build-images      从源码构建所有镜像"
	@echo "  build-images-app  从源码构建应用镜像"
	@echo "  build-images-docreader 从源码构建文档读取器镜像"
	@echo "  build-images-frontend  从源码构建前端镜像"
	@echo "  clean-images      清理本地镜像"
	@echo ""
	@echo "数据库:"
	@echo "  migrate-up        执行数据库迁移"
	@echo "  migrate-down      回滚数据库迁移"
	@echo "  clean-db          清理数据库卷"
	@echo "  reset-project     重置项目到初始状态（清理所有数据）"
	@echo "  reset-project-full 完全重置（包括Docker镜像）"
	@echo ""
	@echo "开发工具:"
	@echo "  fmt               格式化代码"
	@echo "  lint              代码检查"
	@echo "  deps              安装依赖"
	@echo "  docs              生成 Swagger API 文档"
	@echo "  install-swagger   安装 swag 工具"
	@echo ""
	@echo "环境检查:"
	@echo "  check-env         检查环境配置"
	@echo "  list-containers   列出运行中的容器"
	@echo "  pull-images       拉取最新镜像"
	@echo "  show-platform     显示当前构建平台"
	@echo ""
	@echo "开发模式（推荐）:"
	@echo "  dev-start         启动开发环境基础设施（仅启动依赖服务）"
	@echo "  dev-stop          停止开发环境"
	@echo "  dev-restart       重启开发环境"
	@echo "  dev-logs          查看开发环境日志"
	@echo "  dev-status        查看开发环境状态"
	@echo "  dev-app           启动后端应用（本地运行，需先运行 dev-start）"
	@echo "  dev-frontend      启动前端（本地运行，需先运行 dev-start）"

# Go related variables
BINARY_NAME=aiplusall-kb
MAIN_PATH=./cmd/server

# Docker related variables
DOCKER_IMAGE=aiplusall/aiplusall-kb-app
DOCKER_TAG=latest

# Platform detection
ifeq ($(shell uname -m),x86_64)
    PLATFORM=linux/amd64
else ifeq ($(shell uname -m),aarch64)
    PLATFORM=linux/arm64
else ifeq ($(shell uname -m),arm64)
    PLATFORM=linux/arm64
else
    PLATFORM=linux/amd64
endif

# Build the application
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: build
	./$(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Build Docker image
docker-build-app:
	@echo "获取版本信息..."
	@eval $$(./scripts/get_version.sh env); \
	./scripts/get_version.sh info; \
	docker build --platform $(PLATFORM) \
		--build-arg VERSION_ARG="$$VERSION" \
		--build-arg COMMIT_ID_ARG="$$COMMIT_ID" \
		--build-arg BUILD_TIME_ARG="$$BUILD_TIME" \
		--build-arg GO_VERSION_ARG="$$GO_VERSION" \
		-f docker/Dockerfile.app -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Build docreader Docker image
docker-build-docreader:
	docker build --platform $(PLATFORM) -f docker/Dockerfile.docreader -t aiplusall/aiplusall-kb-docreader:latest .

# Build frontend Docker image
docker-build-frontend:
	docker build --platform $(PLATFORM) -f frontend/Dockerfile -t aiplusall/aiplusall-kb-ui:latest frontend/

# Build all Docker images
docker-build-all: docker-build-app docker-build-docreader docker-build-frontend

# Run Docker container (传统方式)
docker-run:
	docker-compose up

# 使用新脚本启动所有服务
start-all:
	./scripts/start_all.sh

# 使用新脚本仅启动Ollama服务
start-ollama:
	./scripts/start_all.sh --ollama

# 使用新脚本仅启动Docker容器
start-docker:
	./scripts/start_all.sh --docker

# 使用新脚本停止所有服务
stop-all:
	./scripts/start_all.sh --stop

# Stop Docker container (传统方式)
docker-stop:
	docker-compose down

# 从源码构建镜像相关命令
build-images:
	./scripts/build_images.sh

build-images-app:
	./scripts/build_images.sh --app

build-images-docreader:
	./scripts/build_images.sh --docreader

build-images-frontend:
	./scripts/build_images.sh --frontend

clean-images:
	./scripts/build_images.sh --clean

# Restart Docker container (stop, start)
docker-restart:
	docker-compose stop -t 60
	docker-compose up

# Database migrations
migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

migrate-version:
	./scripts/migrate.sh version

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required"; \
		echo "Usage: make migrate-create name=your_migration_name"; \
		exit 1; \
	fi
	./scripts/migrate.sh create $(name)

migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-force version=4"; \
		exit 1; \
	fi
	./scripts/migrate.sh force $(version)

migrate-goto:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-goto version=3"; \
		exit 1; \
	fi
	./scripts/migrate.sh goto $(version)

# Generate API documentation (Swagger)
docs:
	@echo "生成 Swagger API 文档..."
	swag init -g $(MAIN_PATH)/main.go -o ./docs --parseDependency --parseInternal
	@echo "文档已生成到 ./docs 目录"
	@echo "启动服务后访问 http://localhost:8080/swagger/index.html 查看文档"

# Install swagger tool
install-swagger:
	go install github.com/swaggo/swag/cmd/swag@latest

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download

# Build for production
build-prod:
	VERSION=$${VERSION:-unknown}; \
	COMMIT_ID=$${COMMIT_ID:-unknown}; \
	BUILD_TIME=$${BUILD_TIME:-unknown}; \
	GO_VERSION=$${GO_VERSION:-unknown}; \
	LDFLAGS="-X 'github.com/aiplusall/aiplusall-kb/internal/handler.Version=$$VERSION' -X 'github.com/aiplusall/aiplusall-kb/internal/handler.CommitID=$$COMMIT_ID' -X 'github.com/aiplusall/aiplusall-kb/internal/handler.BuildTime=$$BUILD_TIME' -X 'github.com/aiplusall/aiplusall-kb/internal/handler.GoVersion=$$GO_VERSION'"; \
	go build -ldflags="-w -s $$LDFLAGS" -o $(BINARY_NAME) $(MAIN_PATH)

clean-db:
	@echo "Cleaning database..."
	@if [ $$(docker volume ls -q -f name=aiplusall_kb_postgres-data) ]; then \
		docker volume rm weknora_postgres-data; \
	fi
	@if [ $$(docker volume ls -q -f name=weknora_minio_data) ]; then \
		docker volume rm weknora_minio_data; \
	fi
	@if [ $$(docker volume ls -q -f name=weknora_redis_data) ]; then \
		docker volume rm weknora_redis_data; \
	fi

# Reset project to initial state (clean all data)
reset-project:
	@echo "Resetting aiplusall-kb project to initial state..."
	@echo "This will delete ALL data. Press Ctrl+C to cancel, or press Enter to continue..."
	@read dummy
	@echo "Stopping all containers..."
	-docker-compose down --remove-orphans 2>/dev/null || true
	-docker-compose -f docker-compose.dev.yml down --remove-orphans 2>/dev/null || true
	@echo "Removing Docker volumes..."
	-docker volume rm aiplusall-kb_postgres-data 2>/dev/null || true
	-docker volume rm aiplusall-kb_minio-data 2>/dev/null || true
	-docker volume rm aiplusall-kb_redis-data 2>/dev/null || true
	-docker volume rm aiplusall-kb_neo4j-data 2>/dev/null || true
	-docker volume rm weknora_postgres-data 2>/dev/null || true
	-docker volume rm weknora_minio_data 2>/dev/null || true
	-docker volume rm weknora_redis_data 2>/dev/null || true
	-docker volume rm weknora_neo4j_data 2>/dev/null || true
	@echo "Cleaning local storage..."
	-rm -rf ./data/* 2>/dev/null || true
	-rm -rf ./tmp/* 2>/dev/null || true
	-rm -rf ./uploads/* 2>/dev/null || true
	-rm -rf ./files/* 2>/dev/null || true
	-rm -rf ./storage/* 2>/dev/null || true
	@echo "Cleaning temporary files..."
	-rm -f aiplusall-kb 2>/dev/null || true
	-rm -rf frontend/node_modules 2>/dev/null || true
	-rm -rf frontend/dist 2>/dev/null || true
	-find . -name "*.tmp" -type f -delete 2>/dev/null || true
	-find . -name "*.log" -type f -delete 2>/dev/null || true
	@echo "Pruning Docker system..."
	-docker volume prune -f 2>/dev/null || true
	@echo ""
	@echo "✅ Project reset complete!"
	@echo ""
	@echo "To start fresh:"
	@echo "  1. make dev-start    (start infrastructure)"
	@echo "  2. make migrate-up   (setup database)"
	@echo "  3. make dev-app      (start backend)"
	@echo "  4. make dev-frontend (start frontend)"

# Reset project and also remove Docker images
reset-project-full:
	@echo "Full reset: removing all data AND Docker images..."
	@echo "This will delete ALL data and images. Press Ctrl+C to cancel, or press Enter to continue..."
	@read dummy
	@$$(MAKE) reset-project
	@echo "Removing Docker images..."
	-docker rmi wechatopenai/aiplusall-kb-app:latest 2>/dev/null || true
	-docker rmi wechatopenai/aiplusall-kb-docreader:latest 2>/dev/null || true
	-docker rmi wechatopenai/aiplusall-kb-ui:latest 2>/dev/null || true
	-docker image prune -f 2>/dev/null || true
	@echo "✅ Full project reset complete!"

# Environment check
check-env:
	./scripts/start_all.sh --check

# List containers
list-containers:
	./scripts/start_all.sh --list

# Pull latest images
pull-images:
	./scripts/start_all.sh --pull

# Show current platform
show-platform:
	@echo "当前系统架构: $(shell uname -m)"
	@echo "Docker构建平台: $(PLATFORM)"

# Development mode commands
dev-start:
	./scripts/dev.sh start

dev-stop:
	./scripts/dev.sh stop

dev-restart:
	./scripts/dev.sh restart

dev-logs:
	./scripts/dev.sh logs

dev-status:
	./scripts/dev.sh status

dev-app:
	./scripts/dev.sh app

dev-frontend:
	./scripts/dev.sh frontend


