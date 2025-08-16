# create-windows-installer.ps1 - Create Windows installation scripts for nself
# Usage: powershell -ExecutionPolicy Bypass -File scripts/create-windows-installer.ps1 <version>

param(
    [Parameter(Mandatory=$true)]
    [string]$Version
)

$ErrorActionPreference = "Stop"

Write-Host "Creating Windows installers for nself $Version" -ForegroundColor Cyan

# Create install.ps1 for direct PowerShell installation
$installScript = @'
# nself Windows Installer
# This script installs nself on Windows using WSL2 or Git Bash

param(
    [string]$Version = "VERSION_PLACEHOLDER",
    [string]$InstallDir = "$env:LOCALAPPDATA\nself",
    [switch]$WSL = $false
)

$ErrorActionPreference = "Stop"

function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

Write-Host "`n=====================================" -ForegroundColor Cyan
Write-Host "       nself Installer for Windows    " -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# Check for Docker Desktop
Write-Host "Checking requirements..." -ForegroundColor Yellow
$dockerInstalled = Get-Command docker -ErrorAction SilentlyContinue

if (-not $dockerInstalled) {
    Write-Host "Docker Desktop is required but not found!" -ForegroundColor Red
    Write-Host "Please install Docker Desktop from: https://www.docker.com/products/docker-desktop" -ForegroundColor Yellow
    
    $install = Read-Host "Would you like to download Docker Desktop now? (y/n)"
    if ($install -eq 'y') {
        Start-Process "https://www.docker.com/products/docker-desktop"
        Write-Host "Please install Docker Desktop and restart this installer." -ForegroundColor Yellow
        exit 1
    }
    exit 1
}

Write-Host "✓ Docker Desktop found" -ForegroundColor Green

# Option 1: WSL Installation (Recommended)
if ($WSL -or (Get-Command wsl -ErrorAction SilentlyContinue)) {
    Write-Host "`nWSL detected. Installing nself in WSL (Recommended)" -ForegroundColor Green
    
    # Check WSL version
    $wslVersion = wsl --list --verbose 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Installing WSL2..." -ForegroundColor Yellow
        wsl --install
        Write-Host "WSL2 installed. Please restart your computer and run this installer again." -ForegroundColor Yellow
        exit 0
    }
    
    # Install in WSL
    Write-Host "Installing nself in WSL..." -ForegroundColor Yellow
    wsl -e bash -c "curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash"
    
    # Create Windows wrapper batch file
    $wrapperPath = "$InstallDir\nself.cmd"
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    
    @"
@echo off
wsl -e nself %*
"@ | Set-Content -Path $wrapperPath
    
    Write-Host "✓ nself installed in WSL" -ForegroundColor Green
    
} else {
    # Option 2: Git Bash Installation (Fallback)
    Write-Host "`nInstalling nself for Git Bash..." -ForegroundColor Yellow
    
    # Check for Git Bash
    $gitBash = Get-Command git -ErrorAction SilentlyContinue
    if (-not $gitBash) {
        Write-Host "Git for Windows is required!" -ForegroundColor Red
        Write-Host "Download from: https://git-scm.com/download/win" -ForegroundColor Yellow
        
        $install = Read-Host "Would you like to download Git for Windows now? (y/n)"
        if ($install -eq 'y') {
            Start-Process "https://git-scm.com/download/win"
            Write-Host "Please install Git for Windows and restart this installer." -ForegroundColor Yellow
            exit 1
        }
        exit 1
    }
    
    # Download nself
    Write-Host "Downloading nself $Version..." -ForegroundColor Yellow
    $zipUrl = "https://github.com/acamarata/nself/archive/refs/tags/$Version.zip"
    $zipPath = "$env:TEMP\nself.zip"
    
    Invoke-WebRequest -Uri $zipUrl -OutFile $zipPath
    
    # Extract
    Write-Host "Extracting files..." -ForegroundColor Yellow
    Expand-Archive -Path $zipPath -DestinationPath "$env:TEMP" -Force
    
    # Move to installation directory
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Move-Item -Path "$env:TEMP\nself-*\*" -Destination $InstallDir -Force
    
    # Create wrapper batch file
    $wrapperPath = "$InstallDir\nself.cmd"
    @"
@echo off
setlocal
set NSELF_HOME=$InstallDir
set PATH=%NSELF_HOME%\bin;%PATH%
bash "%NSELF_HOME%\bin\nself" %*
endlocal
"@ | Set-Content -Path $wrapperPath
    
    Write-Host "✓ nself installed for Git Bash" -ForegroundColor Green
}

# Add to PATH
Write-Host "`nAdding nself to PATH..." -ForegroundColor Yellow
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
    Write-Host "✓ Added to PATH" -ForegroundColor Green
} else {
    Write-Host "✓ Already in PATH" -ForegroundColor Green
}

# Create Start Menu shortcut
$startMenuPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs"
$shortcutPath = "$startMenuPath\nself.lnk"

