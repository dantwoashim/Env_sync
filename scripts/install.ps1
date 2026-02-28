# EnvSync installer for Windows (PowerShell)
# Usage: irm https://envsync.dev/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$Repo = "envsync/envsync"
$InstallDir = "$env:LOCALAPPDATA\EnvSync\bin"

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { $Arch = "arm64" }

Write-Host "  ✦ Installing EnvSync for windows/$Arch" -ForegroundColor Magenta

# Get latest version
try {
    $Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name -replace '^v', ''
} catch {
    Write-Host "  ✗ Failed to get latest version" -ForegroundColor Red
    exit 1
}

Write-Host "  ▸ Version: v$Version"

# Download
$Filename = "envsync_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/v$Version/$Filename"
$TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }

Write-Host "  ▸ Downloading $Filename..."
Invoke-WebRequest -Uri $Url -OutFile "$TempDir\$Filename"

# Extract
Expand-Archive -Path "$TempDir\$Filename" -DestinationPath $TempDir -Force

# Install
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
Move-Item "$TempDir\envsync.exe" "$InstallDir\envsync.exe" -Force

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
    Write-Host "  ▸ Added $InstallDir to PATH"
}

# Cleanup
Remove-Item $TempDir -Recurse -Force

Write-Host "  ✓ Installed envsync v$Version to $InstallDir\envsync.exe" -ForegroundColor Green
Write-Host ""
Write-Host "  Get started:"
Write-Host "    envsync init"
Write-Host ""
