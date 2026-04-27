#!/bin/bash
# scripts/healthcheck.sh

# API 健康检查
curl -f http://localhost:8081/health || exit 1

# MCP 健康检查
curl -f http://localhost:8080/health || exit 1

# Worker 健康检查
curl -f http://localhost:8082/health || exit 1