# nSelf CLI Windows Installer
# Usage: irm install.nself.org/win.ps1 | iex
#
# Flags (set as environment variables before running):
#   $env:NSELF_VERSION   = "1.0.3"     # specific version (default: latest)
#   $env:NSELF_DIR       = "C:\custom"  # install directory
#   $env:NSELF_NO_TELEMETRY = "1"       # disable install telemetry

$ErrorActionPreference = "Stop"

$DefaultInstallDir = "$env:LOCALAPPDATA\nself\bin"
$GithubRepo = "nself-org/cli"
$BinaryName = "nself.exe"

function Get-LatestVersion {
    $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$GithubRepo/releases/latest" -Headers @{ "User-Agent" = "nself-installer" }
    return $response.tag_name
}

function Get-InstalledVersion {
    param([string]$BinPath)
    $exe = Join-Path $BinPath $BinaryName
    if (Test-Path $exe) {
        try {
            $output = & $exe version 2>&1
            if ($output -match "(\d+\.\d+\.\d+)") {
                return $Matches[1]
            }
        } catch {}
    }
    return $null
}

function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
    switch ($arch) {
        "x64"   { return "amd64" }
        "arm64" { return "arm64" }
        default {
            Write-Host "Unsupported architecture: $arch" -ForegroundColor Red
            exit 1
        }
    }
}

function Add-ToPath {
    param([string]$Dir)
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$Dir*") {
        [Environment]::SetEnvironmentVariable("Path", "$Dir;$currentPath", "User")
        $env:Path = "$Dir;$env:Path"
        Write-Host "Added $Dir to PATH" -ForegroundColor Green
    }
}

function Install-Nself {
    # Determine install directory.
    $installDir = if ($env:NSELF_DIR) { $env:NSELF_DIR } else { $DefaultInstallDir }

    # Determine version.
    $version = $env:NSELF_VERSION
    if (-not $version) {
        Write-Host "Fetching latest version..." -ForegroundColor Cyan
        $version = Get-LatestVersion
    }
    if ($version -notmatch "^v") {
        $version = "v$version"
    }
    Write-Host "Installing nSelf CLI $version" -ForegroundColor Cyan

    # Check for existing installation.
    $existing = Get-InstalledVersion -BinPath $installDir
    if ($existing) {
        $existingNorm = $existing.TrimStart("v")
        $targetNorm = $version.TrimStart("v")
        if ($existingNorm -eq $targetNorm) {
            Write-Host "nSelf CLI $version is already installed." -ForegroundColor Green
            return
        }
        Write-Host "Upgrading from $existing to $version" -ForegroundColor Yellow
        # Backup current binary.
        $backupPath = Join-Path $installDir "nself.prev.exe"
        $currentExe = Join-Path $installDir $BinaryName
        if (Test-Path $currentExe) {
            Copy-Item $currentExe $backupPath -Force
            Write-Host "Backup saved to $backupPath" -ForegroundColor DarkGray
        }
    }

    # Determine architecture.
    $arch = Get-Architecture

    # Build download URL.
    $tarball = "nself_windows_${arch}.tar.gz"
    $downloadUrl = "https://github.com/$GithubRepo/releases/download/$version/$tarball"
    $checksumUrl = "https://github.com/$GithubRepo/releases/download/$version/checksums.txt"

    # Create install directory.
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    # Download binary.
    $tempDir = Join-Path $env:TEMP "nself-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    $archivePath = Join-Path $tempDir $tarball

    Write-Host "Downloading $downloadUrl..." -ForegroundColor Cyan
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
    } catch {
        Write-Host "Download failed: $_" -ForegroundColor Red
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        exit 1
    }

    # Download and verify checksum.
    $checksumPath = Join-Path $tempDir "checksums.txt"
    try {
        Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumPath -UseBasicParsing
        $expectedHash = (Get-Content $checksumPath | Where-Object { $_ -like "*$tarball*" } | ForEach-Object { ($_ -split "\s+")[0] })
        if ($expectedHash) {
            $actualHash = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()
            if ($actualHash -ne $expectedHash.ToLower()) {
                Write-Host "Checksum mismatch! Expected: $expectedHash, Got: $actualHash" -ForegroundColor Red
                Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
                exit 1
            }
            Write-Host "Checksum verified." -ForegroundColor Green
        }
    } catch {
        Write-Host "Could not verify checksum (continuing): $_" -ForegroundColor Yellow
    }

    # Extract archive. PowerShell 5+ can handle .tar.gz via tar.exe (ships with Win10+).
    Write-Host "Extracting..." -ForegroundColor Cyan
    $extractDir = Join-Path $tempDir "extract"
    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    try {
        tar -xzf $archivePath -C $extractDir 2>$null
    } catch {
        Write-Host "Extraction failed. Ensure tar is available (Windows 10+)." -ForegroundColor Red
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        exit 1
    }

    # Find the binary in the extracted directory.
    $extractedBin = Get-ChildItem -Path $extractDir -Filter $BinaryName -Recurse | Select-Object -First 1
    if (-not $extractedBin) {
        Write-Host "Binary not found in archive." -ForegroundColor Red
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        exit 1
    }

    # Move binary to install dir.
    $destPath = Join-Path $installDir $BinaryName
    Copy-Item $extractedBin.FullName $destPath -Force

    # Clean up temp files.
    Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue

    # Add to PATH.
    Add-ToPath -Dir $installDir

    # Verify installation.
    try {
        $installedVersion = & $destPath version 2>&1
        Write-Host ""
        Write-Host "nSelf CLI installed successfully!" -ForegroundColor Green
        Write-Host "  Version:  $installedVersion" -ForegroundColor DarkGray
        Write-Host "  Location: $destPath" -ForegroundColor DarkGray
    } catch {
        Write-Host "Installation completed but verification failed." -ForegroundColor Yellow
        Write-Host "  Location: $destPath" -ForegroundColor DarkGray
    }

    # Check Docker Desktop.
    $docker = Get-Command docker -ErrorAction SilentlyContinue
    if (-not $docker) {
        Write-Host ""
        Write-Host "Docker Desktop is required but not found in PATH." -ForegroundColor Yellow
        Write-Host "Install from: https://docs.docker.com/desktop/install/windows-install/" -ForegroundColor DarkGray
    }

    # Anonymous install telemetry (opt-out).
    if (-not $env:NSELF_NO_TELEMETRY) {
        try {
            Invoke-WebRequest -Uri "https://ping.nself.org/telemetry/install" `
                -Method POST `
                -Body (@{ platform = "windows"; arch = $arch; version = $version } | ConvertTo-Json) `
                -ContentType "application/json" `
                -UseBasicParsing `
                -TimeoutSec 5 | Out-Null
        } catch {}
    }

    Write-Host ""
    Write-Host "Get started:" -ForegroundColor Cyan
    Write-Host "  nself init my-project"
    Write-Host "  cd my-project"
    Write-Host "  nself start"
    Write-Host ""
    Write-Host "Documentation: https://docs.nself.org" -ForegroundColor DarkGray
}

Install-Nself
