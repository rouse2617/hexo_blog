# Claude Code Tresor Windows Installation Script
# Author: Alireza Rezvani (Windows adaptation)
# License: MIT
# Created: December 2025

param(
    [switch]$SkillsOnly,
    [switch]$CommandsOnly,
    [switch]$AgentsOnly,
    [switch]$OrchestrationOnly,
    [switch]$ResourcesOnly,
    [switch]$Update,
    [switch]$NoBackup,
    [string]$BackupDir
)

# Configuration
$ClaudeCodeDir = "$env:USERPROFILE\.claude"
$RepoUrl = "https://github.com/alirezarezvani/claude-code-tresor"
$TresorDir = "$ClaudeCodeDir\tresor"
$Timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$BackupDirDefault = "$ClaudeCodeDir\backup-$Timestamp"

if ($BackupDir) {
    $BackupDirDefault = $BackupDir
}

# Functions
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Log-Info {
    param([string]$Message)
    Write-ColorOutput "[INFO] $Message" Green
}

function Log-Warn {
    param([string]$Message)
    Write-ColorOutput "[WARN] $Message" Yellow
}

function Log-Error {
    param(
        [string]$Message,
        [switch]$Exit
    )
    Write-ColorOutput "[ERROR] $Message" Red
    if ($Exit) { exit 1 }
}

function Write-Header {
    param([string]$Title)
    Write-Host ""
    Write-ColorOutput "=== $Title ===" Cyan
    Write-Host ""
}

function Test-GitInstalled {
    Write-Header "Checking Dependencies"

    try {
        $null = git --version
        Log-Info "Git is available"
    } catch {
        Log-Error "Git is not installed. Please install git first." -Exit
    }

    if (Get-Command "claude-code" -ErrorAction SilentlyContinue) {
        Log-Info "Claude Code CLI detected"
    } else {
        Log-Warn "Claude Code CLI not found. Make sure to install it from: https://claude.ai/code"
    }
}

function Initialize-Directories {
    Write-Header "Creating Directories"

    if (!(Test-Path $ClaudeCodeDir)) {
        Log-Info "Creating Claude Code directory: $ClaudeCodeDir"
        New-Item -ItemType Directory -Path $ClaudeCodeDir -Force | Out-Null
    } else {
        Log-Info "Claude Code directory already exists"
    }

    $subdirs = @("commands", "agents", "skills", "templates", "subagents")
    foreach ($subdir in $subdirs) {
        $path = Join-Path $ClaudeCodeDir $subdir
        if (!(Test-Path $path)) {
            New-Item -ItemType Directory -Path $path -Force | Out-Null
        }
    }

    Log-Info "Directory structure ready"
}

function Backup-ExistingConfig {
    Write-Header "Backing Up Existing Configuration"

    if ((Test-Path $TresorDir) -and !$NoBackup) {
        Log-Info "Creating backup at: $BackupDirDefault"
        Copy-Item -Path $ClaudeCodeDir -Destination $BackupDirDefault -Recurse -Force
        Log-Info "Backup created successfully"
    } else {
        Log-Info "No existing tresor installation found or backup skipped"
    }
}

function Clone-Repository {
    Write-Header "Downloading Claude Code Tresor"

    if (Test-Path $TresorDir) {
        Log-Info "Updating existing installation"
        Set-Location $TresorDir
        git pull origin main
    } else {
        Log-Info "Cloning repository to: $TresorDir"
        git clone $RepoUrl $TresorDir
    }

    Set-Location (Get-Location).Path
    Log-Info "Repository downloaded successfully"
}

function Install-Commands {
    Write-Header "Installing Slash Commands"

    $commandsSrc = Join-Path $TresorDir "commands"
    $commandsDest = Join-Path $ClaudeCodeDir "commands"

    if (Test-Path $commandsSrc) {
        Log-Info "Installing commands to: $commandsDest"

        # Get all command directories (2 levels deep)
        $categories = Get-ChildItem -Path $commandsSrc -Directory
        foreach ($category in $categories) {
            if ($category.Name -eq "README.md") { continue }

            $cmdDirs = Get-ChildItem -Path $category.FullName -Directory
            foreach ($cmdDir in $cmdDirs) {
                $cmdName = $cmdDir.Name
                $destDir = Join-Path $commandsDest "$($category.Name)-$cmdName"
                Log-Info "Installing command: $($category.Name)/$cmdName"
                Copy-Item -Path $cmdDir.FullName -Destination $destDir -Recurse -Force
            }
        }

        Log-Info "Commands installed successfully"
    } else {
        Log-Warn "Commands directory not found in repository"
    }
}

