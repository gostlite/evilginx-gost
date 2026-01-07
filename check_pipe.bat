@echo off
echo Batch started
timeout /t 2 >nul
echo Reading from stdin:
findstr "^"
echo Batch finished
