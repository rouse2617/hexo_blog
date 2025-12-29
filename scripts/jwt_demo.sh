#!/bin/bash

# JWT 认证演示脚本
# 使用方法: ./scripts/jwt_demo.sh

BASE_URL="http://localhost:8080"

echo "==================================="
echo "JWT 认证演示"
echo "==================================="
echo ""

# 1. 登录获取 Token
echo "1. 用户登录..."
echo "POST /api/auth/login"
echo ""

RESPONSE=$(curl -s -X POST "${BASE_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }')

echo "响应: ${RESPONSE}"
echo ""

# 提取 Token
TOKEN=$(echo ${RESPONSE} | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败，请检查服务是否运行并启用了认证"
  echo ""
  echo "提示："
  echo "- 确保 config.yaml 中 auth.enabled 为 true"
  echo "- 使用 'go run cmd/server/main.go' 启动服务"
  exit 1
fi

echo "✅ 登录成功，获取到 Token"
echo ""
echo "Token: ${TOKEN:0:50}..."
echo ""

# 2. 验证 Token
echo "2. 验证 Token..."
echo "GET /api/auth/validate"
echo ""

curl -s -X GET "${BASE_URL}/api/auth/validate" \
  -H "Authorization: Bearer ${TOKEN}" \
  | jq '.'
echo ""

# 3. 访问受保护接口
echo "3. 访问受保护接口（主机列表）..."
echo "GET /api/hosts"
echo ""

curl -s -X GET "${BASE_URL}/api/hosts" \
  -H "Authorization: Bearer ${TOKEN}" \
  | jq '.'
echo ""

# 4. 测试无 Token 访问
echo "4. 测试无 Token 访问（应该返回 401）..."
echo "GET /api/hosts (无 Authorization header)"
echo ""

curl -s -X GET "${BASE_URL}/api/hosts" \
  | jq '.'
echo ""

# 5. 刷新 Token
echo "5. 刷新 Token..."
echo "POST /api/auth/refresh"
echo ""

REFRESH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{
    \"token\": \"${TOKEN}\"
  }")

echo "响应: ${REFRESH_RESPONSE}"
echo ""

# 6. 登出
echo "6. 登出..."
echo "POST /api/auth/logout"
echo ""

curl -s -X POST "${BASE_URL}/api/auth/logout" \
  -H "Authorization: Bearer ${TOKEN}" \
  | jq '.'
echo ""

echo "==================================="
echo "演示完成"
echo "==================================="
