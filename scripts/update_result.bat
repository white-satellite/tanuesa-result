@echo off
setlocal enabledelayedexpansion

REM Path to gacha.exe (adjust if needed)
set "EXE=%~dp0..\gacha.exe"

if "%~1"=="" goto :usage
if "%~2"=="" goto :usage

"%EXE%" %~1 %~2
goto :eof

:usage
echo Usage: update_result.bat "WINNER_NAME" HIT_FLAG(0|1)
exit /b 2
