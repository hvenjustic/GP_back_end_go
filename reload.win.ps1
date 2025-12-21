$ErrorActionPreference = "Continue"

Write-Host "🚀 开始重载 Go 后端 (Windows)..."

$ProjectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectDir

Write-Host "🧱 编译并安装到 ./bin/ ..."
if (!(Test-Path "bin")) { New-Item -ItemType Directory "bin" | Out-Null }
if (!(Test-Path "logs")) { New-Item -ItemType Directory "logs" | Out-Null }
go build -ldflags="-w -s" -o "bin\back_end_go.exe" .

Write-Host "🔁 通过 PM2 重载应用..."
$Config = Join-Path $ProjectDir "ecosystem.config.win.js"
$AppName = "GP_back_end_go"

$Status = ""
$Desc = pm2 describe $AppName 2>$null
if ($LASTEXITCODE -eq 0) {
  $Match = $Desc | Select-String -Pattern "status\s*:\s*(\w+)" -AllMatches | Select-Object -First 1
  if ($Match) {
    $Status = $Match.Matches[0].Groups[1].Value
  }
}

switch ($Status) {
  "online" {
    pm2 reload $AppName
    if ($LASTEXITCODE -ne 0) {
      pm2 restart $AppName
      if ($LASTEXITCODE -ne 0) {
        pm2 delete $AppName | Out-Null
        pm2 start $Config
      }
    }
  }
  "stopped" { 
    Write-Host "ℹ️ $AppName 状态为 $Status，先 delete 再 start config ..."
    pm2 delete $AppName | Out-Null
    pm2 start $Config
  }
  "errored" {
    Write-Host "ℹ️ $AppName 状态为 $Status，先 delete 再 start config ..."
    pm2 delete $AppName | Out-Null
    pm2 start $Config
  }
  default {
    Write-Host "ℹ️ 未发现 $AppName，先清理后 start ..."
    pm2 delete $AppName | Out-Null
    pm2 start $Config
  }
}

Write-Host "当前 PM2 状态："
pm2 status $AppName

Write-Host "🎉 Go 后端已重载完成！"
