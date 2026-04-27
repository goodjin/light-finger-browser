#!/bin/bash
# deploy.sh

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1"
}

# 检查环境
check_env() {
    if [ ! -f .env ]; then
        error ".env file not found"
        exit 1
    fi
}

# 拉取最新代码
pull_code() {
    log "Pulling latest code..."
    git pull origin main
}

# 构建镜像
build_images() {
    log "Building Docker images..."
    docker-compose build --parallel
}

# 运行数据库迁移
run_migrations() {
    log "Running database migrations..."
    docker-compose exec -T postgres psql -U $POSTGRES_USER -d $POSTGRES_DB < migrations/*.sql
}

# 启动服务
start_services() {
    log "Starting services..."
    docker-compose up -d
}

# 等待服务健康
wait_healthy() {
    log "Waiting for services to be healthy..."
    docker-compose ps --format json | jq -r 'select(.Service == "api") | .State'
    # 更复杂的健康检查逻辑...
}

# 主流程
main() {
    check_env
    pull_code
    build_images
    run_migrations
    start_services
    wait_healthy
    log "Deployment completed successfully!"
}

main "$@"