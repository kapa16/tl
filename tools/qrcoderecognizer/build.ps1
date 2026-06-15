$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Go = if (Test-Path "F:\tl\tools\go-portable\go\bin\go.exe") { "F:\tl\tools\go-portable\go\bin\go.exe" } else { "go" }
$OutDir = Join-Path $Root "dist"
$OutExe = Join-Path $OutDir "QRCodeRecognizer.exe"
$RepoRoot = Resolve-Path (Join-Path $Root "..\..")
$EpfTemplate = Get-ChildItem -LiteralPath (Join-Path $RepoRoot "src\epf") -Recurse -Filter "Template.bin" |
    Where-Object { (Split-Path (Split-Path $_.FullName -Parent) -Parent | Split-Path -Leaf) -like "*QR*" } |
    Select-Object -First 1

if (-not $EpfTemplate) {
    throw "EPF template КомпонентаQR/Ext/Template.bin not found under src/epf"
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
Push-Location $Root
& $Go mod tidy
& $Go build -o $OutExe ./cmd/qrcoderecognizer
Pop-Location

Copy-Item -Force $OutExe $EpfTemplate.FullName
Write-Host "Built: $OutExe"
Write-Host "Copied to: $($EpfTemplate.FullName)"
