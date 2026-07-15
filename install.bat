@echo off
setlocal EnableExtensions

rem MothX online installer for Windows
rem Usage: curl.exe -fsSL https://mothx.net/install.bat -o install.bat ^&^& install.bat

where node >nul 2>&1
if errorlevel 1 (
  echo [INFO] Node.js was not found. Installing the latest Node.js LTS release...
  where winget >nul 2>&1
  if not errorlevel 1 (
    winget install --id OpenJS.NodeJS.LTS --exact --source winget --accept-source-agreements --accept-package-agreements
  ) else (
    where choco >nul 2>&1
    if not errorlevel 1 (
      choco install nodejs-lts -y
    ) else (
      echo [INFO] Downloading the latest Node.js LTS installer...
      powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$r=Invoke-RestMethod 'https://nodejs.org/dist/index.json' ^| Where-Object { $_.lts -ne $false } ^| Select-Object -First 1; if (-not $r) { throw 'Unable to find Node.js LTS' }; $a=if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'x64' }; if ($r.files -notcontains ('win-'+$a+'-msi')) { throw 'This Windows architecture is not supported' }; $p=Join-Path $env:TEMP 'mothx-node.msi'; Invoke-WebRequest ('https://nodejs.org/dist/'+$r.version+'/node-'+$r.version+'-'+$a+'.msi') -OutFile $p; Start-Process msiexec.exe -ArgumentList '/i',$p,'/qn','/norestart' -Wait -PassThru ^| ForEach-Object { if ($_.ExitCode -ne 0) { throw ('Node.js installation failed: '+$_.ExitCode) } }; Remove-Item $p -Force"
      if errorlevel 1 goto :error
    )
  )
  rem Refresh PATH after a package-manager installation.
  set "PATH=%ProgramFiles%\nodejs;%APPDATA%\npm;%PATH%"
)

where npm >nul 2>&1
if errorlevel 1 (
  echo [ERROR] npm was not found. Open a new Command Prompt and run this script again.
  goto :error
)

echo [INFO] Installing the latest MothX release...
npm install -g mothx-installer
if errorlevel 1 goto :error

echo [SUCCESS] MothX installed successfully.
where mothx >nul 2>&1
if errorlevel 1 echo [WARN] Open a new Command Prompt if the mothx command is not yet in PATH.
exit /b 0

:error
echo [ERROR] Installation failed.
exit /b 1
