# ── Auto-Clean Zombie Processes ─────────────────────────────────
Write-Host "Cleaning up old ghost processes..." -ForegroundColor DarkYellow
Stop-Process -Name "main" -Force -ErrorAction SilentlyContinue
# Comment to kill Go.exe
# Stop-Process -Name "go" -Force -ErrorAction SilentlyContinue
# Set D-drive caching to completely prevent your C: drive from filling up!
$env:GOMODCACHE="D:\Code\projects\omnisync_wms\go_cache\pkg\mod"
$env:GOCACHE="D:\Code\projects\omnisync_wms\go_cache\build"

# ── Helper: load a .env file into the current session ──────────────────────────
function Import-DotEnv($path) {
    if (-not (Test-Path $path)) {
        Write-Host "WARNING: $path not found. Copy .env.example to .env and fill in values." -ForegroundColor Red
        exit 1
    }
    Get-Content $path | Where-Object { $_ -match '^\s*[^#]' -and $_ -match '=' } | ForEach-Object {
        $key, $val = $_ -split '=', 2
        [System.Environment]::SetEnvironmentVariable($key.Trim(), $val.Trim(), "Process")
    }
}

# Load environment variables from each service's .env
Import-DotEnv "D:\Code\projects\omnisync_wms\auth_services\.env"
Import-DotEnv "D:\Code\projects\omnisync_wms\wms_dashboard\.env"

# Configure dynamic ports (allow override, fall back to .env / defaults)
$AUTH_PORT = if ($env:AUTH_PORT) { $env:AUTH_PORT } else { "8000" }
$WMS_PORT  = if ($env:WMS_PORT)  { $env:WMS_PORT }  else { "9901" }
$JWT_KEY   = $env:JWT_SECRET_KEY

Clear-Host
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "          OMNISYNC WMS DECOUPLED SUITE            " -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "Dependencies path: D:\Code\projects\omnisync_wms\go_cache" -ForegroundColor Green
Write-Host ""

# Launch Auth Service — forward all required env vars into the child window
Write-Host "[1/2] Booting Standalone Auth Service (Port $AUTH_PORT)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", @"
  `$env:PORT='$AUTH_PORT';
  `$env:JWT_SECRET_KEY='$JWT_KEY';
  `$env:ALLOWED_ORIGIN='http://localhost:${WMS_PORT},http://127.0.0.1:${WMS_PORT}';
  `$env:DB_TYPE='$env:DB_TYPE';
  `$env:AUTH_DATABASE_URL='$env:AUTH_DATABASE_URL';
  `$env:GOMODCACHE='D:\Code\projects\omnisync_wms\go_cache\pkg\mod';
  `$env:GOCACHE='D:\Code\projects\omnisync_wms\go_cache\build';
  cd D:\Code\projects\omnisync_wms\auth_services;
  go run cmd/main.go
"@

# Launch WMS Dashboard — forward all required env vars into the child window
Write-Host "[2/2] Booting WMS Dashboard Service (Port $WMS_PORT)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", @"
  `$env:PORT='$WMS_PORT';
  `$env:JWT_SECRET_KEY='$JWT_KEY';
  `$env:AUTH_API_URL='http://localhost:${AUTH_PORT}';
  `$env:DB_TYPE='$env:DB_TYPE';
  `$env:WMS_DATABASE_URL='$env:WMS_DATABASE_URL';
  `$env:GOMODCACHE='D:\Code\projects\omnisync_wms\go_cache\pkg\mod';
  `$env:GOCACHE='D:\Code\projects\omnisync_wms\go_cache\build';
  cd D:\Code\projects\omnisync_wms\wms_dashboard;
  go run cmd/main.go
"@

Write-Host ""
Write-Host "--------------------------------------------------" -ForegroundColor Gray
Write-Host "SUCCESS: Both services launched in separate windows!" -ForegroundColor Green
Write-Host "1. Authentication API:        http://localhost:$AUTH_PORT" -ForegroundColor Gray
Write-Host "2. WMS Operational Dashboard: http://localhost:$WMS_PORT" -ForegroundColor Gray
Write-Host "==================================================" -ForegroundColor Cyan
