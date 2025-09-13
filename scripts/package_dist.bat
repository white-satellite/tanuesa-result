@echo off
setlocal

set PS=PowerShell -NoProfile -ExecutionPolicy Bypass -File
set SCRIPT=%~dp0package_dist.ps1

%PS% "%SCRIPT%" %*
if errorlevel 1 (
  echo Package creation failed.
  pause
  exit /b 1
)
exit /b 0

