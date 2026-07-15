# MothX online installer for Windows
# Preferred entry point: https://mothx.net/install.bat

param(
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"
$Package = "mothx-installer"

function Write-Info($Message) { Write-Host "[INFO] $Message" -ForegroundColor Cyan }
function Write-Success($Message) { Write-Host "[SUCCESS] $Message" -ForegroundColor Green }

if ($Uninstall) {
    if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
        throw "npm is required to uninstall MothX."
    }
    & npm uninstall -g $Package
    if ($LASTEXITCODE -ne 0) { throw "npm uninstall failed." }
    Write-Success "MothX uninstalled."
    exit 0
}

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Info "Node.js was not found. Installing the latest Node.js LTS release..."
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        & winget install --id OpenJS.NodeJS.LTS --exact --source winget --accept-source-agreements --accept-package-agreements
        if ($LASTEXITCODE -ne 0) { throw "winget could not install Node.js." }
    } elseif (Get-Command choco -ErrorAction SilentlyContinue) {
        & choco install nodejs-lts -y
        if ($LASTEXITCODE -ne 0) { throw "Chocolatey could not install Node.js." }
    } else {
        $release = Invoke-RestMethod "https://nodejs.org/dist/index.json" |
            Where-Object { $_.lts -ne $false } |
            Select-Object -First 1
        if (-not $release) { throw "Unable to find the latest Node.js LTS release." }

        $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "x64" }
        if ($release.files -notcontains "win-$arch-msi") {
            throw "Node.js LTS is not available for this Windows architecture."
        }
        $installer = Join-Path $env:TEMP "mothx-node.msi"
        $url = "https://nodejs.org/dist/$($release.version)/node-$($release.version)-$arch.msi"
        Invoke-WebRequest $url -OutFile $installer
        $process = Start-Process msiexec.exe -ArgumentList "/i", $installer, "/qn", "/norestart" -Wait -PassThru
        Remove-Item $installer -Force -ErrorAction SilentlyContinue
        if ($process.ExitCode -ne 0) { throw "Node.js installer failed with exit code $($process.ExitCode)." }
    }
    $env:Path = "$env:ProgramFiles\nodejs;$env:APPDATA\npm;$env:Path"
} else {
    Write-Info "Using Node.js $(& node --version)."
}

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    throw "npm was not found. Open a new terminal and run the installer again."
}

Write-Info "Installing the latest MothX release..."
& npm install -g $Package
if ($LASTEXITCODE -ne 0) { throw "npm installation failed." }
Write-Success "MothX installed successfully."

if (-not (Get-Command mothx -ErrorAction SilentlyContinue)) {
    Write-Host "[WARN] Open a new terminal if the mothx command is not yet in PATH." -ForegroundColor Yellow
}
