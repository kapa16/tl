$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Go = if (Test-Path "F:\tl\tools\go-portable\go\bin\go.exe") { "F:\tl\tools\go-portable\go\bin\go.exe" } else { "go" }
$OutDir = Join-Path $Root "dist"
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
Push-Location $Root
& $Go build -o (Join-Path $OutDir "FuelStatementOCR.exe") ./cmd/fuel-statement-ocr
Pop-Location
Write-Host "Built: $OutDir\FuelStatementOCR.exe"
