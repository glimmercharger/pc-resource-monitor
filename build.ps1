# Builds dist\resourcemonitor-portable.exe and dist\resourcemonitor-setup.exe
# Needs Go and Inno Setup 6 (ISCC.exe on PATH or pass -Iscc).
param(
    [string]$Version = "dev",
    [string]$Iscc = "ISCC.exe"
)
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
New-Item -ItemType Directory -Force dist | Out-Null

$env:CGO_ENABLED = "0"
go build -trimpath -ldflags "-s -w -X main.version=$Version" -o dist\resourcemonitor.exe .
if ($LASTEXITCODE) { exit $LASTEXITCODE }
Copy-Item dist\resourcemonitor.exe dist\resourcemonitor-portable.exe -Force

& $Iscc /Q "/DAppVersion=$($Version.TrimStart('v'))" installer\setup.iss
if ($LASTEXITCODE) { exit $LASTEXITCODE }
Get-ChildItem dist\*.exe | ForEach-Object { "{0,-32} {1,6:N1} MB" -f $_.Name, ($_.Length / 1MB) }
