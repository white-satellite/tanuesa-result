Param(
  [string]$GoVersion = "1.21.13",
  [switch]$Cleanup
)

$ErrorActionPreference = 'Stop'
$Root = Resolve-Path (Join-Path $PSScriptRoot '..')
$Tools = Join-Path $Root 'tools'
$GoRoot = Join-Path $Tools 'go'
$Zip = Join-Path $Tools ("go$GoVersion.windows-amd64.zip")

Write-Host "[+] Prepare tools dir: $Tools"
New-Item -ItemType Directory -Force -Path $Tools *> $null

$Url = "https://go.dev/dl/go$GoVersion.windows-amd64.zip"
Write-Host "[+] Download Go $GoVersion ..."
Invoke-WebRequest -Uri $Url -OutFile $Zip

Write-Host "[+] Extract ..."
if (Test-Path $GoRoot) { Remove-Item -Recurse -Force $GoRoot }
Expand-Archive -LiteralPath $Zip -DestinationPath $Tools -Force

$env:GOROOT = $GoRoot
$env:PATH = "$GoRoot\bin;$env:PATH"
Write-Host "[+] Go version: " -NoNewline; & go version

Push-Location $Root
try {
  Write-Host "[+] Build windows/amd64 ..."
  $env:GOOS = 'windows'
  $env:GOARCH = 'amd64'
  & go build -o gacha.exe ./src
  Write-Host "[+] Built: $(Join-Path $Root 'gacha.exe')"
}
finally {
  Pop-Location
}

if ($Cleanup) {
  Write-Host "[+] Cleanup tools ..."
  Remove-Item -Recurse -Force $Tools
}

Write-Host "[+] Done."

