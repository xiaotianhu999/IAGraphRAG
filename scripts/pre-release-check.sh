#!/bin/bash

# aiplusall-kb Pre-Release Verification Script
# This script performs comprehensive checks before production deployment

set -e

ENVIRONMENT=${1:-"staging"}
VERSION=$(cat VERSION)
FAILED_CHECKS=()
PASSED_CHECKS=()

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    PASSED_CHECKS+=("$1")
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    FAILED_CHECKS+=("$1")
}

# Check functions
check_go_version() {
    log_info "检查 Go 版本..."
    REQUIRED_GO_VERSION="1.24"
    CURRENT_GO_VERSION=$(go version | awk '{print $3}' | cut -d'.' -f1-2 | sed 's/go//')
    
    if [[ $(echo -e "$CURRENT_GO_VERSION\n$REQUIRED_GO_VERSION" | sort -V | head -n1) == "$REQUIRED_GO_VERSION" ]]; then
        log_success "Go 版本检查: $CURRENT_GO_VERSION (>= $REQUIRED_GO_VERSION)"
    else
        log_error "Go 版本过低: $CURRENT_GO_VERSION (需要 >= $REQUIRED_GO_VERSION)"
    fi
}

check_docker_version() {
    log_info "检查 Docker 版本..."
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | cut -d'.' -f1-2 | sed 's/,//')
    REQUIRED_DOCKER_VERSION="20.10"
    
    if [[ $(echo -e "$DOCKER_VERSION\n$REQUIRED_DOCKER_VERSION" | sort -V | head -n1) == "$REQUIRED_DOCKER_VERSION" ]]; then
        log_success "Docker 版本检查: $DOCKER_VERSION (>= $REQUIRED_DOCKER_VERSION)"
    else
        log_error "Docker 版本过低: $DOCKER_VERSION (需要 >= $REQUIRED_DOCKER_VERSION)"
    fi
}

check_dependencies() {
    log_info "检查项目依赖..."
    
    # Check go mod tidy
    if go mod tidy &>/dev/null; then
        log_success "Go 依赖检查通过"
    else
        log_error "Go 依赖检查失败"
    fi
    
    # Check for outdated dependencies
    OUTDATED_DEPS=$(go list -u -m all | grep '\[' | wc -l)
    if [ "$OUTDATED_DEPS" -gt 0 ]; then
        log_warning "发现 $OUTDATED_DEPS 个过时的依赖包"
    else
        log_success "所有依赖包都是最新版本"
    fi
}

check_code_format() {
    log_info "检查代码格式..."
    
    # Check for gofmt
    UNFORMATTED=$(find . -name "*.go" -not -path "./vendor/*" -exec gofmt -l {} + | wc -l)
    if [ "$UNFORMATTED" -eq 0 ]; then
        log_success "代码格式检查通过"
    else
        log_error "发现 $UNFORMATTED 个格式不正确的文件"
    fi
}

check_linting() {
    log_info "运行代码检查 (golangci-lint)..."
    
    if command -v golangci-lint &> /dev/null; then
        if golangci-lint run --timeout=5m &>/dev/null; then
            log_success "代码检查通过"
        else
            log_warning "代码检查发现问题，请手动检查"
        fi
    else
        log_warning "golangci-lint 未安装，跳过"
    fi
}

check_security_scan() {
    log_info "检查安全漏洞..."
    
    if command -v gosec &> /dev/null; then
        if gosec ./... &>/dev/null; then
            log_success "安全检查通过"
        else
            log_warning "安全检查发现潜在问题"
        fi
    else
        log_warning "gosec 未安装，建议安装: go install github.com/securego/gosec/v2/cmd/gosec@latest"
    fi
}

check_unit_tests() {
    log_info "运行单元测试..."
    
    if go test -v ./... -timeout=10m &>/dev/null; then
        log_success "单元测试全部通过"
    else
        log_error "单元测试失败"
    fi
}

check_dockerfile() {
    log_info "检查 Dockerfile..."
    
    if command -v hadolint &> /dev/null; then
        DOCKERFILES=$(find docker -name "Dockerfile*")
        for df in $DOCKERFILES; do
            if hadolint "$df" &>/dev/null; then
                log_success "$df 检查通过"
            else
                log_warning "$df 检查发现问题"
            fi
        done
    else
        log_warning "hadolint 未安装，建议安装进行 Dockerfile 检查"
    fi
}

check_configuration() {
    log_info "检查配置文件..."
    
    # Check if all config files exist
    CONFIG_FILES=(
        "config/config.yaml"
        "config/config-ws.yaml"
        "config/config-raw.yaml"
    )
    
    for config in "${CONFIG_FILES[@]}"; do
        if [ -f "$config" ]; then
            log_success "$config 存在"
        else
            log_warning "$config 不存在"
        fi
    done
}

check_environment_setup() {
    log_info "检查环境文件..."
    
    if [ -f ".env.production.template" ]; then
        log_success ".env.production.template 存在"
    else
        log_error ".env.production.template 不存在"
    fi
}

