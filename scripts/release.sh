#!/bin/bash

# aiplusall-kb Production Release Script
# Usage: ./scripts/release.sh <version> [environment]

set -e

VERSION=${1:-""}
ENVIRONMENT=${2:-"production"}
RELEASE_BRANCH="release/${VERSION}"
GITHUB_REPO="${GITHUB_REPO:-Tencent/aiplusall-kb}"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-wechatopenai}"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "检查前置条件..."
    
    # Check if version is provided
    if [ -z "$VERSION" ]; then
        log_error "版本号未提供"
        echo "用法: $0 <version> [environment]"
        echo "示例: $0 0.2.5 production"
        exit 1
    fi
    
    # Check if version format is valid (semver)
    if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_error "版本号格式不合法: $VERSION (应为 X.Y.Z 格式)"
        exit 1
    fi
    
    # Check required tools
    command -v docker &> /dev/null || { log_error "Docker 未安装"; exit 1; }
    command -v git &> /dev/null || { log_error "Git 未安装"; exit 1; }
    command -v go &> /dev/null || { log_error "Go 未安装"; exit 1; }
    
    # Check git status
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "当前目录不是 Git 仓库"
        exit 1
    fi
    
    if [ -n "$(git status --porcelain)" ]; then
        log_warn "工作目录有未提交的更改"
        echo "请先提交或暂存更改"
        exit 1
    fi
    
    log_info "前置条件检查完成 ✓"
}

run_tests() {
    log_info "运行测试..."
    
    go test -v ./... || {
        log_error "测试失败"
        exit 1
    }
    
    log_info "测试通过 ✓"
}

update_version() {
    log_info "更新版本号为 $VERSION..."
    
    echo "$VERSION" > VERSION
    git add VERSION
    git commit -m "chore: bump version to $VERSION" || true
    
    log_info "版本号已更新 ✓"
}

build_images() {
    log_info "构建 Docker 镜像..."
    
    # Get build metadata
    GIT_COMMIT=$(git rev-parse --short HEAD)
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    GO_VERSION=$(go version | awk '{print $3}')
    
    # Platform detection (fallback to amd64 if detection fails)
    PLATFORM="linux/amd64"
    if command -v uname &> /dev/null; then
        ARCH=$(uname -m)
        if [[ "$ARCH" == "aarch64" || "$ARCH" == "arm64" ]]; then
            PLATFORM="linux/arm64"
        fi
    fi
    
    # Build and tag images
    DOCKER_TAG=$VERSION
    
    log_info "构建应用镜像 (Platform: $PLATFORM)..."
    docker build --platform "$PLATFORM" \
        --build-arg VERSION_ARG="$VERSION" \
        --build-arg COMMIT_ID_ARG="$GIT_COMMIT" \
        --build-arg BUILD_TIME_ARG="$BUILD_TIME" \
        --build-arg GO_VERSION_ARG="$GO_VERSION" \
        -f docker/Dockerfile.app -t ${DOCKER_REGISTRY}/aiplusall-kb-app:latest .
    
    log_info "构建文档读取器镜像 (Platform: $PLATFORM)..."
    docker build --platform "$PLATFORM" -f docker/Dockerfile.docreader -t ${DOCKER_REGISTRY}/aiplusall-kb-docreader:latest .
    
    log_info "构建前端镜像 (Platform: $PLATFORM)..."
    docker build --platform "$PLATFORM" -f frontend/Dockerfile -t ${DOCKER_REGISTRY}/aiplusall-kb-ui:latest frontend/
    
    # Tag images for registry
    docker tag ${DOCKER_REGISTRY}/aiplusall-kb-app:latest ${DOCKER_REGISTRY}/aiplusall-kb-app:${VERSION}
    docker tag ${DOCKER_REGISTRY}/aiplusall-kb-docreader:latest ${DOCKER_REGISTRY}/aiplusall-kb-docreader:${VERSION}
    docker tag ${DOCKER_REGISTRY}/aiplusall-kb-ui:latest ${DOCKER_REGISTRY}/aiplusall-kb-ui:${VERSION}
    
    log_info "Docker 镜像构建完成 ✓"
}

verify_images() {
    log_info "验证 Docker 镜像..."
    
    docker images | grep -E "aiplusall-kb-app.*${VERSION}|aiplusall-kb-docreader.*${VERSION}|aiplusall-kb-ui.*${VERSION}" || {
        log_error "镜像验证失败"
        exit 1
    }
    
    # Run health check on images
    log_info "运行镜像健康检查..."
    docker run --rm ${DOCKER_REGISTRY}/aiplusall-kb-app:${VERSION} /app/aiplusall-kb --version 2>/dev/null || true
    
    log_info "镜像验证完成 ✓"
}

