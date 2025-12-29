@echo off
setlocal enabledelayedexpansion

REM JWT 认证演示脚本 (Windows)
REM 使用方法: scripts\jwt_demo.bat

set BASE_URL=http://localhost:8080

echo ===================================
echo JWT 认证演示
echo ===================================
echo.

REM 1. 登录获取 Token
echo 1. 用户登录...
echo POST /api/auth/login
echo.

curl -s -X POST "%BASE_URL%/api/auth/login" ^
  -H "Content-Type: application/json" ^
  -d "{\"username\": \"admin\", \"password\": \"admin123\"}" > login_response.json

type login_response.json
echo.

REM 检查登录是否成功
findstr /C:"\"token\"" login_response.json >nul
if errorlevel 1 (
    echo ❌ 登录失败，请检查服务是否运行并启用了认证
    echo.
    echo 提示：
    echo - 确保 config.yaml 中 auth.enabled 为 true
    echo - 使用 'go run cmd/server/main.go' 启动服务
    del login_response.json
    pause
    exit /b 1
)

echo ✅ 登录成功
echo.

REM 提取 Token (使用 PowerShell)
for /f "tokens=*" %%a in ('powershell -Command "(Get-Content login_response.json | ConvertFrom-Json).data.token"') do set TOKEN=%%a

echo Token: %TOKEN:~0,50%...
echo.

REM 2. 验证 Token
echo 2. 验证 Token...
echo GET /api/auth/validate
echo.

curl -s -X GET "%BASE_URL%/api/auth/validate" ^
  -H "Authorization: Bearer %TOKEN%"
echo.
echo.

REM 3. 访问受保护接口
echo 3. 访问受保护接口（主机列表）...
echo GET /api/hosts
echo.

curl -s -X GET "%BASE_URL%/api/hosts" ^
  -H "Authorization: Bearer %TOKEN%"
echo.
echo.

REM 4. 测试无 Token 访问
echo 4. 测试无 Token 访问（应该返回 401）...
echo GET /api/hosts (无 Authorization header)
echo.

curl -s -X GET "%BASE_URL%/api/hosts"
echo.
echo.

REM 5. 登出
echo 5. 登出...
echo POST /api/auth/logout
echo.

curl -s -X POST "%BASE_URL%/api/auth/logout" ^
  -H "Authorization: Bearer %TOKEN%"
echo.
echo.

REM 清理
del login_response.json

echo ===================================
echo 演示完成
echo ===================================
pause
