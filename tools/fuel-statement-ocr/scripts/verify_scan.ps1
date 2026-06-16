$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Exe = Join-Path $Root "dist\FuelStatementOCR.exe"
$Scans = Join-Path (Split-Path -Parent $Root) "..\scans"
if (-not (Test-Path $Scans)) {
    $Scans = "F:\tl\scans"
}

$cases = @(
    @{ File = "ведомости.1.jpg"; Type = "perelivnaya" },
    @{ File = "ведомости.2.jpg"; Type = "perelivnaya" },
    @{ File = "ведомости.3.jpg"; Type = "zapravka" },
    @{ File = "ведомости.4.jpg"; Type = "prihodnaya" }
)

if (-not (Test-Path $Exe)) {
    & (Join-Path $Root "build.ps1")
}

$failed = 0
foreach ($c in $cases) {
    $path = Join-Path $Scans $c.File
    if (-not (Test-Path $path)) {
        Write-Warning "Skip missing scan: $path"
        continue
    }
    $json = & $Exe --type=$($c.Type) $path 2>$null | Out-String
    $obj = $json | ConvertFrom-Json
    $rowCount = @($obj.rows).Count
    $orient = $obj.layout.orientationApplied
    Write-Host ("{0} type={1} orient={2} rows={3} refConf={4}" -f $c.File, $c.Type, $orient, $rowCount, $obj.referenceDigits.confidence)
    if ($rowCount -eq 0 -and $c.File -eq "ведомости.4.jpg") {
        Write-Warning "prihodnaya scan .4: rows=0 (expected >= 1)"
        $failed++
    }
}

if ($failed -gt 0) {
    exit 1
}
