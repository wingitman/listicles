@echo off
setlocal EnableExtensions DisableDelayedExpansion
rem A private directory avoids stale cd files and concurrent invocation collisions.
:temp
set "scratch=%TEMP%\listicles-%RANDOM%-%RANDOM%"
if exist "%scratch%" goto temp
mkdir "%scratch%" 2>nul || exit /b 1
"%~dp0listicles.exe" --cd-file "%scratch%\cd" %*
set "result=%errorlevel%"
set "target="
rem The Go cd-file is UTF-8. Restore the caller's console code page afterward.
for /f "tokens=2 delims=:" %%C in ('chcp') do set "codepage=%%C"
chcp 65001 >nul
if exist "%scratch%\cd" set /p "target=" < "%scratch%\cd"
if defined codepage chcp %codepage% >nul
del /q "%scratch%\cd" >nul 2>&1
rd "%scratch%" >nul 2>&1
if not "%result%"=="0" exit /b %result%
if not defined target exit /b 0
rem Leave the local scope before CD so the caller retains the new directory.
endlocal & cd /d "%target%"
exit /b %errorlevel%