function Install-OrchestrationCommands {
    Write-Header "Installing Orchestration Commands (v2.7+)"

    $commandsSrc = Join-Path $TresorDir "commands"
    $commandsDest = Join-Path $ClaudeCodeDir "commands"

    if (Test-Path $commandsSrc) {
        Log-Info "Installing orchestration commands to: $commandsDest"

        $orchestrationCategories = @("security", "performance", "operations", "quality")

        foreach ($category in $orchestrationCategories) {
            $categoryPath = Join-Path $commandsSrc $category
            if (Test-Path $categoryPath) {
                $cmdDirs = Get-ChildItem -Path $categoryPath -Directory
                foreach ($cmdDir in $cmdDirs) {
                    $cmdName = $cmdDir.Name
                    $destDir = Join-Path $commandsDest "$category-$cmdName"
                    Log-Info "Installing orchestration command: $category/$cmdName"
                    Copy-Item -Path $cmdDir.FullName -Destination $destDir -Recurse -Force
                }
            }
        }

        Log-Info "Orchestration commands installed successfully"
    } else {
        Log-Warn "Commands directory not found in repository"
    }
}

function Install-Agents {
    Write-Header "Installing Core Agents"

    $subagentsSrc = Join-Path $TresorDir "subagents\core"
    $agentsDest = Join-Path $ClaudeCodeDir "agents"

    if (Test-Path $subagentsSrc) {
        Log-Info "Installing core agents to: $agentsDest"

        $agentDirs = Get-ChildItem -Path $subagentsSrc -Directory
        foreach ($agentDir in $agentDirs) {
            $agentName = $agentDir.Name
            $agentFile = Join-Path $agentDir.FullName "agent.md"

            if (Test-Path $agentFile) {
                Log-Info "Installing core agent: $agentName"

                # Read and filter the agent file
                $content = Get-Content $agentFile -Raw
                $lines = $content -split "`n"
                $output = @()
                $inFrontmatter = $false
                $frontmatterDone = $false

                foreach ($line in $lines) {
                    if ($line -eq "---") {
                        if (!$frontmatterDone) {
                            $output += $line
                            $inFrontmatter = !$inFrontmatter
                            if (!$inFrontmatter) { $frontmatterDone = $true }
                        } else {
                            $output += $line
                        }
                    } elseif ($inFrontmatter) {
                        # Keep only supported YAML fields
                        if ($line -match '^(name|description|tools|model|enabled):') {
                            $output += $line
                        }
                    } else {
                        $output += $line
                    }
                }

                $output -join "`n" | Out-File -FilePath (Join-Path $agentsDest "$agentName.md") -Encoding UTF8
            }
        }

        Log-Info "Core agents installed successfully"
    } else {
        Log-Warn "Subagents core directory not found in repository"
    }
}

function Install-Subagents {
    Write-Header "Installing Extended Subagents"

    $subagentsSrc = Join-Path $TresorDir "subagents"
    $subagentsDest = Join-Path $ClaudeCodeDir "subagents"

    if (Test-Path $subagentsSrc) {
        Log-Info "Installing subagents to: $subagentsDest"

        if (Test-Path $subagentsDest) {
            Remove-Item -Path $subagentsDest -Recurse -Force
        }

        Copy-Item -Path $subagentsSrc -Destination $subagentsDest -Recurse -Force

        $agentCount = (Get-ChildItem -Path $subagentsDest -Filter "agent.md" -Recurse -File | Measure-Object).Count
        Log-Info "Installed $agentCount subagents across 10 categories"
        Log-Info "Claude Code Tresor Subagents installed successfully"
    } else {
        Log-Warn "Claude Code Tresor Subagents directory not found in repository"
    }
}

function Install-Skills {
    Write-Header "Installing Autonomous Skills"

    $skillsSrc = Join-Path $TresorDir "skills"
    $skillsDest = Join-Path $ClaudeCodeDir "skills"

    if (Test-Path $skillsSrc) {
        Log-Info "Installing skills to: $skillsDest"

        # Get all skill directories (2 levels deep)
        $categories = Get-ChildItem -Path $skillsSrc -Directory
        foreach ($category in $categories) {
            if ($category.Name -eq "README.md" -or $category.Name -eq "TEMPLATES.md") { continue }

            $skillDirs = Get-ChildItem -Path $category.FullName -Directory
            foreach ($skillDir in $skillDirs) {
                $skillName = $skillDir.Name
                $skillFile = Join-Path $skillDir.FullName "SKILL.md"

                if (Test-Path $skillFile) {
                    Log-Info "Installing skill: $skillName (from $($category.Name))"
                    $destDir = Join-Path $skillsDest $skillName
                    Copy-Item -Path $skillDir.FullName -Destination $destDir -Recurse -Force
                }
            }
        }

        Log-Info "Claude Code Tresor Skills installed successfully"
    } else {
        Log-Warn "Skills directory not found in repository"
    }
}

