Param([switch]$DryRun)

$ErrorActionPreference = 'SilentlyContinue'
$targets = @(
  Join-Path $PSScriptRoot '..\scripts\dist',
  Join-Path $PSScriptRoot '..\dist'
)
foreach($t in $targets){
  if(Test-Path $t){
    if($DryRun){ Write-Host "[DRY] remove: $t" }
    else{ Remove-Item -Recurse -Force $t; Write-Host "removed: $t" }
  }
}
Write-Host "done."
