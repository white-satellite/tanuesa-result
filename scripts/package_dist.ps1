Param(
  [string]$OutDir = "app",
  [switch]$WithVersion
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Root = Resolve-Path (Join-Path $PSScriptRoot '..')
$Exe = Join-Path $Root 'gacha.exe'
if (-not (Test-Path $Exe)) { throw "gacha.exe not found: $Exe" }

try { $ver = (& $Exe --version) -join '' } catch { $ver = '' }
if (-not $ver) { $ver = '0.0.0' }

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$stage = Join-Path $OutDir 'pkg_temp'
if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
New-Item -ItemType Directory -Force -Path $stage | Out-Null

# layout: gacha.exe / README.md / public/ / scripts
Copy-Item -LiteralPath $Exe -Destination (Join-Path $stage 'gacha.exe') -Force
$readme = Join-Path $Root 'README.md'
if (Test-Path $readme) { Copy-Item -LiteralPath $readme -Destination (Join-Path $stage 'README.md') -Force }

$pubSrc = Join-Path $Root 'public'
if (Test-Path $pubSrc) { Copy-Item -LiteralPath $pubSrc -Destination (Join-Path $stage 'public') -Recurse -Force }

# create runtime directories (ensure present after unzip)
foreach ($d in @('data','logs','backups')) {
  $dirPath = Join-Path $stage $d
  New-Item -ItemType Directory -Force -Path $dirPath | Out-Null
  # place a placeholder to keep the directory in ZIP
  $keep = Join-Path $dirPath '.keep'
  @(
    "This placeholder ensures the '$d' directory exists after unzip.",
    "It may be safely deleted at runtime."
  ) | Set-Content -LiteralPath $keep -Encoding UTF8
}

# seed empty state file: data/current.json
$curJson = Join-Path $stage 'data/current.json'
$stateObj = @{ users = @(); updatedAt = (Get-Date).ToUniversalTime().ToString('o') }
$stateObj | ConvertTo-Json | Set-Content -LiteralPath $curJson -Encoding UTF8

# seed settings: setting.json�i���[�g�̌��s�ݒ�����̂܂܍̗p�j
$rootSetting = Join-Path $Root 'setting.json'
if (Test-Path $rootSetting) {
  Copy-Item -LiteralPath $rootSetting -Destination (Join-Path $stage 'setting.json') -Force
} else {
  # �t�H�[���o�b�N: �Œ���̊���l�𖄂߂�iroot��setting.json�������ꍇ�j
  $settingsObj = @{ eventJsonLog = $false; autoServe = $true; serverPort = 3010; discordEnabled = $true }
  $settingsObj | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $stage 'setting.json') -Encoding UTF8
}

$scriptOut = Join-Path $stage 'scripts'
New-Item -ItemType Directory -Force -Path $scriptOut | Out-Null
$includeScripts = @(
  'scripts/update_result.ps1',
  'scripts/update_result.bat',
  'scripts/reset.bat',
  'scripts/restore.bat',
  'scripts/new_session.bat'
)
foreach ($rel in $includeScripts) {
  $src = Join-Path $Root $rel
  if (Test-Path $src) {
    Copy-Item -LiteralPath $src -Destination (Join-Path $scriptOut (Split-Path $rel -Leaf)) -Force
  }
}

# prefer serve_api.bat; if missing, fallback to serve_api_cmd.bat as serve_api.bat
$serveBat = Join-Path $Root 'scripts/serve_api.bat'
$serveCmd = Join-Path $Root 'scripts/serve_api_cmd.bat'
if (Test-Path $serveBat) {
  Copy-Item -LiteralPath $serveBat -Destination (Join-Path $scriptOut 'serve_api.bat') -Force
} elseif (Test-Path $serveCmd) {
  Copy-Item -LiteralPath $serveCmd -Destination (Join-Path $scriptOut 'serve_api.bat') -Force
}

$zipName = if ($WithVersion) { "gacha-$ver-windows-x64.zip" } else { "gacha-windows-x64.zip" }
$zipPath = Join-Path $OutDir $zipName
if (Test-Path $zipPath) { Remove-Item -LiteralPath $zipPath -Force }
Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $zipPath -Force

Remove-Item -Recurse -Force $stage
Write-Host "Wrote: $zipPath"
exit 0
