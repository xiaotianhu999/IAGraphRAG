# 项目重命名方案文档

## 📋 变更概述

**变更内容：** 将项目从 `github.com/Tencent/WeKnora` 重命名为 `aiplusall/GraphRAG`

**变更范围：** 
- Go Module 路径
- 所有 Go 源文件中的 import 语句
- Docker 镜像和容器名称
- 构建脚本和配置文件
- 文档和说明文件

---

## ⚠️ 重要说明

### 风险评估：中高风险
1. **Go Module 路径变更** 会影响所有 255+ 个 Go 源文件
2. **Import 路径变更** 需要确保全部替换，否则编译失败
3. **Docker 配置变更** 需要重新构建镜像
4. **需要充分测试** 确保项目能正常编译和运行

### 建议备份策略
```bash
# 执行前请先备份或创建 Git 分支
git checkout -b rename-to-graphrag
git add -A
git commit -m "Backup before rename to GraphRAG"
```

---

## 📊 影响范围统计

### 核心代码文件（必须修改）

#### 1. Go Module 配置
- **文件数量：** 2 个
- **文件清单：**
  - `/go.mod` - 主项目 Module 定义
  - `/client/go.mod` - 客户端库 Module 定义

#### 2. Go 源文件 Import 语句
- **文件数量：** 约 255 个 .go 文件
- **影响行数：** 预计 1000+ 行
- **替换内容：** `github.com/Tencent/WeKnora` → `aiplusall/GraphRAG`
- **典型文件：**
  - `/internal/**/*.go` - 所有内部包
  - `/cmd/server/main.go` - 主程序入口
  - `/docreader/client/*.go` - 文档读取器客户端

#### 3. 构建和编译配置
- **文件数量：** 3 个
- **文件清单：**
  - `/Makefile` - 构建目标中的 LDFLAGS
  - `/scripts/get_version.sh` - 版本信息注入脚本
  - `/docker/Dockerfile.app` - 应用 Docker 镜像构建文件

### Docker 相关（建议修改）

#### 4. Docker Compose 配置
- **文件数量：** 2 个
- **变更内容：**
  ```yaml
  # 镜像名称变更
  wechatopenai/weknora-app → wechatopenai/graphrag-app
  wechatopenai/weknora-docreader → wechatopenai/graphrag-docreader
  wechatopenai/weknora-ui → wechatopenai/graphrag-ui
  
  # 容器名称变更
  WeKnora-* → GraphRAG-*
  
  # 网络名称变更
  WeKnora-network → GraphRAG-network
  
  # 环境变量
  OTEL_SERVICE_NAME=WeKnora → OTEL_SERVICE_NAME=GraphRAG
  ```
- **文件清单：**
  - `/docker-compose.yml`
  - `/docker-compose.dev.yml`

#### 5. Docker 构建脚本
- **文件数量：** 1 个
- **文件清单：**
  - `/scripts/build_images.sh` - 镜像构建和管理脚本

### 文档和配置（建议修改）

#### 6. 项目文档
- **文件数量：** 约 20+ 个 Markdown 文件
- **变更内容：**
  - GitHub 仓库链接
  - 项目名称引用
  - 安装和使用说明
- **主要文件：**
  - `/README.md`
  - `/README_CN.md`
  - `/README_JA.md`
  - `/docs/*.md`

#### 7. Helm Chart 配置
- **文件数量：** 1 个
- **变更内容：**
  ```yaml
  name: weknora → graphrag
  home: https://github.com/Tencent/WeKnora → https://github.com/aiplusall/GraphRAG
  ```
- **文件清单：**
  - `/helm/Chart.yaml`

#### 8. Python MCP Server
- **文件数量：** 1 个
- **文件清单：**
  - `/mcp-server/pyproject.toml` - 包名称和元数据

#### 9. 其他配置文件
- `/config/config-raw.yaml` - AI 助手名称（可选）
- `/.env` - 示例配置注释（可选）
- `/test_agent_config.sh` - 数据库名称（可选）

