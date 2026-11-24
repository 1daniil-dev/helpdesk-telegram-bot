@echo off
cd /d "%~dp0"
if not exist ".env" (
    echo File .env not found.
    echo Copy .env.example to .env and fill BOT_TOKEN and OWNER_IDS.
    pause
    exit /b 1
)
go build -o helpdesk-bot.exe ./cmd/bot
if errorlevel 1 (
    echo.
    echo Build failed.
    pause
    exit /b 1
)
helpdesk-bot.exe
pause
