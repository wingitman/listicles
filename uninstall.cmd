@echo off
setlocal EnableExtensions DisableDelayedExpansion
set "dest=%LOCALAPPDATA%\Programs\listicles"
if not "%~1"=="" set "dest=%~f1"
if not "%~2"=="" goto usage
rem For an installation that replaced an existing executable elsewhere, pass
rem that directory explicitly. Never recursively delete a shared bin directory.
if exist "%dest%\listicles.exe" del /q "%dest%\listicles.exe" || goto failed
if exist "%dest%\l.cmd" del /q "%dest%\l.cmd" || goto failed
rem A custom directory may serve other tools. Keep its existing PATH entry.
if /i not "%dest%"=="%LOCALAPPDATA%\Programs\listicles" goto done
pushd "%~dp0" || goto failed
set "LISTICLES_CMD_PATH=%dest%"
call shell\cmd-path.cmd remove
if errorlevel 1 (
    popd
    goto failed
)
popd
rd "%dest%" >nul 2>&1
:done
echo Uninstalled. Config and themes in "%APPDATA%\delbysoft" were preserved.
echo Sign out and back in to refresh PATH in all terminals.
exit /b 0
:failed
echo ERROR: Uninstall failed. Close listicles and check file and registry permissions.
exit /b 1
:usage
echo Usage: uninstall.cmd [custom-install-directory]
exit /b 1
