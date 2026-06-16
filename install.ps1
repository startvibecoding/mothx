# VibeCoding Installer for Windows
# Downloads and installs the latest release from GitHub
#
# Repository: https://github.com/startvibecoding/vibecoding
# Gitee:      https://gitee.com/startvibecoding/vibecoding
# Author:     zhenruyan
# Blog:       https://pkold.com
#
# Usage:
#   # Install (default)
#   irm https://gitee.com/startvibecoding/vibecoding/raw/main/install.ps1 | iex
#
#   # Install to custom directory
#   $env:VIBECODING_INSTALL_DIR="C:\Tools\vibecoding"; irm https://gitee.com/startvibecoding/vibecoding/raw/main/install.ps1 | iex
#
#   # Uninstall
#   irm https://gitee.com/startvibecoding/vibecoding/raw/main/install.ps1 | iex; Uninstall-VibeCoding

$ErrorActionPreference = "Stop"

$REPO = "startvibecoding/vibecoding"
$BINARY_NAME = "vibecoding.exe"
$DEFAULT_INSTALL_DIR = "$env:LOCALAPPDATA\vibecoding"

# Colors
function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Cyan }
function Write-Success { Write-Host "[SUCCESS] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Error-Custom { Write-Host "[ERROR] $args" -ForegroundColor Red; exit 1 }

# Show help
function Show-Help {
    Write-Host ""
    Write-Host "╔═══════════════════════════════════════════════════════════════╗" -ForegroundColor DarkCyan
    Write-Host "║                   VibeCoding Installer                       ║" -ForegroundColor DarkCyan
    Write-Host "║         https://github.com/startvibecoding/vibecoding        ║" -ForegroundColor DarkCyan
    Write-Host "║                Author: zhenruyan | pkold.com                 ║" -ForegroundColor DarkCyan
    Write-Host "╚═══════════════════════════════════════════════════════════════╝" -ForegroundColor DarkCyan
    Write-Host ""
    Write-Host "Usage: install.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Help           Show this help message"
    Write-Host "  -Uninstall      Uninstall VibeCoding"
    Write-Host ""
    Write-Host "Environment variables:"
    Write-Host "  VIBECODING_INSTALL_DIR   Install directory (default: $env:LOCALAPPDATA\vibecoding)"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  # Install"
    Write-Host "  irm https://gitee.com/startvibecoding/vibecoding/raw/main/install.ps1 | iex"
    Write-Host ""
    Write-Host "  # Install to custom directory"
    Write-Host "  `$env:VIBECODING_INSTALL_DIR=`"C:\Tools\vibecoding`"; irm https://gitee.com/startvibecoding/vibecoding/raw/main/install.ps1 | iex"
    Write-Host ""
    Write-Host "  # Uninstall"
    Write-Host "  irm https://gitee.com/startvibecoding/vibecoding/raw/main/install.ps1 | iex; Uninstall-VibeCoding"
    Write-Host ""
}

# Uninstall VibeCoding
function Uninstall-VibeCoding {
    Write-Host ""
    Write-Host "╔═══════════════════════════════════════════════════════════════╗" -ForegroundColor DarkCyan
    Write-Host "║                 VibeCoding Uninstaller                       ║" -ForegroundColor DarkCyan
    Write-Host "╚═══════════════════════════════════════════════════════════════╝" -ForegroundColor DarkCyan
    Write-Host ""

    $foundPaths = @()

    # Check common install locations
    $checkDirs = @(
        "$env:LOCALAPPDATA\vibecoding",
        "$env:USERPROFILE\.vibecoding\bin",
        "$env:ProgramFiles\vibecoding"
    )

    foreach ($dir in $checkDirs) {
        $binPath = Join-Path $dir $BINARY_NAME
        if (Test-Path $binPath) {
            $foundPaths += $binPath
        }
    }

    # Also check PATH
    $vibecodingInPath = Get-Command vibecoding -ErrorAction SilentlyContinue
    if ($vibecodingInPath) {
        $whichPath = $vibecodingInPath.Source
        if ($foundPaths -notcontains $whichPath) {
            $foundPaths += $whichPath
        }
    }

    if ($foundPaths.Count -eq 0) {
        Write-Warn "VibeCoding not found in common locations"
        Write-Host ""
        Write-Host "Checked locations:"
        foreach ($dir in $checkDirs) {
            Write-Host "  - $dir\$BINARY_NAME"
        }
        Write-Host ""
        Write-Host "If installed elsewhere, remove it manually"
        Write-Host ""
        return
    }

    # Show found installations
    Write-Info "Found VibeCoding installations:"
    Write-Host ""
    foreach ($p in $foundPaths) {
        Write-Host "  - $p"
    }
    Write-Host ""

    # Ask for confirmation
    $answer = Read-Host "Remove all installations? [y/N]"
    if ($answer -ne 'y' -and $answer -ne 'Y') {
        Write-Info "Uninstall cancelled"
        return
    }

    # Remove binaries
    Write-Host ""
    foreach ($p in $foundPaths) {
        if (Test-Path $p) {
            try {
                Remove-Item -Path $p -Force
                Write-Success "Removed: $p"
            } catch {
                Write-Warn "Failed to remove: $p - $_"
            }
        }
    }

    # Ask about config directory
    Write-Host ""
    $configDir = Join-Path $env:APPDATA "vibecoding"
    if (Test-Path $configDir) {
        Write-Info "Config directory: $configDir"
        Write-Host ""
        $answer = Read-Host "Remove config directory ($configDir)? [y/N]"
        if ($answer -eq 'y' -or $answer -eq 'Y') {
            try {
                Remove-Item -Path $configDir -Recurse -Force
                Write-Success "Removed: $configDir"
            } catch {
                Write-Warn "Failed to remove: $configDir - $_"
            }
        } else {
            Write-Info "Kept: $configDir"
        }
    }

    # Clean PATH entries
    Write-Host ""
    Write-Info "Checking PATH for VibeCoding entries..."
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = if ($currentPath) { $currentPath -split ';' | Where-Object { $_ -ne '' } } else { @() }
    
    $vibecodingEntries = $pathEntries | Where-Object { $_ -like '*vibecoding*' }
    
    if ($vibecodingEntries.Count -gt 0) {
        Write-Info "Found VibeCoding PATH entries:"
        foreach ($entry in $vibecodingEntries) {
            Write-Host "  - $entry"
        }
        Write-Host ""
        $answer = Read-Host "Remove VibeCoding from PATH? [y/N]"
        if ($answer -eq 'y' -or $answer -eq 'Y') {
            $newEntries = $pathEntries | Where-Object { $_ -notlike '*vibecoding*' }
            $newPath = $newEntries -join ';'
            [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
            $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + $newPath
            Write-Success "Removed VibeCoding from PATH"
        }
    } else {
        Write-Info "No VibeCoding PATH entries found"
    }

    # Uninstall npm package if installed via npm
    Write-Host ""
    $npmCommand = Get-Command npm -ErrorAction SilentlyContinue
    if ($npmCommand) {
        $npmGlobalRoot = & npm root -g 2>$null
        $npmInstallerPath = Join-Path $npmGlobalRoot "vibecoding-installer"
        if (Test-Path $npmInstallerPath) {
            Write-Info "Found npm global installation"
            $answer = Read-Host "Uninstall npm package (vibecoding-installer)? [y/N]"
            if ($answer -eq 'y' -or $answer -eq 'Y') {
                try {
                    & npm uninstall -g vibecoding-installer
                    Write-Success "Uninstalled npm package"
                } catch {
                    Write-Warn "Failed to uninstall npm package: $_"
                }
            }
        }
    }

    Write-Host ""
    Write-Success "Uninstall complete!"
    Write-Host ""
    Write-Host "  Thank you for using VibeCoding! 🙏" -ForegroundColor White
    Write-Host ""
    Write-Host "  If you have any feedback, please visit:" -ForegroundColor White
    Write-Host "    - GitHub: https://github.com/startvibecoding/vibecoding" -ForegroundColor Cyan
    Write-Host "    - Gitee:  https://gitee.com/startvibecoding/vibecoding" -ForegroundColor Cyan
    Write-Host ""
}

# Parse arguments
param(
    [switch]$Help,
    [switch]$Uninstall
)

if ($Help) {
    Show-Help
    exit 0
}

if ($Uninstall) {
    Uninstall-VibeCoding
    exit 0
}

# Banner
Write-Host ""
Write-Host "╔═══════════════════════════════════════════════════════════════╗" -ForegroundColor DarkCyan
Write-Host "║                   VibeCoding Installer                       ║" -ForegroundColor DarkCyan
Write-Host "║         https://github.com/startvibecoding/vibecoding        ║" -ForegroundColor DarkCyan
Write-Host "║                Author: zhenruyan | pkold.com                 ║" -ForegroundColor DarkCyan
Write-Host "╚═══════════════════════════════════════════════════════════════╝" -ForegroundColor DarkCyan
Write-Host ""

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { Write-Error-Custom "32-bit systems are not supported" }
Write-Info "Detected architecture: windows/$arch"

# Get install directory
$installDir = if ($env:VIBECODING_INSTALL_DIR) { $env:VIBECODING_INSTALL_DIR } else { $DEFAULT_INSTALL_DIR }
Write-Info "Install directory: $installDir"

# Get latest version from GitHub
Write-Info "Fetching latest version..."
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -Headers @{
        "Accept" = "application/vnd.github.v3+json"
    }
    $version = $release.tag_name
    Write-Info "Latest version: $version"
} catch {
    Write-Error-Custom "Failed to fetch latest version: $_"
}

# Find download URL
$versionNum = $release.tag_name -replace '^v', ''
$archiveName = "vibecoding-${versionNum}-windows-$arch.zip"
$asset = $release.assets | Where-Object { $_.name -eq $archiveName }

if (-not $asset) {
    Write-Error-Custom "Release asset not found: $archiveName"
}

$downloadUrl = $asset.browser_download_url
Write-Info "Download URL: $downloadUrl"

# Create temp directory
$tempDir = Join-Path $env:TEMP "vibecoding-install-$(Get-Random)"
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    # Download archive
    $archivePath = Join-Path $tempDir $archiveName
    Write-Info "Downloading $archiveName..."

    $progressPreference = 'SilentlyContinue'
    Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
    $progressPreference = 'Continue'

    Write-Success "Download complete"

    # Extract archive
    Write-Info "Extracting archive..."
    $extractPath = Join-Path $tempDir "extract"
    Expand-Archive -Path $archivePath -DestinationPath $extractPath -Force

    # Find binary
    $binaryPath = Get-ChildItem -Path $extractPath -Filter $BINARY_NAME -Recurse | Select-Object -First 1

    if (-not $binaryPath) {
        Write-Error-Custom "Binary not found in archive"
    }

    # Create install directory
    if (-not (Test-Path $installDir)) {
        Write-Info "Creating install directory: $installDir"
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    # Install binary
    $destPath = Join-Path $installDir $BINARY_NAME
    Write-Info "Installing to $destPath..."
    Copy-Item -Path $binaryPath.FullName -Destination $destPath -Force
    Write-Success "Installed $BINARY_NAME to $installDir"

    # Add to PATH if not already present
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")

    # Use exact matching by splitting PATH into entries
    $pathEntries = if ($currentPath) { $currentPath -split ';' | Where-Object { $_ -ne '' } } else { @() }

    if ($pathEntries -notcontains $installDir) {
        Write-Info "Adding $installDir to PATH..."
        # Safely join without leading/trailing semicolons
        $newPath = if ($currentPath) { "$currentPath;$installDir" } else { $installDir }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        # Update current session PATH so user can use it immediately
        $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")
        Write-Success "Added to PATH (restart other terminals to take effect)"
    } else {
        Write-Info "$installDir is already in PATH"
    }

    # Show config directory info
    $configDir = Join-Path $env:APPDATA "vibecoding"
    $settingsPath = Join-Path $configDir "settings.json"

    Write-Host ""
    Write-Success "Installation complete!"
    Write-Host ""
    Write-Host "  Install directory: $destPath" -ForegroundColor White
    Write-Host "  Config directory : $configDir" -ForegroundColor White
    Write-Host "    - Settings file: $settingsPath" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Version: $version" -ForegroundColor White
    Write-Host ""

    # Check if vibecoding is available
    $vibecodingPath = Get-Command vibecoding -ErrorAction SilentlyContinue
    if ($vibecodingPath) {
        Write-Host "  Get started:" -ForegroundColor White
        Write-Host "    vibecoding --help" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Uninstall:" -ForegroundColor White
        Write-Host "    irm https://gitee.com/startvibecoding/vibecoding/raw/main/install.ps1 | iex; Uninstall-VibeCoding" -ForegroundColor Gray
        Write-Host ""
    } else {
        Write-Warn "'vibecoding' is not found in your current PATH."
        Write-Host ""
        Write-Host "  To add it to your PATH manually:" -ForegroundColor White
        Write-Host ""
        Write-Host "  # PowerShell (current session):" -ForegroundColor Cyan
        Write-Host "    `$env:Path += `";$installDir`"" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  # PowerShell (permanent, current user):" -ForegroundColor Cyan
        Write-Host "    [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$installDir', 'User')" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  # CMD (permanent, current user):" -ForegroundColor Cyan
        Write-Host "    setx Path `"%Path%;$installDir`"" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  # Or add via System Settings > Environment Variables > User PATH" -ForegroundColor Cyan
        Write-Host ""
    }

} catch {
    Write-Error-Custom "Installation failed: $_"
} finally {
    # Cleanup
    if (Test-Path $tempDir) {
        Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
