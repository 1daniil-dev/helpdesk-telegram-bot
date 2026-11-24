@echo off
cd /d "%~dp0"
echo Building helpdesk-bot.exe...
go build -o helpdesk-bot.exe ./cmd/bot
if errorlevel 1 (
    echo.
    echo Build failed.
    pause
    exit /b 1
)
echo Build OK: helpdesk-bot.exe
pause