$WScriptShell = New-Object -ComObject WScript.Shell
$shortcut = $WScriptShell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = "cmd.exe"
$shortcut.Arguments = "/k `"$wrapperPath help`""
$shortcut.WorkingDirectory = "%USERPROFILE%"
$shortcut.IconLocation = "cmd.exe"
$shortcut.Description = "nself - Self-hosted infrastructure manager"
$shortcut.Save()

Write-Host "✓ Created Start Menu shortcut" -ForegroundColor Green

Write-Host "`n=====================================" -ForegroundColor Green
Write-Host "    nself Installation Complete!      " -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Green
Write-Host ""
Write-Host "Please restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
Write-Host ""
Write-Host "To get started:" -ForegroundColor Cyan
Write-Host "  1. Open a new terminal (CMD, PowerShell, or Windows Terminal)" -ForegroundColor White
Write-Host "  2. Run: nself help" -ForegroundColor White
Write-Host "  3. Create a project: mkdir my-project && cd my-project" -ForegroundColor White
Write-Host "  4. Initialize: nself init" -ForegroundColor White
Write-Host ""
Write-Host "Documentation: https://github.com/acamarata/nself" -ForegroundColor Cyan
'@

# Replace version placeholder
$installScript = $installScript -replace 'VERSION_PLACEHOLDER', $Version

# Save install.ps1
$installScript | Set-Content -Path "install-windows.ps1" -Encoding UTF8
Write-Host "✓ Created install-windows.ps1" -ForegroundColor Green

# Create Scoop manifest
$scoopManifest = @"
{
    "version": "$($Version -replace '^v', '')",
    "description": "Self-hosted infrastructure manager for developers",
    "homepage": "https://nself.org",
    "license": "MIT",
    "notes": "nself requires Docker Desktop to be installed and running.",
    "url": "https://github.com/acamarata/nself/archive/refs/tags/$Version.zip",
    "extract_dir": "nself-$($Version -replace '^v', '')",
    "bin": [
        [
            "bin\\nself",
            "nself",
            "bash"
        ]
    ],
    "post_install": [
        "if (!(Test-Path '`$env:DOCKER_HOST')) {",
        "    Write-Host 'Docker Desktop is required. Please install from: https://www.docker.com/products/docker-desktop' -ForegroundColor Yellow",
        "}"
    ],
    "checkver": {
        "github": "https://github.com/acamarata/nself"
    },
    "autoupdate": {
        "url": "https://github.com/acamarata/nself/archive/refs/tags/v`$version.zip",
        "extract_dir": "nself-`$version"
    }
}
"@

# Save Scoop manifest
$scoopManifest | Set-Content -Path "nself.json" -Encoding UTF8
Write-Host "✓ Created nself.json (Scoop manifest)" -ForegroundColor Green

# Create Chocolatey package files
$chocoNuspec = @"
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>nself</id>
    <version>$($Version -replace '^v', '')</version>
    <title>nself</title>
    <authors>acamarata</authors>
    <owners>acamarata</owners>
    <licenseUrl>https://github.com/acamarata/nself/blob/main/LICENSE</licenseUrl>
    <projectUrl>https://nself.org</projectUrl>
    <iconUrl>https://raw.githubusercontent.com/acamarata/nself/main/assets/logo.png</iconUrl>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <description>Self-hosted infrastructure manager for developers. Deploy and manage your own backend infrastructure with Docker.</description>
    <summary>Self-hosted infrastructure manager</summary>
    <releaseNotes>https://github.com/acamarata/nself/releases/tag/$Version</releaseNotes>
    <tags>docker infrastructure devops self-hosted backend</tags>
    <dependencies>
      <dependency id="docker-desktop" version="2.0.0.0" />
    </dependencies>
  </metadata>
  <files>
    <file src="tools\**" target="tools" />
  </files>
</package>
"@

# Create Chocolatey tools directory
New-Item -ItemType Directory -Force -Path "chocolatey\tools" | Out-Null

# Chocolatey install script
$chocoInstall = @'
$ErrorActionPreference = 'Stop'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url = 'https://github.com/acamarata/nself/archive/refs/tags/VERSION_PLACEHOLDER.zip'

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url           = $url
  checksum      = 'CHECKSUM_PLACEHOLDER'
  checksumType  = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

$nselfPath = Join-Path $toolsDir "nself-VERSION_NUM_PLACEHOLDER"
Install-ChocolateyPath $nselfPath 'User'

Write-Host "nself has been installed successfully!" -ForegroundColor Green
Write-Host "Please restart your terminal to use nself." -ForegroundColor Yellow
'@

# Replace placeholders
$versionNum = $Version -replace '^v', ''
$chocoInstall = $chocoInstall -replace 'VERSION_PLACEHOLDER', $Version
$chocoInstall = $chocoInstall -replace 'VERSION_NUM_PLACEHOLDER', $versionNum

# Save Chocolatey files
$chocoNuspec | Set-Content -Path "chocolatey\nself.nuspec" -Encoding UTF8
$chocoInstall | Set-Content -Path "chocolatey\tools\chocolateyinstall.ps1" -Encoding UTF8
Write-Host "✓ Created Chocolatey package files" -ForegroundColor Green

Write-Host "`n=====================================" -ForegroundColor Green
Write-Host "  Windows installers created!" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Green
Write-Host ""
Write-Host "Files created:" -ForegroundColor Cyan
Write-Host "  • install-windows.ps1 - Direct PowerShell installer" -ForegroundColor White
Write-Host "  • nself.json - Scoop manifest" -ForegroundColor White
Write-Host "  • chocolatey/ - Chocolatey package" -ForegroundColor White
Write-Host ""
Write-Host "Distribution:" -ForegroundColor Cyan
Write-Host "  PowerShell: irm https://your-domain/install-windows.ps1 | iex" -ForegroundColor White
Write-Host "  Scoop: Submit to scoop-extras bucket" -ForegroundColor White
Write-Host "  Chocolatey: choco pack && choco push" -ForegroundColor White