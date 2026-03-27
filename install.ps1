param(
  [switch]$Force,
  [switch]$NonInteractive
)

$ErrorActionPreference = "Stop"

$RepoDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$VersionFile = Join-Path $RepoDir "VERSION"
$Version = "unknown"
if (Test-Path $VersionFile) {
  $Version = (Get-Content $VersionFile -ErrorAction SilentlyContinue | Select-Object -First 1)
  if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = "unknown"
  }
}

function Resolve-Go {
  $command = Get-Command go -ErrorAction SilentlyContinue
  if ($command) {
    return $command.Source
  }

  $candidates = @(
    "C:\Program Files\Go\bin\go.exe",
    "$env:USERPROFILE\scoop\apps\go\current\bin\go.exe"
  )

  foreach ($candidate in $candidates) {
    if ($candidate -and (Test-Path $candidate)) {
      return $candidate
    }
  }

  throw "Failed to locate Go toolchain"
}

function Resolve-RepoCli {
  if (-not [string]::IsNullOrWhiteSpace($env:AGENT47_REPO_CLI) -and (Test-Path $env:AGENT47_REPO_CLI)) {
    return $env:AGENT47_REPO_CLI
  }

  return $null
}

function Resolve-ExplicitLauncher {
  param(
    [string]$EnvName
  )

  $candidate = [Environment]::GetEnvironmentVariable($EnvName)
  if ([string]::IsNullOrWhiteSpace($candidate)) {
    return $null
  }
  if (-not (Test-Path $candidate)) {
    throw "$EnvName points to a missing path: $candidate"
  }
  if (-not (Test-Path $candidate -PathType Leaf)) {
    throw "$EnvName must point to a file: $candidate"
  }
  return $candidate
}

function Ensure-UserPath {
  param(
    [string]$ManagedBin
  )

  $currentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
  $segments = @()
  if (-not [string]::IsNullOrWhiteSpace($currentUserPath)) {
    $segments = $currentUserPath.Split(';') | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
  }

  $normalizedManagedBin = [IO.Path]::GetFullPath($ManagedBin).TrimEnd('\')
  $alreadyPresent = $false
  foreach ($segment in $segments) {
    try {
      if ([IO.Path]::GetFullPath($segment).TrimEnd('\') -eq $normalizedManagedBin) {
        $alreadyPresent = $true
        break
      }
    } catch {
      Write-Host "[WARN] Skipping invalid user PATH segment: $segment"
    }
  }

  if ($alreadyPresent) {
    Write-Host "[OK] managed bin already present in user PATH"
    return
  }

  $newSegments = @($segments + $ManagedBin)
  [Environment]::SetEnvironmentVariable("Path", ($newSegments -join ';'), "User")
  if ($env:Path) {
    $env:Path = "$ManagedBin;$env:Path"
  } else {
    $env:Path = $ManagedBin
  }
  Write-Host "[OK] Added managed bin to user PATH"
}

if (-not $env:GOCACHE) {
  $env:GOCACHE = Join-Path $env:TEMP "agent47-go-build-cache"
}
if (-not $env:AGENT47_TEMPLATE_SOURCE) {
  $env:AGENT47_TEMPLATE_SOURCE = "filesystem"
}
if (-not $env:AGENT47_REPO_ROOT) {
  $env:AGENT47_REPO_ROOT = $RepoDir
}
$env:AGENT47_CALLER_DIR = (Get-Location).Path
$env:AGENT47_SKIP_WINDOWS_POSTINSTALL_PATH_HINT = "true"

Write-Host "[ AGENT47 v$Version ]"
Write-Host "[*] Installing agent47..."

$ManagedCli = Resolve-ExplicitLauncher -EnvName "AGENT47_GO_CLI"
$RepoCli = Resolve-ExplicitLauncher -EnvName "AGENT47_REPO_CLI"

if (-not [string]::IsNullOrWhiteSpace($ManagedCli)) {
  $Launcher = $ManagedCli
  $InstallArgs = @("__agent47_internal_install")
} elseif (-not [string]::IsNullOrWhiteSpace($RepoCli)) {
  $Launcher = $RepoCli
  $InstallArgs = @("__agent47_internal_install")
} else {
  $GoBin = Resolve-Go
  $Launcher = $GoBin
  $InstallArgs = @("run", ".\cmd\afs", "__agent47_internal_install")
}
if ($Force) {
  $InstallArgs += "--force"
}
if ($NonInteractive) {
  $InstallArgs += "--non-interactive"
}

Push-Location $RepoDir
try {
  & $Launcher @InstallArgs
  if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
  }
} finally {
  Pop-Location
}

$Agent47Home = $env:AGENT47_HOME
if ([string]::IsNullOrWhiteSpace($Agent47Home)) {
  $LocalAppData = $env:LOCALAPPDATA
  if ([string]::IsNullOrWhiteSpace($LocalAppData)) {
    $LocalAppData = $HOME
  }
  $Agent47Home = Join-Path $LocalAppData "agent47"
}
$ManagedBin = Join-Path $Agent47Home "bin"

Ensure-UserPath -ManagedBin $ManagedBin
