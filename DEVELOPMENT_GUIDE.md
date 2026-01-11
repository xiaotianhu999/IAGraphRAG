# aiplusall-kb 开发环境启动指南

## 前提条件

1. **Docker Desktop** 已安装并运行
2. **Go 1.24+** 已安装
3. **Node.js 18+** 和 **npm** 已安装
4. **golang-migrate** 工具已安装
   ```cmd
   go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

## 快速启动 (推荐)

### 方式一：一键启动脚本
```cmd
quick-dev.bat
```

这个脚本会：
- 启动所有基础设施服务
- 询问是否在新窗口启动后端和前端
- 显示所有访问地址

### 方式二：分步启动

#### 1. 启动基础设施服务
```cmd
docker-compose -f docker-compose.dev.yml --profile full up -d
```

#### 2. 检查服务状态
```cmd
docker-compose -f docker-compose.dev.yml ps
```

#### 3. 运行数据库迁移
```cmd
migrate.bat up
```

#### 4. 启动后端 (新终端窗口)
```cmd
go run cmd/server/main.go
```

#### 5. 启动前端 (新终端窗口)
```cmd
cd frontend
npm install
npm run dev
```

## 服务访问地址

启动完成后，你可以访问：

- **前端应用**: http://localhost:5173
- **后端API**: http://localhost:8080
- **MinIO控制台**: http://localhost:9001 (minioadmin/kingyan888)
- **Neo4j浏览器**: http://localhost:7474 (neo4j/kingyan888)
- **Jaeger UI**: http://localhost:16686
- **Qdrant**: http://localhost:6333

## 开发工作流

### 后端开发
- 修改 Go 代码后，程序会自动重启（如果安装了 Air）
- API 文档：http://localhost:8080/swagger/index.html

### 前端开发
- 修改 Vue 代码后，页面会自动热重载
- 开发服务器：http://localhost:5173

### 数据库操作
```cmd
# 查看当前迁移版本
migrate.bat version

# 应用所有迁移
migrate.bat up

# 回滚最后一个迁移
migrate.bat down

# 创建新迁移
migrate.bat create add_new_feature
```

## 常用命令

### Docker 服务管理
```cmd
# 查看服务状态
docker-compose -f docker-compose.dev.yml ps

# 查看服务日志
docker-compose -f docker-compose.dev.yml logs -f

# 停止所有服务
docker-compose -f docker-compose.dev.yml down

# 重启服务
docker-compose -f docker-compose.dev.yml restart
```

### 项目重置
```cmd
# 重置项目到初始状态（清理所有数据）
reset-project.bat

# 完全重置（包括Docker镜像）
reset-project-full.bat
```

## 故障排除

### 1. docreader 服务启动失败
- 确保 docreader 镜像已构建：`docker images | findstr docreader`
- 如果镜像不存在，运行：`docker build -f docker/Dockerfile.docreader -t aiplusall/aiplusall-kb-docreader:latest .`

### 2. 数据库连接失败
- 检查 PostgreSQL 容器是否运行：`docker ps | findstr postgres`
- 检查 .env 文件中的数据库配置
- 确保数据库迁移已运行：`migrate.bat version`

### 3. 前端无法访问后端
- 检查后端是否在 8080 端口运行
- 检查 .env 文件中的端口配置
- 确保防火墙没有阻止端口访问

### 4. MinIO 访问失败
- 检查 MinIO 容器是否运行：`docker ps | findstr minio`
- 使用默认凭据：minioadmin/kingyan888

## 环境配置

主要配置文件：
- `.env` - 环境变量配置
- `config/config.yaml` - 应用配置
- `docker-compose.dev.yml` - 开发环境服务配置

关键环境变量：
- `DB_HOST=localhost` - 数据库主机
- `DB_PORT=5432` - 数据库端口
- `DB_USER=legalmind` - 数据库用户
- `DB_PASSWORD=kingyan888` - 数据库密码
- `DB_NAME=legalmind` - 数据库名称
- `REDIS_PASSWORD=kingyan888` - Redis密码
- `MINIO_ACCESS_KEY_ID=minioadmin` - MinIO访问密钥
- `MINIO_SECRET_ACCESS_KEY=kingyan888` - MinIO密钥

## 开发提示

1. **热重载**: 安装 Air 工具实现 Go 代码热重载
   ```cmd
   go install github.com/air-verse/air@latest
   ```

2. **API文档**: 修改 API 后重新生成文档
   ```cmd
   swag init -g cmd/server/main.go -o ./docs
   ```

3. **代码格式化**: 提交前格式化代码
   ```cmd
   go fmt ./...
   ```

4. **依赖管理**: 添加新依赖后
   ```cmd
   go mod tidy
   ```

## 架构说明

- **后端**: Go + Gin + GORM + PostgreSQL
- **前端**: Vue 3 + TypeScript + Vite + TDesign
- **文档处理**: Python + gRPC + PaddleOCR
- **向量存储**: PostgreSQL pgvector / Qdrant
- **对象存储**: MinIO
- **缓存**: Redis
- **图数据库**: Neo4j (可选)
- **链路追踪**: Jaeger (可选)

开发环境采用混合模式：
- 基础设施服务（数据库、缓存等）运行在容器中
- 应用服务（后端、前端）运行在本地，便于开发和调试