# Docker 部署计划

## 文档信息

| 字段 | 内容 |
|------|------|
| **项目编号** | TMOS |
| **计划名称** | Docker 部署计划 |
| **对应架构** | docs/v1/02-architecture/ |
| **优先级** | P0 |
| **预估工时** | 2 天 |

---

## 1. 部署架构

### 1.1 整体架构

```
                    ┌─────────────────┐
                    │   Nginx (SSL)   │
                    │   端口: 80, 443 │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
    ┌─────────────────┐ ┌──────────┐ ┌──────────────┐
    │   API Server    │ │ MCP Svr  │ │   Worker     │
    │   端口: 8081    │ │ 端口:8080│ │ 端口: 8082   │
    └────────┬────────┘ └────┬─────┘ └──────┬───────┘
             │               │              │
    ┌────────┴────────┐ ┌────┴─────┐ ┌──────┴───────┐
    │    PostgreSQL    │ │  Redis   │ │ CloakBrowser │
    │    端口: 5432    │ │ 端口:6379│ │   (二进制)   │
    └─────────────────┘ └──────────┘ └──────────────┘
```

### 1.2 服务组件

| 服务 | 镜像 | 端口 | 说明 |
|------|------|------|------|
| Nginx | nginx:alpine | 80, 443 | 反向代理 + SSL |
| API Server | tmos/api | 8081 | REST API |
| MCP Server | tmos/mcp | 8080 | MCP 服务端 |
| Worker | tmos/worker | 8082 | 后台任务处理 |
| PostgreSQL | postgres:15 | 5432 | 主数据库 |
| Redis | redis:7 | 6379 | 缓存 + 队列 |

---

## 2. 开发任务拆分

| 任务编号 | 任务名称 | 涉及文件 | 预估工时 | 依赖 |
|---------|---------|---------|---------|------|
| T-01 | Dockerfile 编写 | Dockerfile, Dockerfile.worker | 0.25 天 | - |
| T-02 | Docker Compose 配置 | docker-compose.yml, .env.example | 0.25 天 | - |
| T-03 | Nginx 配置 | nginx/ | 0.25 天 | T-01, T-02 |
| T-04 | 部署脚本 | deploy.sh, docker-entrypoint.sh | 0.25 天 | T-01~03 |
| T-05 | 健康检查配置 | healthcheck/ | 0.25 天 | T-01~04 |
| T-06 | 监控配置 | prometheus.yml, grafana/ | 0.25 天 | T-01~05 |
| T-07 | 部署验证 | - | 0.5 天 | T-01~06 |

---

## 3. 详细任务定义

### T-01: Dockerfile 编写

**任务概述**: 编写各服务的 Dockerfile

**输出**:
- `Dockerfile` (API Server)
- `Dockerfile.worker` (Worker)
- `Dockerfile.mcp` (MCP Server)

**实现要求**:

```dockerfile
# Dockerfile (API Server)
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /api /usr/local/bin/
COPY --from=builder /app/migrations /migrations

EXPOSE 8081
ENTRYPOINT ["api"]
CMD ["--config", "/config/config.yaml"]
```

```dockerfile
# Dockerfile.mcp (MCP Server)
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /mcp ./cmd/mcp

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /mcp /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["/mcp"]
```

**验收标准**:
- [ ] **构建成功**: 各服务镜像构建成功
- [ ] **镜像大小**: 镜像大小 < 200MB
- [ ] **多阶段构建**: 使用多阶段构建减小镜像

---

### T-02: Docker Compose 配置

**任务概述**: 配置 Docker Compose 编排

**输出**:
- `docker-compose.yml`
- `.env.example`

**实现要求**:

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8081:8081"
    env_file:
      - .env
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ./config:/config:ro
    restart: unless-stopped

  mcp:
    build:
      context: .
      dockerfile: Dockerfile.mcp
    ports:
      - "8080:8080"
    env_file:
      - .env
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    env_file:
      - .env
    depends_on:
      - postgres
      - redis
    volumes:
      - ./cloakbrowser:/app/cloakbrowser
      - instance_data:/data/instances
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    depends_on:
      - api
      - mcp
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  instance_data:
```

```bash
# .env.example
POSTGRES_DB=tmos
POSTGRES_USER=tmos
POSTGRES_PASSWORD=change_me_in_production

REDIS_PASSWORD=change_me_in_production

JWT_SECRET=change_me_in_production

API_PORT=8081
MCP_PORT=8080
WORKER_PORT=8082

