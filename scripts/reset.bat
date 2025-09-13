@echo off
setlocal

set "EXE=%~dp0..\gacha.exe"
if not exist "%EXE%" (
  echo gacha.exe not found: "%EXE%"
  exit /b 1
)

"%EXE%" reset
if errorlevel 1 exit /b 1
echo Reset completed
exit /b 0
