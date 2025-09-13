@echo off
setlocal

set "EXE=%~dp0..\gacha.exe"
if not exist "%EXE%" (
  echo gacha.exe not found: "%EXE%"
  pause
  exit /b 1
)

echo [INFO] Backup and reset (new session)...
"%EXE%" reset
if errorlevel 1 (
  echo [ERROR] reset failed
  pause
  exit /b 1
)

echo [INFO] Regenerating backup index...
"%EXE%" gen-backup-index
if errorlevel 1 (
  echo [WARN] gen-backup-index failed (you can run it later)
)

echo [OK] New session created.
pause
exit /b 0