CLOAKBROWSER_PATH=/app/cloakbrowser
```

**验收标准**:
- [ ] **一键启动**: `docker-compose up -d` 成功
- [ ] **依赖健康**: 服务按依赖顺序启动
- [ ] **数据持久化**: 数据卷正确配置

---

### T-03: Nginx 配置

**任务概述**: 配置 Nginx 反向代理

**输出**:
- `nginx/nginx.conf`
- `nginx/ssl/` (SSL 证书目录)

**实现要求**:

```nginx
# nginx.conf
events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # 日志格式
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;
    error_log /var/log/nginx/error.log warn;

    # Gzip 压缩
    gzip on;
    gzip_types text/plain application/json application/javascript;

    # 上传大小限制
    client_max_body_size 500M;

    # API Server
    upstream api_backend {
        server api:8081;
    }

    # MCP Server
    upstream mcp_backend {
        server mcp:8080;
    }

    server {
        listen 80;
        server_name _;

        # 重定向到 HTTPS
        return 301 https://$host$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name _;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;

        # API 路由
        location /api/ {
            proxy_pass http://api_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_read_timeout 300s;
        }

        # MCP 路由
        location /mcp/ {
            proxy_pass http://mcp_backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_read_timeout 86400s;
        }

        # WebSocket 支持
        location /ws/ {
            proxy_pass http://api_backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

**验收标准**:
- [ ] **反向代理**: 请求正确路由到后端
- [ ] **SSL**: HTTPS 配置正确
- [ ] **WebSocket**: MCP WebSocket 支持

---

### T-04: 部署脚本

**任务概述**: 编写部署和运维脚本

**输出**:
- `deploy.sh`
- `docker-entrypoint.sh`
- `scripts/`

**实现要求**:

```bash
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
```

```bash
#!/bin/bash
# docker-entrypoint.sh (API Server)

set -e

echo "Starting TMOS API Server..."

# 等待数据库就绪
until pg_isready -h $POSTGRES_HOST -U $POSTGRES_USER; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done

# 运行迁移
echo "Running database migrations..."
ls -1 /migrations/*.sql | sort | xargs -I{} psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -f {}

# 启动服务
echo "Starting API Server..."
exec "$@"
```

**验收标准**:
- [ ] **部署脚本**: 一键部署成功
- [ ] **回滚脚本**: 支持版本回滚
- [ ] **日志查看**: 日志脚本可用

---

### T-05: 健康检查配置

**任务概述**: 配置服务健康检查

**输出**:
- `docker-compose.yml` (补充 healthcheck)
- `scripts/healthcheck.sh`

**实现要求**:

```bash
#!/bin/bash
# scripts/healthcheck.sh

# API 健康检查
curl -f http://localhost:8081/health || exit 1

# MCP 健康检查
curl -f http://localhost:8080/health || exit 1

# Worker 健康检查
curl -f http://localhost:8082/health || exit 1
```

---

### T-06: 监控配置

**任务概述**: 配置 Prometheus + Grafana 监控

**输出**:
- `monitoring/prometheus.yml`
- `monitoring/grafana/`

**实现要求**:

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'tmos-api'
    static_configs:
      - targets: ['api:8081']
    metrics_path: /metrics

  - job_name: 'tmos-mcp'
    static_configs:
      - targets: ['mcp:8080']
    metrics_path: /metrics

  - job_name: 'tmos-worker'
    static_configs:
      - targets: ['worker:8082']
    metrics_path: /metrics

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:5432']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis:6379']
```

**验收标准**:
- [ ] **指标暴露**: 各服务暴露 Prometheus 指标
- [ ] **Grafana 看板**: 默认看板可用

---

### T-07: 部署验证

**任务概述**: 验证部署正确性

**验证步骤**:
1. 启动所有服务
2. 检查服务健康状态
3. 执行 API 调用测试
4. 检查日志输出

**验收标准**:
- [ ] **服务健康**: 所有服务状态为 healthy
- [ ] **API 可用**: REST API 响应正常
- [ ] **MCP 可用**: MCP 工具调用正常
- [ ] **日志正常**: 无错误日志

---

## 4. 部署检查清单

### 4.1 部署前检查

- [ ] `.env` 文件已创建
- [ ] SSL 证书已配置
- [ ] 数据库迁移脚本就绪
- [ ] CloakBrowser 二进制已准备

### 4.2 部署后检查

- [ ] 所有容器运行中
- [ ] 服务健康检查通过
- [ ] API 端点可访问
- [ ] 数据库连接正常
- [ ] Redis 连接正常

### 4.3 监控检查

- [ ] Prometheus 采集正常
- [ ] Grafana 看板正常
- [ ] 日志收集正常

---

## 5. 运维操作

### 5.1 常用命令

```bash
# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f api
docker-compose logs -f mcp

# 重启服务
docker-compose restart api

# 停止服务
docker-compose down

# 重新构建
docker-compose build --force-recreate

# 执行数据库迁移
docker-compose exec api migrate
```

### 5.2 备份策略

```bash
# 数据库备份
docker-compose exec postgres pg_dump -U $POSTGRES_USER $POSTGRES_DB > backup_$(date +%Y%m%d).sql

# 数据卷备份
docker run --rm -v tmos_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_data.tar.gz /data
```
