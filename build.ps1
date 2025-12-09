# ===============================
#  Build MetricsAllerts binaries
# ===============================

Write-Host "=== Building Windows binaries ==="

# Create directories if needed
New-Item -ItemType Directory -Force -Path "cmd/server" | Out-Null
New-Item -ItemType Directory -Force -Path "cmd/agent"  | Out-Null

# Build Windows server.exe
go build -o "cmd/server/server.exe" "./cmd/server"
if ($LASTEXITCODE -ne 0) { Write-Error "Failed to build server.exe"; exit 1 }

# Build Windows agent.exe
go build -o "cmd/agent/agent.exe" "./cmd/agent"
if ($LASTEXITCODE -ne 0) { Write-Error "Failed to build agent.exe"; exit 1 }

Write-Host "Windows builds complete."
Write-Host ""


# ===============================
# Build Linux binaries (CI-like)
# ===============================
Write-Host "=== Building Linux CI-style binaries ==="

$env:GOOS  = "linux"
$env:GOARCH = "amd64"

# Build Linux server
go build -o "cmd/server/server" "./cmd/server"
if ($LASTEXITCODE -ne 0) { Write-Error "Failed to build Linux server"; exit 1 }

# Build Linux agent
go build -o "cmd/agent/agent" "./cmd/agent"
if ($LASTEXITCODE -ne 0) { Write-Error "Failed to build Linux agent"; exit 1 }

Write-Host "Linux builds complete."
Write-Host ""

# Reset GOOS/GOARCH to Windows defaults
Remove-Item Env:GOOS
Remove-Item Env:GOARCH

Write-Host "=== All builds finished successfully ==="
