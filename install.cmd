@echo off
setlocal EnableExtensions DisableDelayedExpansion
set "reset="
if /i "%~1"=="--default" set "reset=--default"
if /i "%~1"=="-Default" set "reset=--default"
if not "%~1"=="" if not defined reset goto usage
if not "%~2"=="" goto usage
if not defined LOCALAPPDATA goto failed
pushd "%~dp0" || goto failed
set "dest=%LOCALAPPDATA%\Programs\listicles"
rem Replace the existing PATH-resolved executable when one is installed.
set "existing="
for /f "delims=" %%F in ('where listicles.exe 2^>nul') do if not defined existing set "existing=%%F"
if defined existing for %%F in ("%existing%") do set "dest=%%~dpF"
for %%F in ("%dest%\.") do set "dest=%%~fF"
where go >nul 2>&1
if errorlevel 1 goto release
echo Building listicles from source...
set "commit=dev"
for /f "delims=" %%C in ('git rev-parse HEAD 2^>nul') do set "commit=%%C"
if not exist bin mkdir bin || goto failed_pop
rem Do not inherit cross-compilation settings from a previous build.
set "GOOS=windows"
set "GOARCH="
set "CGO_ENABLED=0"
go build -ldflags="-s -w -X github.com/wingitman/listicles/internal/version.Commit=%commit%" -o bin\listicles.exe .
if errorlevel 1 goto failed_pop
set "source=%CD%\bin\listicles.exe"
goto install
:release
rem The bundled release is Windows amd64, not an arbitrary architecture.
if /i "%PROCESSOR_ARCHITECTURE%"=="AMD64" goto release_ok
if /i "%PROCESSOR_ARCHITEW6432%"=="AMD64" goto release_ok
echo ERROR: The bundled release requires Windows x64. Install Go to build for this machine.
goto failed_pop
:release_ok
set "source=%CD%\releases\windows\listicles.exe"
if exist "%source%" goto install
echo ERROR: Missing releases\windows\listicles.exe. Extract the complete repository ZIP or install Go.
goto failed_pop
:install
echo Installing to "%dest%"...
if not exist "%dest%" mkdir "%dest%" || goto failed_pop
rem Stage and verify before replacing. A running executable causes a clear failure.
copy /y "%source%" "%dest%\listicles-new.exe" >nul || goto failed_pop
fc.exe /b "%source%" "%dest%\listicles-new.exe" >nul || goto failed_pop
"%dest%\listicles-new.exe" --ensure-config %reset%
if errorlevel 1 goto failed_pop
move /y "%dest%\listicles-new.exe" "%dest%\listicles.exe" >nul || goto failed_pop
fc.exe /b "%source%" "%dest%\listicles.exe" >nul || goto failed_pop
copy /y "shell\l.cmd" "%dest%\l.cmd" >nul || goto failed_pop
set "LISTICLES_CMD_PATH=%dest%"
call shell\cmd-path.cmd add
if errorlevel 1 goto failed_pop
echo Installed to "%dest%".
echo Config: "%APPDATA%\delbysoft\listicles.toml"
echo Sign out and back in for all terminals to inherit the updated user PATH.
echo In this prompt you can use:
echo   set "PATH=%dest%;%%PATH%%"
echo   l
rem Check using the current search order, not an artificially prepended PATH.
set "resolved="
for /f "delims=" %%F in ('where l 2^>nul') do if not defined resolved set "resolved=%%F"
if defined resolved if /i not "%resolved%"=="%dest%\l.cmd" echo WARNING: l currently resolves to "%resolved%". Remove the conflicting command or adjust PATH.
popd
exit /b 0
:failed_pop
if defined dest if exist "%dest%\listicles-new.exe" del /q "%dest%\listicles-new.exe" >nul 2>&1
popd
:failed
echo ERROR: Installation failed. Check the error above and close any running listicles instance before retrying.
exit /b 1
:usage
echo Usage: install.cmd [--default]
exit /b 1
