@echo off
chcp 65001 >nul
echo ========================================
echo LoRa传感器模拟器
echo ========================================
echo.

cd /d "%~dp0lora-simulator"
go run main.go

pause
