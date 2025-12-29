@echo off
REM Claude Code Tresor Quick Install for Windows
REM Author: Alireza Rezvani (Windows adaptation)

setlocal enabledelayedexpansion

REM Configuration
set "CLAUDE_DIR=%USERPROFILE%\.claude"
set "TRESOR_SRC=%TEMP%\claude-code-tresor"
set "PROJECT_DIR=C:\Users\hrp\Downloads\ai-pro"

echo ========================================
echo Claude Code Tresor - Quick Install
echo ========================================
echo.

REM Check if tresor is already cloned
if exist "%TRESOR_SRC%" (
    echo [INFO] Tresor already cloned at %TRESOR_SRC%
) else (
    echo [INFO] Cloning tresor to %TRESOR_SRC%...
    git clone https://github.com/alirezarezvani/claude-code-tresor.git "%TRESOR_SRC%" --depth 1
    if errorlevel 1 (
        echo [ERROR] Failed to clone repository
        pause
        exit /b 1
    )
)

REM Create directories
echo [INFO] Creating directories...
if not exist "%CLAUDE_DIR%\agents" mkdir "%CLAUDE_DIR%\agents"
if not exist "%CLAUDE_DIR%\skills" mkdir "%CLAUDE_DIR%\skills"
if not exist "%CLAUDE_DIR%\commands" mkdir "%CLAUDE_DIR%\commands"
if not exist "%CLAUDE_DIR%\subagents" mkdir "%CLAUDE_DIR%\subagents"
if not exist "%CLAUDE_DIR%\tresor-resources" mkdir "%CLAUDE_DIR%\tresor-resources"

REM Install Skills
echo [INFO] Installing Skills...
xcopy /s /e /y /q "%TRESOR_SRC%\skills\*" "%CLAUDE_DIR%\skills\"

REM Install Core Agents
echo [INFO] Installing Core Agents...
for /d %%D in ("%TRESOR_SRC%\subagents\core\*") do (
    for %%F in ("%%~D\agent.md") do (
        if exist "%%F" (
            copy /y "%%F" "%CLAUDE_DIR%\agents\%%~nD.md"
        )
    )
)

REM Install Subagents
echo [INFO] Installing Subagents...
xcopy /s /e /y /q "%TRESOR_SRC%\subagents\*" "%CLAUDE_DIR%\subagents\"

REM Install Commands
echo [INFO] Installing Commands...
for /d %%C in ("%TRESOR_SRC%\commands\*") do (
    if not "%%~nC"=="README" (
        for /d %%D in ("%%~C\*") do (
            robocopy "%TRESOR_SRC%\commands\%%~nC\%%~nD" "%CLAUDE_DIR%\commands\%%~nC-%%~nD" /E /NFL /NDL /NJH /NJS >nul 2>&1
        )
    )
)

REM Install Resources
echo [INFO] Installing Resources...
xcopy /s /e /y /q "%TRESOR_SRC%\prompts\*" "%CLAUDE_DIR%\tresor-resources\prompts\" 2>nul
xcopy /s /e /y /q "%TRESOR_SRC%\standards\*" "%CLAUDE_DIR%\tresor-resources\standards\" 2>nul
xcopy /s /e /y /q "%TRESOR_SRC%\examples\*" "%CLAUDE_DIR%\tresor-resources\examples\" 2>nul
copy /y "%TRESOR_SRC%\README.md" "%CLAUDE_DIR%\tresor-resources\" 2>nul

echo.
echo ========================================
echo Installation Complete!
echo ========================================
echo.
echo Installed:
echo   - Skills: %CLAUDE_DIR%\skills\
echo   - Agents: %CLAUDE_DIR%\agents\
echo   - Commands: %CLAUDE_DIR%\commands\
echo   - Subagents: %CLAUDE_DIR%\subagents\
echo.
pause
