Param(
  [Parameter(Mandatory=$true)][string]$Winner,
  [Parameter(Mandatory=$true)][int]$HitFlag
)

$ErrorActionPreference = 'Stop'

# 実行ファイル（gacha-update）へのパスを自環境に合わせて設定してください
$Exe = Join-Path $PSScriptRoot '..\gacha.exe'

& $Exe --version *>$null # ウォームアップ（任意）
& $Exe $Winner $HitFlag