---

## 🔧 详细修改步骤

### 阶段一：核心代码修改（必须执行）

#### Step 1: 修改 Go Module 配置
```bash
# 1.1 主项目 go.mod
修改第 1 行：
module github.com/Tencent/WeKnora
→
module aiplusall/GraphRAG

# 1.2 客户端 go.mod
修改 client/go.mod 第 1 行：
module github.com/Tencent/WeKnora/client
→
module aiplusall/GraphRAG/client
```

#### Step 2: 批量替换所有 Go 文件中的 Import 路径
```bash
# 使用查找替换功能（推荐使用 VS Code 全局替换）
查找：github.com/Tencent/WeKnora
替换为：aiplusall/GraphRAG

影响范围：
- /cmd/**/*.go
- /internal/**/*.go
- /docreader/client/*.go
- 所有其他 .go 文件
```

#### Step 3: 修改构建配置
```bash
# 3.1 Makefile (line 227)
LDFLAGS 中的 4 处：
github.com/Tencent/WeKnora/internal/handler
→
aiplusall/GraphRAG/internal/handler

# 3.2 scripts/get_version.sh (line 68)
4 处相同替换
```

#### Step 4: 清理和重新生成依赖
```bash
# 删除旧的依赖缓存
rm -rf go.sum
rm -rf vendor/

# 重新整理依赖
go mod tidy

# 验证编译
go build ./cmd/server
```

### 阶段二：Docker 配置修改（建议执行）

#### Step 5: 修改 Docker Compose 文件
```yaml
# docker-compose.yml 和 docker-compose.dev.yml

# 镜像名称（3 处）
image: wechatopenai/weknora-app:latest
→ wechatopenai/graphrag-app:latest

image: wechatopenai/weknora-docreader:latest
→ wechatopenai/graphrag-docreader:latest

image: wechatopenai/weknora-ui:latest
→ wechatopenai/graphrag-ui:latest

# 容器名称（约 8 处）
container_name: WeKnora-*
→ container_name: GraphRAG-*

# 网络名称
networks:
  - WeKnora-network
→ networks:
  - GraphRAG-network

# 环境变量
OTEL_SERVICE_NAME=WeKnora
→ OTEL_SERVICE_NAME=GraphRAG
```

#### Step 6: 修改构建脚本
```bash
# scripts/build_images.sh
# 查找所有 weknora 相关的镜像名称并替换
weknora-app → graphrag-app
weknora-docreader → graphrag-docreader
weknora-ui → graphrag-ui
WeKnora → GraphRAG
```

#### Step 7: 修改 Makefile Docker 相关命令
```makefile
# 在 Makefile 中搜索并替换所有 weknora 镜像名称
wechatopenai/weknora-* → wechatopenai/graphrag-*
```

### 阶段三：文档和配置修改（可选但推荐）

#### Step 8: 更新主要文档
```markdown
# README.md, README_CN.md, README_JA.md

# 查找替换：
WeKnora → GraphRAG
weknora → graphrag
github.com/Tencent/WeKnora → github.com/aiplusall/GraphRAG
https://github.com/Tencent/WeKnora → https://github.com/aiplusall/GraphRAG
```

#### Step 9: 更新 Helm Chart
```yaml
# helm/Chart.yaml
name: weknora → name: graphrag
home: https://github.com/Tencent/WeKnora → https://github.com/aiplusall/GraphRAG
sources:
  - https://github.com/Tencent/WeKnora → https://github.com/aiplusall/GraphRAG
```

#### Step 10: 更新其他文档
```bash
# docs/ 目录下所有 .md 文件
查找：WeKnora
替换为：GraphRAG

# 保留一些特定的历史引用（如果需要）
```

---

## ✅ 验证清单

