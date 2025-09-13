Param(
  [int]$Port,
  [switch]$Verbose
)

$ErrorActionPreference = 'SilentlyContinue'
function Get-ServerPort {
  if ($Port) { return $Port }
  $root = Resolve-Path (Join-Path $PSScriptRoot '..')
  $setPath = Join-Path $root 'setting.json'
  if (Test-Path $setPath) {
    try {
      $json = Get-Content -Raw -Path $setPath | ConvertFrom-Json
      if ($json.serverPort) { return [int]$json.serverPort }
    } catch {}
  }
  return 3010
}

function Stop-ByPort([int]$p){
  $killed = 0
  # Prefer Get-NetTCPConnection (Win10+)
  try {
    $conns = Get-NetTCPConnection -LocalPort $p -State Listen -ErrorAction Stop
    foreach ($c in $conns) {
      if ($Verbose) { Write-Host "[INFO] Killing PID $($c.OwningProcess) on port $p" }
      Stop-Process -Id $c.OwningProcess -Force -ErrorAction SilentlyContinue
      $killed++
    }
  } catch {
    # Fallback to netstat
    $lines = netstat -ano | Select-String ":$p .*LISTENING" | ForEach-Object { $_.ToString() }
    foreach ($ln in $lines) {
      $pid = ($ln.Trim() -split '\\s+')[-1]
      if ($pid -match '^\d+$') {
        if ($Verbose) { Write-Host "[INFO] Killing PID $pid on port $p (fallback)" }
        Stop-Process -Id [int]$pid -Force -ErrorAction SilentlyContinue
        $killed++
      }
    }
  }
  return $killed
}

$p = Get-ServerPort
$n = Stop-ByPort -p $p
if ($n -gt 0) {
  Write-Host "Stopped $n process(es) on port $p"
  exit 0
} else {
  Write-Host "No listening process found on port $p"
  exit 2
}

