@echo off
setlocal

set "EXE=%~dp0..\gacha.exe"
set "NAME=%~1"
if "%NAME%"=="" (
  echo Usage: restore.bat BACKUP_NAME(.json^.js)
  echo Example: restore.bat 2025-09-13_120000.json
  exit /b 2
)

if not exist "%EXE%" (
  echo gacha.exe not found: "%EXE%"
  exit /b 1
)

"%EXE%" restore %NAME%
if errorlevel 1 exit /b 1
echo Restored: %NAME%
exit /b 0
