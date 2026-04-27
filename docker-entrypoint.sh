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