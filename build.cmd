@echo off
setlocal enabledelayedexpansion

REM ============================================================
REM  Cross-build script for netspeed (linux + windows CLI/GUI)
REM  Usage: build.cmd
REM  Output: dist\netspeed-{os}-{arch}[.exe]
REM          dist\netspeed-windows-amd64-gui.exe
REM ============================================================

set OUTPUT_DIR=dist
if not exist %OUTPUT_DIR% mkdir %OUTPUT_DIR%

echo ============================================================
echo  netspeed cross-build
echo ============================================================
echo.

REM ---- Linux amd64 ----
echo [1/4] Building linux/amd64 ...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o %OUTPUT_DIR%\netspeed-linux-amd64 .
if errorlevel 1 (
    echo [FAIL] linux/amd64
    exit /b 1
)
echo [OK]   linux/amd64  -^> %OUTPUT_DIR%\netspeed-linux-amd64
echo.

REM ---- Linux loong64 ----
echo [2/4] Building linux/loong64 ...
set GOOS=linux
set GOARCH=loong64
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o %OUTPUT_DIR%\netspeed-linux-loong64 .
if errorlevel 1 (
    echo [FAIL] linux/loong64
    exit /b 1
)
echo [OK]   linux/loong64 -^> %OUTPUT_DIR%\netspeed-linux-loong64
echo.

REM ---- Windows amd64 (CLI) ----
echo [3/4] Building windows/amd64 (CLI) ...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o %OUTPUT_DIR%\netspeed-windows-amd64.exe .
if errorlevel 1 (
    echo [FAIL] windows/amd64 CLI
    exit /b 1
)
echo [OK]   windows/amd64 CLI  -^> %OUTPUT_DIR%\netspeed-windows-amd64.exe
echo.

REM ---- Windows amd64 (GUI) ----
echo [4/4] Building windows/amd64 (GUI) ...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

REM Generate manifest resource (.syso) for walk common controls
echo       Generating manifest resource ...
go run github.com/akavel/rsrc@latest -manifest gui\manifest.xml -o gui\rsrc.syso
if errorlevel 1 (
    echo [FAIL] rsrc manifest generation
    exit /b 1
)

go build -trimpath -ldflags "-s -w -H windowsgui" -o %OUTPUT_DIR%\netspeed-windows-amd64-gui.exe ./gui
if errorlevel 1 (
    echo [FAIL] windows/amd64 GUI
    del gui\rsrc.syso 2>nul
    exit /b 1
)
del gui\rsrc.syso 2>nul
echo [OK]   windows/amd64 GUI  -^> %OUTPUT_DIR%\netspeed-windows-amd64-gui.exe
echo.

REM ---- Optional: UPX compression (skip loong64) ----
where upx >nul 2>nul
if errorlevel 1 (
    echo [SKIP] UPX not found in PATH, skipping compression.
    goto :show_result
)

echo ------------------------------------------------------------
echo  Compressing with UPX ...
echo ------------------------------------------------------------
upx --best --lzma %OUTPUT_DIR%\netspeed-linux-amd64
upx --best --lzma %OUTPUT_DIR%\netspeed-windows-amd64.exe
upx --best --lzma %OUTPUT_DIR%\netspeed-windows-amd64-gui.exe
echo [SKIP] UPX for loong64 (not supported)
echo.

:show_result
echo ============================================================
echo  Build complete. Output files in %OUTPUT_DIR%\:
echo ============================================================
for %%f in (%OUTPUT_DIR%\*) do (
    echo   %%~zf bytes  %%~nxf
)
echo.

endlocal
