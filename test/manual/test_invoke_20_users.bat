@echo off
setlocal enableextensions

REM Prefer absolute PowerShell path, fallback to PATH lookup
set "PS=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"
if not exist "%PS%" set "PS=powershell"

echo [INFO] Running 20-user invoke test...
"%PS%" -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0test_invoke_20_users.ps1" -Reset -Prefix "USER" -Count 20
if errorlevel 1 (
  echo [ERROR] TEST FAILED. Review the error messages above.
  pause
  exit /b 1
)
echo [OK] TEST PASSED.
pause
exit /b 0
