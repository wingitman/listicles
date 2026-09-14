@echo off
rem Native CMD user PATH editing. Never copy the merged process PATH to HKCU,
rem never use SETX (truncation/expansion), and preserve the registry value type.
setlocal EnableExtensions DisableDelayedExpansion
set "action=%~1"
set "directory=%LISTICLES_CMD_PATH%"
if not defined directory exit /b 1
set "oldPath="
set "kind=REG_EXPAND_SZ"
reg query HKCU\Environment >nul 2>&1 || exit /b 1
for /f "tokens=2 delims=:" %%C in ('chcp') do set "codepage=%%C"
chcp 65001 >nul
rem Bound the raw output BEFORE loading it into a CMD environment variable.
:temp
set "scratch=%TEMP%\listicles-path-%RANDOM%-%RANDOM%"
if exist "%scratch%" goto temp
mkdir "%scratch%" 2>nul || goto failed
reg query HKCU\Environment /v Path > "%scratch%\path" 2>nul
for %%F in ("%scratch%\path") do if %%~zF GTR 6000 goto raw_too_long
for /f "usebackq tokens=1,2,*" %%A in ("%scratch%\path") do if /i "%%A"=="Path" (
    set "kind=%%B"
    set "oldPath=%%C"
)
del /q "%scratch%\path"
rd "%scratch%"
setlocal EnableDelayedExpansion
if not "!kind!"=="REG_SZ" if not "!kind!"=="REG_EXPAND_SZ" goto failed
rem CMD has an 8191-character command limit. Leave generous room for arguments;
rem refuse a large PATH rather than attempting a possibly destructive rewrite.
if not "!oldPath:~6000!"=="" goto too_long
set "remaining=!oldPath!"
set "newPath="
set "entry="
set "found="
set "separator="
rem Scan characters so unrelated empty entries and literal %%variables%% survive.
:scan
if not defined remaining goto last
set "char=!remaining:~0,1!"
set "remaining=!remaining:~1!"
if "!char!"==";" goto boundary
set "entry=!entry!!char!"
goto scan
:boundary
call :entry
set "entry="
goto scan
:last
call :entry
if /i "!action!"=="add" if not defined found (
    if defined newPath set "newPath=!newPath!;"
    set "newPath=!newPath!!directory!"
)
if "!newPath!"=="!oldPath!" goto success
if not "!newPath:~6000!"=="" goto too_long
rem Escape embedded quotes for reg.exe's Windows argument parser.
reg add HKCU\Environment /v Path /t !kind! /d "!newPath:"=\"!" /f >nul || goto failed
goto success
:success
if defined codepage chcp %codepage% >nul
exit /b 0
:entry
set "compare=!entry:"=!"
if /i "!compare!"=="!directory!" (
    set "found=1"
    if /i "!action!"=="remove" exit /b 0
)
set "newPath=!newPath!!separator!!entry!"
set "separator=;"
exit /b 0
:too_long
echo ERROR: User PATH is too long to edit safely with CMD. Edit it in Windows Environment Variables instead.
goto error
:raw_too_long
del /q "%scratch%\path"
rd "%scratch%"
goto too_long
:failed
echo ERROR: Could not update the user PATH. Check registry permissions.
:error
if defined codepage chcp %codepage% >nul
exit /b 1
