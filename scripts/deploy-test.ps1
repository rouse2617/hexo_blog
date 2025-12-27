param(
  [string]$ConfigPath = "config.yaml",
  [int]$Port = 8080,
  [switch]$NoFrontend,
  [switch]$NoBackendBuild,
  [switch]$SkipKillPort,
  [string]$TestHostId = ""
)

$ErrorActionPreference = "Stop"

function Write-Info([string]$msg) { Write-Host "[INFO] $msg" -ForegroundColor Cyan }
function Write-Warn([string]$msg) { Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Write-Err([string]$msg) { Write-Host "[ERR ] $msg" -ForegroundColor Red }

function Get-RepoRoot {
  return (Resolve-Path (Join-Path $PSScriptRoot ".."))
}

function Kill-Port([int]$p) {
  $conns = @(Get-NetTCPConnection -LocalPort $p -State Listen -ErrorAction SilentlyContinue)
  if ($conns.Count -eq 0) {
    Write-Info "Port $p not in use"
    return
  }
  $pids = $conns | Select-Object -ExpandProperty OwningProcess -Unique
  foreach ($pid in $pids) {
    try {
      $proc = Get-Process -Id $pid -ErrorAction SilentlyContinue
      if ($null -ne $proc) {
        Write-Warn "Killing process on port ${p}: $($proc.ProcessName) (PID ${pid})"
      } else {
        Write-Warn "Killing PID ${pid} on port ${p}"
      }
      Stop-Process -Id $pid -Force
    } catch {
      Write-Warn "Failed to kill PID ${pid}: $($_.Exception.Message)"
    }
  }
}

function Wait-HttpOk([string]$url, [int]$timeoutSeconds = 30) {
  $deadline = (Get-Date).AddSeconds($timeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $r = Invoke-WebRequest -Uri $url -Method Get -UseBasicParsing -TimeoutSec 5
      if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 300) { return $true }
    } catch {
      Start-Sleep -Milliseconds 500
    }
  }
  return $false
}

$root = Get-RepoRoot
Write-Info "Repo root: $root"

if (-not (Test-Path (Join-Path $root $ConfigPath))) {
  throw "Config not found: $ConfigPath"
}

if (-not $NoFrontend) {
  Write-Info "Building frontend..."
  $webDir = Join-Path $root "web"
  if (-not (Test-Path $webDir)) { throw "web dir not found" }
  Push-Location $webDir
  try {
    if (-not (Test-Path (Join-Path $webDir "node_modules"))) {
      Write-Info "Installing frontend deps (first time)..."
      npm install
    }
    npm run build
  } finally {
    Pop-Location
  }
}

if (-not $NoBackendBuild) {
  Write-Info "Building backend..."
  Push-Location $root
  try {
    go build -o (Join-Path $root "bin\ai-ops.exe") .\cmd\server
  } finally {
    Pop-Location
  }
}

if (-not $SkipKillPort) {
  Kill-Port $Port
}

Write-Info "Starting backend on :$Port with config: $ConfigPath"
$backendExe = Join-Path $root "bin\ai-ops.exe"
if (-not (Test-Path $backendExe)) {
  Write-Warn "Backend exe not found at $backendExe; falling back to 'go run'."
  $proc = Start-Process -FilePath "go" -ArgumentList @("run",".\cmd\server") -WorkingDirectory $root -PassThru
} else {
  $env:CONFIG_PATH = $ConfigPath
  $proc = Start-Process -FilePath $backendExe -WorkingDirectory $root -PassThru
}

$healthUrl = "http://localhost:$Port/api/system/health"
Write-Info "Waiting for health: $healthUrl"
if (-not (Wait-HttpOk -url $healthUrl -timeoutSeconds 40)) {
  try { Stop-Process -Id $proc.Id -Force } catch {}
  throw "Backend did not become healthy in time"
}
Write-Info "Backend is healthy (PID $($proc.Id))"

if ($TestHostId -ne "") {
  $testUrl = "http://localhost:$Port/api/hosts/$TestHostId/test"
  Write-Info "Testing host connection: $testUrl"
  try {
    $resp = Invoke-RestMethod -Uri $testUrl -Method Post
    $resp | ConvertTo-Json -Depth 6 | Write-Host
  } catch {
    Write-Err "Host test failed: $($_.Exception.Message)"
    throw
  }
}

Write-Info "Done. Backend PID: $($proc.Id)"
Write-Info "Open UI: http://localhost:$Port/"
