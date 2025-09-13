@echo off
setlocal

set PS=powershell -NoProfile -ExecutionPolicy Bypass -File
set SCRIPT=%~dp0stop_api.ps1

%PS% "%SCRIPT%" %*
if errorlevel 1 (
  echo [INFO] API server may not be running.
  pause
  exit /b 1
)
pause
exit /b 0

