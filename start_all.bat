@echo off
chcp 65001 >nul
echo ========================================
echo 古代木结构建筑虫蛀监测系统 - 一键启动
echo ========================================
echo.

echo 启动InfluxDB...
cd /d "%~dp0influxdb"
start "InfluxDB" docker-compose up -d

echo 等待InfluxDB启动...
timeout /t 10 /nobreak >nul

echo.
echo 启动后端服务...
cd /d "%~dp0backend"
start "后端服务" cmd /k "go mod tidy && go run cmd/server/main.go"

echo 等待后端服务启动...
timeout /t 8 /nobreak >nul

echo.
echo 启动LoRa模拟器...
cd /d "%~dp0lora-simulator"
start "LoRa模拟器" cmd /k "go run main.go"

echo.
echo ========================================
echo 所有服务已启动！
echo 前端地址: http://localhost:8080/
echo API地址:  http://localhost:8080/api/v1/
echo InfluxDB: http://localhost:8086/
echo ========================================
echo.
pause