### 编译验证
```bash
# 1. 清理构建缓存
go clean -cache -modcache -i -r

# 2. 重新下载依赖
go mod download
go mod tidy

# 3. 编译主程序
go build -o tmp/test_build ./cmd/server
echo "编译成功，执行文件: tmp/test_build"

# 4. 验证导入路径
go list -m all | grep -i "weknora"
# 应该没有任何 weknora 相关的输出

# 5. 检查是否有遗漏的旧路径
grep -r "github.com/Tencent/WeKnora" --include="*.go" .
# 应该没有任何匹配结果
```

### Docker 验证
```bash
# 1. 停止并清理旧容器
docker-compose down
docker system prune -f

# 2. 构建新镜像
docker-compose build

# 3. 启动服务
docker-compose up -d

# 4. 检查服务状态
docker-compose ps
```

### 运行时验证
```bash
# 1. 启动开发环境
./scripts/dev.sh start

# 2. 检查应用是否正常启动
curl http://localhost:8080/health

# 3. 检查日志是否有错误
./scripts/dev.sh logs
```

---

## 🔄 回滚方案

如果修改出现问题，可以通过以下方式回滚：

### 方式一：Git 回滚（推荐）
```bash
# 回到修改前的状态
git checkout main  # 或原分支名
git branch -D rename-to-graphrag  # 删除改名分支

# 或使用 reset
git reset --hard HEAD~1
```

### 方式二：手动回滚
```bash
# 反向执行所有替换
查找：aiplusall/GraphRAG
替换为：github.com/Tencent/WeKnora

查找：graphrag
替换为：weknora

查找：GraphRAG
替换为：WeKnora
```

---

## 📝 执行后注意事项

### 1. 更新 Git 远程仓库
```bash
# 如果需要推送到新的 GitHub 仓库
git remote set-url origin https://github.com/aiplusall/GraphRAG.git
git push -u origin main
```

### 2. 更新 CI/CD 配置
- 更新所有持续集成配置文件
- 更新镜像仓库地址
- 更新部署脚本

### 3. 通知团队成员
- 通知所有开发者克隆新的仓库
- 更新文档和 wiki
- 更新依赖此项目的其他项目

### 4. 清理开发环境
```bash
# 每个开发者需要：
cd <项目目录>
git pull
go clean -modcache
go mod tidy
go mod download
```

---

## 📌 执行建议

### 推荐执行顺序：
1. ✅ **先执行 阶段一**（核心代码）- 必须
2. ✅ **验证编译通过** - 必须
3. ✅ **执行 阶段二**（Docker）- 强烈推荐
4. ✅ **执行 阶段三**（文档）- 推荐
5. ✅ **全面测试** - 必须
6. ✅ **Git 提交** - 必须

### 时间估算：
- **准备和备份：** 5 分钟
- **代码修改：** 10-15 分钟（自动化替换）
- **编译验证：** 5 分钟
- **Docker 构建：** 10-20 分钟
- **功能测试：** 20-30 分钟
- **总计：** 约 1-1.5 小时

---

## ❓ 常见问题

### Q1: 修改后无法编译？
**A:** 检查是否有遗漏的 import 路径：
```bash
grep -r "github.com/Tencent/WeKnora" . --include="*.go"
```

### Q2: Docker 容器无法启动？
**A:** 检查 docker-compose.yml 中的所有名称是否已更新，清理旧容器后重试。

### Q3: 依赖下载失败？
**A:** 如果 aiplusall/GraphRAG 还不存在，需要先推送到 GitHub，或使用 replace 指令：
```go
// 在 go.mod 中临时添加
replace aiplusall/GraphRAG => ./
```

### Q4: 需要保持向后兼容吗？
**A:** 如果需要，可以在 go.mod 中使用 replace 同时支持两个路径。

---

## ✨ 确认后即可开始执行

请仔细阅读以上方案，确认无误后回复 **"确认执行"**，我将开始按照此方案进行修改。

建议在执行前：
1. ✅ 创建 Git 备份分支
2. ✅ 确保当前代码可以正常编译运行
3. ✅ 准备好回滚方案

---

**文档生成时间：** 2026-01-02  
**预计修改文件数：** 280+ 个  
**预计修改行数：** 1500+ 行