function Install-Resources {
    Write-Header "Installing Resources"

    $resourcesDest = Join-Path $ClaudeCodeDir "tresor-resources"
    Log-Info "Installing resources to: $resourcesDest"

    if (!(Test-Path $resourcesDest)) {
        New-Item -ItemType Directory -Path $resourcesDest -Force | Out-Null
    }

    $resourceDirs = @("prompts", "standards", "examples")
    foreach ($resource in $resourceDirs) {
        $srcPath = Join-Path $TresorDir $resource
        if (Test-Path $srcPath) {
            Log-Info "Installing $resource resources"
            Copy-Item -Path $srcPath -Destination $resourcesDest -Recurse -Force
        }
    }

    # Copy documentation
    $docFiles = @("README.md", "CONTRIBUTING.md")
    foreach ($doc in $docFiles) {
        $srcDoc = Join-Path $TresorDir $doc
        if (Test-Path $srcDoc) {
            Copy-Item -Path $srcDoc -Destination $resourcesDest -Force
        }
    }

    Log-Info "Resources installed successfully"
}

function New-ConfigFile {
    Write-Header "Creating Configuration"

    $configFile = Join-Path $ClaudeCodeDir "tresor.config.json"
    Log-Info "Creating configuration file: $configFile"

    $config = @{
        version = "1.0.0"
        installed = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        author = "Alireza Rezvani"
        repository = $RepoUrl
        directories = @{
            commands = Join-Path $ClaudeCodeDir "commands"
            agents = Join-Path $ClaudeCodeDir "agents"
            resources = Join-Path $ClaudeCodeDir "tresor-resources"
        }
    } | ConvertTo-Json -Depth 10

    $config | Out-File -FilePath $configFile -Encoding UTF8
    Log-Info "Configuration created successfully"
}

function Show-Summary {
    Write-Header "Installation Summary"

    Write-ColorOutput "Claude Code Tresor installed successfully!" Green
    Write-Host ""

    Write-Host "Installation Location:"
    Write-Host "   $ClaudeCodeDir"
    Write-Host ""

    Write-Host "Core Workflow Commands (4):"
    Write-Host "   /scaffold    - Generate project structures and components"
    Write-Host "   /review      - Automated code review with best practices"
    Write-Host "   /test-gen    - Generate comprehensive test suites"
    Write-Host "   /docs-gen    - Create documentation from code"
    Write-Host ""

    Write-Host "Tresor Workflow Commands (5):"
    Write-Host "   /prompt-create  - Generate optimized prompts for complex tasks"
    Write-Host "   /prompt-run     - Execute prompts in sub-agents"
    Write-Host "   /todo-add       - Capture ideas with full context"
    Write-Host "   /todo-check     - Resume work on todos"
    Write-Host "   /handoff-create - Create comprehensive context handoff"
    Write-Host ""

    Write-Host "Orchestration Commands (10) - v2.7.0:"
    Write-Host "   Security:      /audit, /vulnerability-scan, /compliance-check"
    Write-Host "   Performance:   /profile, /benchmark"
    Write-Host "   Operations:    /deploy-validate, /health-check, /incident-response"
    Write-Host "   Quality:       /code-health, /debt-analysis"
    Write-Host ""

    Write-Host "Core Agents (8):"
    Write-Host "   @systems-architect        - System design and architecture"
    Write-Host "   @config-safety-reviewer   - Configuration safety"
    Write-Host "   @root-cause-analyzer      - Debugging and RCA"
    Write-Host "   @security-auditor         - Security and OWASP"
    Write-Host "   @test-engineer            - Testing specialist"
    Write-Host "   @performance-tuner        - Performance optimization"
    Write-Host "   @refactor-expert          - Code refactoring"
    Write-Host "   @docs-writer              - Documentation"
    Write-Host ""

    Write-Host "Extended Subagents (133+):"
    Write-Host "   Browse: $ClaudeCodeDir\subagents\"
    Write-Host ""

    Write-Host "Resources:"
    Write-Host "   Prompts: $ClaudeCodeDir\tresor-resources\prompts"
    Write-Host "   Standards: $ClaudeCodeDir\tresor-resources\standards"
    Write-Host "   Examples: $ClaudeCodeDir\tresor-resources\examples"
    Write-Host ""

    Write-ColorOutput "Happy coding with Claude Code Tresor!" Green
}

# Main Installation
function Main {
    Write-Header "Claude Code Tresor Installation (Windows)"
    Write-Host "Author: Alireza Rezvani"
    Write-Host "Repository: $RepoUrl"
    Write-Host "Installation Directory: $ClaudeCodeDir"
    Write-Host ""

    Test-GitInstalled
    Initialize-Directories
    Backup-ExistingConfig
    Clone-Repository

    if ($SkillsOnly) {
        Install-Skills
    } elseif ($CommandsOnly) {
        Install-Commands
    } elseif ($AgentsOnly) {
        Install-Agents
        Install-Subagents
    } elseif ($OrchestrationOnly) {
        Install-OrchestrationCommands
    } elseif ($ResourcesOnly) {
        Install-Resources
    } elseif ($Update) {
        Install-Skills
        Install-Commands
        Install-Agents
        Install-Subagents
        Install-Resources
        Log-Info "Claude Code Tresor Update completed successfully"
    } else {
        # Full installation
        Install-Skills
        Install-Commands
        Install-Agents
        Install-Subagents
        Install-Resources
        New-ConfigFile
    }

    Show-Summary
}

# Run main function
Main
