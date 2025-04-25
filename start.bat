@echo off
cd /d "%~dp0"

REM Проверка архитектуры системы
if "%PROCESSOR_ARCHITECTURE%" == "AMD64" (
    start .\main_64.exe
) else if "%PROCESSOR_ARCHITEW6432%" == "AMD64" (
    start .\main_64.exe
) else (
    start .\main_32.exe
)