@echo off
chcp 65001 >nul
echo ========================================
echo 古代木结构建筑虫蛀监测系统
echo ========================================
echo.

echo [1/3] 检查Go环境...
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo 警告: 未检测到Go环境，请先安装Go 1.21+
    echo 下载地址: https://golang.org/dl/
    echo.
    pause
    exit /b 1
)
echo Go环境检查通过
echo.

echo [2/3] 启动后端服务...
cd /d "%~dp0backend"
start "后端服务" cmd /k "go mod tidy && go run cmd/server/main.go"

echo 等待后端服务启动...
timeout /t 5 /nobreak >nul

echo.
echo [3/3] 服务启动完成！
echo ========================================
echo 前端地址: http://localhost:8080/
echo API地址:  http://localhost:8080/api/v1/
echo ========================================
echo.
echo 提示: 要启动LoRa模拟器，请运行 start_lora_simulator.bat
echo.
pause
