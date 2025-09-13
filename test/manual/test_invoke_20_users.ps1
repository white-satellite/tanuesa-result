Param(
  [string]$Prefix = "USER",
  [int]$Count = 20,
  [switch]$Reset
)

$ErrorActionPreference = 'Stop'
$Exe = Join-Path $PSScriptRoot '..\..\gacha.exe'
if (-not (Test-Path $Exe)) { throw "gacha.exe not found: $Exe" }

if ($Reset) { & $Exe reset | Out-Null }

# 20 users x 1 update. Multiples of 5 => jackpot(1), others => hit(0).
for ($i=1; $i -le $Count; $i++) {
  $name = "{0}{1}" -f $Prefix, ($i.ToString('00'))
  if (($i % 5) -eq 0) { $flag = 1 } else { $flag = 0 }
  & $Exe $name $flag | Out-Null
  if ($LASTEXITCODE -ne 0) { throw "gacha.exe exited with code $LASTEXITCODE (name=$name flag=$flag)" }
}

# Verify
$statePath = Join-Path $PSScriptRoot '..\..\data\current.json'
if (-not (Test-Path $statePath)) { throw "current.json not found: $statePath" }
$json = Get-Content -Raw -Path $statePath | ConvertFrom-Json

for ($i=1; $i -le $Count; $i++) {
  $name = "{0}{1}" -f $Prefix, ($i.ToString('00'))
  $u = $json.users | Where-Object { $_.name -eq $name }
  if (-not $u) { throw "user not found: $name" }
  $isJack = (($i % 5) -eq 0)
  if ($isJack) { $expectedHit = 0 } else { $expectedHit = 1 }
  if ($isJack) { $expectedJack = 1 } else { $expectedJack = 0 }
  if ($u.hit -ne $expectedHit) { throw "hit mismatch ($name): actual=$($u.hit) expected=$expectedHit" }
  if ($u.jackpot -ne $expectedJack) { throw "jackpot mismatch ($name): actual=$($u.jackpot) expected=$expectedJack" }
  $illust = [bool]($u.hit -ge 1)
  $gif = [bool]($u.hit -ge 3 -or $u.jackpot -ge 1)
  if ([bool]$u.flags.illust -ne $illust) { throw "illust flag mismatch ($name): actual=$([bool]$u.flags.illust) expected=$illust" }
  if ([bool]$u.flags.gif -ne $gif) { throw "gif flag mismatch ($name): actual=$([bool]$u.flags.gif) expected=$gif" }
  $present = $u.present
  $expectedPresent = if ($gif) { 'Gif' } elseif ($illust) { 'Illustration' } else { '' }
  if ($present -ne $expectedPresent) { throw "present mismatch ($name): actual=$present expected=$expectedPresent" }
}

Write-Host "OK: processed $Count users (multiples of 5 = jackpot)."
exit 0