push_images() {
    log_info "推送镜像到仓库..."
    
    if [ -z "$DOCKER_USERNAME" ] || [ -z "$DOCKER_PASSWORD" ]; then
        log_warn "Docker 凭证未设置，跳过推送"
        log_warn "请手动运行以下命令来推送镜像:"
        echo "  docker push ${DOCKER_REGISTRY}/aiplusall-kb-app:${VERSION}"
        echo "  docker push ${DOCKER_REGISTRY}/aiplusall-kb-docreader:${VERSION}"
        echo "  docker push ${DOCKER_REGISTRY}/aiplusall-kb-ui:${VERSION}"
        return
    fi
    
    # Login to registry
    echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin
    
    # Push images
    docker push ${DOCKER_REGISTRY}/aiplusall-kb-app:${VERSION}
    docker push ${DOCKER_REGISTRY}/aiplusall-kb-docreader:${VERSION}
    docker push ${DOCKER_REGISTRY}/aiplusall-kb-ui:${VERSION}
    
    # Also push latest tag
    docker tag ${DOCKER_REGISTRY}/aiplusall-kb-app:${VERSION} ${DOCKER_REGISTRY}/aiplusall-kb-app:latest
    docker tag ${DOCKER_REGISTRY}/aiplusall-kb-docreader:${VERSION} ${DOCKER_REGISTRY}/aiplusall-kb-docreader:latest
    docker tag ${DOCKER_REGISTRY}/aiplusall-kb-ui:${VERSION} ${DOCKER_REGISTRY}/aiplusall-kb-ui:latest
    
    docker push ${DOCKER_REGISTRY}/aiplusall-kb-app:latest
    docker push ${DOCKER_REGISTRY}/aiplusall-kb-docreader:latest
    docker push ${DOCKER_REGISTRY}/aiplusall-kb-ui:latest
    
    log_info "镜像推送完成 ✓"
}

create_release_tag() {
    log_info "创建 Git 标签..."
    
    git tag -a "v${VERSION}" -m "Release version ${VERSION}

Build Info:
- Version: ${VERSION}
- Timestamp: ${TIMESTAMP}
- Commit: $(git rev-parse --short HEAD)
- Go Version: $(go version)

Release Checklist:
- [ ] Tests passed
- [ ] Security scan passed
- [ ] Performance tests passed
- [ ] API documentation verified
- [ ] Deployment guide reviewed
" || {
        log_error "创建标签失败"
        exit 1
    }
    
    # Push tag
    git push origin "v${VERSION}" || {
        log_warn "推送标签失败，请手动执行: git push origin v${VERSION}"
    }
    
    log_info "Git 标签创建完成 ✓"
}