check_migrations() {
    log_info "检查数据库迁移文件..."
    
    MIGRATIONS=$(find migrations -name "*.sql" 2>/dev/null | wc -l)
    if [ "$MIGRATIONS" -gt 0 ]; then
        log_success "发现 $MIGRATIONS 个迁移文件"
    else
        log_warning "未发现迁移文件"
    fi
}

check_documentation() {
    log_info "检查文档完整性..."
    
    DOCS=(
        "README.md"
        "PRODUCTION_DEPLOYMENT_GUIDE.md"
        "CHANGELOG.md"
        "SECURITY.md"
    )
    
    for doc in "${DOCS[@]}"; do
        if [ -f "$doc" ]; then
            log_success "$doc 存在"
        else
            log_error "$doc 缺失"
        fi
    done
}

check_api_documentation() {
    log_info "检查 API 文档..."
    
    if [ -f "docs/swagger.yaml" ] || [ -f "docs/swagger.json" ]; then
        log_success "API 文档存在"
    else
        log_warning "Swagger 文档缺失，建议运行: make docs"
    fi
}

check_version_consistency() {
    log_info "检查版本号一致性..."
    
    VERSION_FILE=$(cat VERSION)
    HELM_VERSION=$(grep "appVersion:" helm/Chart.yaml | awk '{print $2}' | sed 's/"//g')
    
    if [[ "$VERSION_FILE" == "$HELM_VERSION" ]]; then
        log_success "版本号一致性检查通过: $VERSION_FILE"
    else
        log_warning "版本号不一致 - VERSION: $VERSION_FILE, Helm: $HELM_VERSION"
    fi
}

check_git_status() {
    log_info "检查 Git 状态..."
    
    if [ -z "$(git status --porcelain)" ]; then
        log_success "工作目录干净，无未提交更改"
    else
        log_error "工作目录有未提交的更改:"
        git status --short
    fi
}

check_git_tags() {
    log_info "检查 Git 标签..."
    
    TAG="v${VERSION}"
    if ! git rev-parse "$TAG" >/dev/null 2>&1; then
        log_warning "标签 $TAG 不存在，需要在发布时创建"
    else
        log_success "标签 $TAG 已存在"
    fi
}

check_docker_images() {
    log_info "检查 Docker 镜像..."
    
    if docker image ls | grep -q "aiplusall-kb-app"; then
        log_success "aiplusall-kb-app 镜像存在"
    else
        log_warning "aiplusall-kb-app 镜像不存在，需要构建"
    fi
}

check_docker_compose() {
    log_info "检查 docker-compose 配置..."
    
    if docker-compose config &>/dev/null; then
        log_success "docker-compose 配置有效"
    else
        log_error "docker-compose 配置无效"
    fi
}

check_helm_chart() {
    log_info "检查 Helm Chart..."
    
    if command -v helm &> /dev/null; then
        if helm lint helm/ &>/dev/null; then
            log_success "Helm Chart 检查通过"
        else
            log_warning "Helm Chart 检查发现问题"
        fi
    else
        log_warning "Helm 未安装"
    fi
}

check_performance() {
    log_info "检查代码性能指标..."
    
    # Run benchmarks if available
    if go test -bench=. -benchmem ./... -timeout=5m &>/dev/null; then
        log_success "性能测试运行完成"
    else
        log_warning "未找到性能测试"
    fi
}

check_api_stability() {
    log_info "检查 API 稳定性..."
    
    # This would require a running application
    if [ "$ENVIRONMENT" == "staging" ]; then
        log_info "跳过 API 稳定性检查（需要运行中的应用）"
    fi
}

display_summary() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}预发布检查总结${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    
    echo ""
    echo -e "${GREEN}通过的检查 (${#PASSED_CHECKS[@]}):${NC}"
    for check in "${PASSED_CHECKS[@]}"; do
        echo "  ✓ $check"
    done
    
    if [ ${#FAILED_CHECKS[@]} -gt 0 ]; then
        echo ""
        echo -e "${RED}失败的检查 (${#FAILED_CHECKS[@]}):${NC}"
        for check in "${FAILED_CHECKS[@]}"; do
            echo "  ✗ $check"
        done
        
        echo ""
        echo -e "${RED}检查未通过，请解决上述问题后重试${NC}"
        return 1
    else
        echo ""
        echo -e "${GREEN}✓ 所有检查都已通过，项目可以进行发布${NC}"
        echo ""
        echo -e "${BLUE}建议的下一步:${NC}"
        echo "  1. 运行发布脚本: ./scripts/release.sh $VERSION"
        echo "  2. 推送镜像到仓库"
        echo "  3. 部署到生产环境"
        return 0
    fi
}

# Main execution
main() {
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}aiplusall-kb 预发布检查${NC}"
    echo -e "${BLUE}版本: $VERSION${NC}"
    echo -e "${BLUE}环境: $ENVIRONMENT${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    
    check_go_version
    check_docker_version
    check_dependencies
    check_code_format
    check_linting
    check_security_scan
    check_unit_tests
    check_dockerfile
    check_configuration
    check_environment_setup
    check_migrations
    check_documentation
    check_api_documentation
    check_version_consistency
    check_git_status
    check_git_tags
    check_docker_images
    check_docker_compose
    check_helm_chart
    check_performance
    check_api_stability
    
    display_summary
}

main "$@"
