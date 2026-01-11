#!/bin/bash

# aiplusall-kb Production Deployment Validation Script
# Validates successful deployment and application health

set -e

ENVIRONMENT=${1:-"production"}
TIMEOUT=${2:-300}
HEALTH_CHECK_INTERVAL=5
START_TIME=$(date +%s)

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
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Helper functions
check_service_health() {
    local service=$1
    local endpoint=$2
    local expected_code=${3:-200}
    
    log_info "检查 $service 健康状态: $endpoint"
    
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "$endpoint" || echo "000")
    
    if [ "$RESPONSE" = "$expected_code" ]; then
        log_success "$service 健康检查通过 (HTTP $RESPONSE)"
        return 0
    else
        log_error "$service 健康检查失败 (HTTP $RESPONSE, 期望: $expected_code)"
        return 1
    fi
}

wait_for_service() {
    local service=$1
    local endpoint=$2
    local max_attempts=$((TIMEOUT / HEALTH_CHECK_INTERVAL))
    local attempt=0
    
    log_info "等待 $service 就绪..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s -f "$endpoint" >/dev/null 2>&1; then
            log_success "$service 已就绪"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -ne "\r  尝试 $attempt/$max_attempts..."
        sleep $HEALTH_CHECK_INTERVAL
    done
    
    log_error "$service 启动超时"
    return 1
}

# Validation functions
validate_application() {
    log_info "验证应用部署..."
    
    # Check if containers are running
    log_info "检查容器状态..."
    
    local expected_containers=("aiplusall-kb-app" "aiplusall-kb-frontend" "aiplusall-kb-docreader")
    for container in "${expected_containers[@]}"; do
        if docker ps | grep -q "$container"; then
            log_success "容器 $container 正在运行"
        else
            log_warning "容器 $container 未找到或未运行"
        fi
    done
}

validate_api() {
    log_info "验证 API 可用性..."
    
    # Health check
    if check_service_health "API Health" "http://localhost:8080/health"; then
        # Parse response
        HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
        log_info "健康检查响应: $HEALTH_RESPONSE"
    else
        return 1
    fi
}

validate_database() {
    log_info "验证数据库连接..."
    
    # Test database connectivity through API
    RESPONSE=$(curl -s -X GET http://localhost:8080/health/db)
    
    if echo "$RESPONSE" | grep -q "ok\|true\|healthy"; then
        log_success "数据库连接正常"
    else
        log_error "数据库连接失败"
        return 1
    fi
}

validate_frontend() {
    log_info "验证前端部署..."
    
    if check_service_health "Frontend" "http://localhost/"; then
        # Check for main HTML file
        RESPONSE=$(curl -s http://localhost/index.html)
        if echo "$RESPONSE" | grep -q "aiplusall-kb\|<title>"; then
            log_success "前端加载成功"
        else
            log_warning "前端可能未正确部署"
        fi
    else
        return 1
    fi
}

validate_api_endpoints() {
    log_info "验证关键 API 端点..."
    
    # Test various API endpoints
    local endpoints=(
        "http://localhost:8080/api/v1/health"
        "http://localhost:8080/api/v1/user/profile"
        "http://localhost:8080/api/v1/knowledge"
    )
    
    for endpoint in "${endpoints[@]}"; do
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "$endpoint")
        if [ "$RESPONSE" != "000" ]; then
            log_success "端点 $endpoint 响应: HTTP $RESPONSE"
        else
            log_warning "端点 $endpoint 无响应"
        fi
    done
}

validate_vector_db() {
    log_info "验证向量数据库..."
    
    # Check Qdrant health
    if curl -s http://localhost:6334/health >/dev/null 2>&1; then
        log_success "Qdrant 连接正常"
    else
        log_warning "Qdrant 连接失败，请检查配置"
    fi
}

validate_redis() {
    log_info "验证 Redis 连接..."
    
    # Check Redis using docker
    if docker exec -it redis redis-cli ping 2>/dev/null | grep -q "PONG"; then
        log_success "Redis 连接正常"
    else
        log_warning "Redis 连接失败"
    fi
}

validate_storage() {
    log_info "验证存储配置..."
    
    # Check MinIO
    if curl -s http://localhost:9000/minio/health/live >/dev/null 2>&1; then
        log_success "MinIO 连接正常"
    else
        log_info "MinIO 未部署或未配置"
    fi
}

validate_logs() {
    log_info "检查应用日志..."
    
    # Check for errors in recent logs
    ERRORS=$(docker-compose logs app --tail=100 2>/dev/null | grep -i "error\|fatal" | wc -l)
    
    if [ "$ERRORS" -eq 0 ]; then
        log_success "日志中无错误"
    else
        log_warning "日志中发现 $ERRORS 条错误信息，请检查"
        docker-compose logs app --tail=20 | grep -i "error\|fatal"
    fi
}

validate_performance() {
    log_info "性能基线测试..."
    
    # Simple load test
    log_info "发送 10 个请求进行基线测试..."
    
    local total_time=0
    for i in {1..10}; do
        TIME=$(curl -s -w "%{time_total}" -o /dev/null http://localhost:8080/health)
        total_time=$(echo "$total_time + $TIME" | bc)
    done
    
    local avg_time=$(echo "scale=3; $total_time / 10" | bc)
    log_info "平均响应时间: ${avg_time}s"
    
    # Check if response time is reasonable
    if (( $(echo "$avg_time < 1" | bc -l) )); then
        log_success "性能基线通过"
    else
        log_warning "响应时间较长: ${avg_time}s"
    fi
}

validate_security() {
    log_info "验证安全配置..."
    
    # Check HTTPS (if configured)
    # Check for security headers
    RESPONSE=$(curl -s -I http://localhost/ | grep -i "security\|x-\|strict")
    
    if [ -n "$RESPONSE" ]; then
        log_success "检测到安全头配置"
    else
        log_warning "未检测到安全头，建议添加"
    fi
}

display_summary() {
    local elapsed=$(($(date +%s) - START_TIME))
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}部署验证完成${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BLUE}验证摘要:${NC}"
    echo "  环境: $ENVIRONMENT"
    echo "  耗时: ${elapsed}s"
    echo ""
    echo -e "${GREEN}✓ 部署验证已完成${NC}"
    echo ""
    echo -e "${BLUE}关键服务状态:${NC}"
    docker-compose ps
    echo ""
    echo -e "${BLUE}建议的后续步骤:${NC}"
    echo "  1. 访问应用: http://localhost"
    echo "  2. 查看 API 文档: http://localhost:8080/swagger/index.html"
    echo "  3. 查看 Jaeger 跟踪: http://localhost:16686"
    echo "  4. 监控日志: docker-compose logs -f app"
    echo ""
}

# Main execution
main() {
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}aiplusall-kb 生产部署验证${NC}"
    echo -e "${BLUE}环境: $ENVIRONMENT${NC}"
    echo -e "${BLUE}超时时间: ${TIMEOUT}s${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    
    # Wait for services to be ready
    wait_for_service "Application" "http://localhost:8080/health" || {
        log_error "应用启动失败"
        exit 1
    }
    
    # Run validation checks
    validate_application
    echo ""
    validate_api
    echo ""
    validate_database
    echo ""
    validate_frontend
    echo ""
    validate_api_endpoints
    echo ""
    validate_vector_db
    echo ""
    validate_redis
    echo ""
    validate_storage
    echo ""
    validate_logs
    echo ""
    validate_performance
    echo ""
    validate_security
    echo ""
    
    display_summary
}

main "$@"