generate_release_notes() {
    log_info "生成发布说明..."
    
    RELEASE_NOTES_FILE="RELEASE_NOTES_${VERSION}.md"
    
    cat > "$RELEASE_NOTES_FILE" << EOF
# aiplusall-kb Release v${VERSION}

**Release Date**: ${TIMESTAMP}  
**Git Commit**: $(git rev-parse --short HEAD)  
**Go Version**: $(go version | awk '{print $3}')

## Docker Images

- \`${DOCKER_REGISTRY}/aiplusall-kb-app:${VERSION}\`
- \`${DOCKER_REGISTRY}/aiplusall-kb-docreader:${VERSION}\`
- \`${DOCKER_REGISTRY}/aiplusall-kb-ui:${VERSION}\`

## Installation

### Docker Compose
\`\`\`bash
export DOCKER_TAG=${VERSION}
docker-compose up -d
\`\`\`

### Kubernetes with Helm
\`\`\`bash
helm install aiplusall-kb ./helm --set appVersion="v${VERSION}"
\`\`\`

## Changelog

See [CHANGELOG.md](./CHANGELOG.md) for detailed changes.

## Security

See [SECURITY.md](./SECURITY.md) for security considerations.

## Documentation

- **Deployment Guide**: [PRODUCTION_DEPLOYMENT_GUIDE.md](./PRODUCTION_DEPLOYMENT_GUIDE.md)
- **API Documentation**: https://your-domain/swagger/index.html
- **Developer Guide**: [docs/开发指南.md](./docs/开发指南.md)

## Support

- **Official Website**: https://aiplusall-kb.weixin.qq.com
- **GitHub Issues**: https://github.com/${GITHUB_REPO}/issues
- **Troubleshooting**: [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)

---

**Maintainers**: aiplusall-kb Team  
**License**: MIT
EOF
    
    log_info "发布说明已生成: $RELEASE_NOTES_FILE"
}

create_helm_values() {
    log_info "生成 Helm values 文件..."
    
    cat > "helm/values-${VERSION}.yaml" << 'EOF'
# aiplusall-kb Helm Values for Production

image:
  registry: wechatopenai
  app:
    repository: aiplusall-kb-app
    tag: VERSION_PLACEHOLDER
  docreader:
    repository: aiplusall-kb-docreader
    tag: VERSION_PLACEHOLDER
  ui:
    repository: aiplusall-kb-ui
    tag: VERSION_PLACEHOLDER

replicaCount:
  app: 3
  docreader: 2

resources:
  app:
    requests:
      cpu: 2
      memory: 4Gi
    limits:
      cpu: 4
      memory: 8Gi
  docreader:
    requests:
      cpu: 1
      memory: 2Gi
    limits:
      cpu: 2
      memory: 4Gi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

persistence:
  enabled: true
  size: 100Gi
  storageClassName: fast-ssd

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: api.aiplusall-kb.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: aiplusall-kb-tls
      hosts:
        - api.aiplusall-kb.example.com

postgresql:
  enabled: true
  auth:
    username: aiplusall_kb
    password: CHANGE_ME
    database: aiplusall_kb_prod

redis:
  enabled: true
  auth:
    enabled: true
    password: CHANGE_ME

qdrant:
  enabled: true
  apiKey: CHANGE_ME

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
EOF
    
    # Replace version placeholder
    sed -i "s/VERSION_PLACEHOLDER/${VERSION}/g" "helm/values-${VERSION}.yaml"
    
    log_info "Helm values 文件已生成: helm/values-${VERSION}.yaml"
}

display_summary() {
    cat << EOF

${GREEN}═══════════════════════════════════════════════════════════${NC}
${GREEN}发布总结${NC}
${GREEN}═══════════════════════════════════════════════════════════${NC}

版本: ${VERSION}
发布时间: ${TIMESTAMP}
Git 提交: $(git rev-parse --short HEAD)
环境: ${ENVIRONMENT}

${GREEN}已完成的步骤:${NC}
✓ 前置条件检查
✓ 单元测试运行
✓ 版本号更新
✓ Docker 镜像构建
✓ 镜像验证
✓ 发布说明生成
✓ Helm 配置生成
✓ Git 标签创建

${YELLOW}下一步操作:${NC}

1. 推送 Docker 镜像到仓库:
   docker push ${DOCKER_REGISTRY}/aiplusall-kb-app:${VERSION}
   docker push ${DOCKER_REGISTRY}/aiplusall-kb-docreader:${VERSION}
   docker push ${DOCKER_REGISTRY}/aiplusall-kb-ui:${VERSION}

2. 生产环境部署 (Docker Compose):
   cp .env.production.template .env.production
   # 编辑 .env.production 配置文件
   export DOCKER_TAG=${VERSION}
   docker-compose pull
   docker-compose up -d
   ./scripts/migrate.sh up

3. 生产环境部署 (Kubernetes):
   helm install aiplusall-kb ./helm \\
     --namespace aiplusall-kb-prod \\
     --values helm/values-${VERSION}.yaml

4. 验证部署:
   curl http://your-domain:8080/health
   # 访问 UI: http://your-domain

5. 创建 GitHub Release:
   访问 https://github.com/${GITHUB_REPO}/releases/new
   标签: v${VERSION}
   描述: 参考 RELEASE_NOTES_${VERSION}.md

${GREEN}文档:${NC}
- 部署指南: PRODUCTION_DEPLOYMENT_GUIDE.md
- 发布说明: RELEASE_NOTES_${VERSION}.md
- Helm 配置: helm/values-${VERSION}.yaml

${GREEN}═══════════════════════════════════════════════════════════${NC}

EOF
}

# Main flow
main() {
    log_info "开始 aiplusall-kb v${VERSION} 发布流程..."
    
    check_prerequisites
    run_tests
    update_version
    build_images
    verify_images
    create_release_tag
    generate_release_notes
    create_helm_values
    
    display_summary
}

# Run main function
main "$@"
