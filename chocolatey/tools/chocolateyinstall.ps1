$ErrorActionPreference = 'Stop'

$packageName = 'ssh-x-term'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$version = '2.0.3'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v$version/ssh-x-term-windows-amd64.exe"
  checksum64     = 'PLACEHOLDER_CHECKSUM'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# Rename the exe to sxt.exe for consistency
$exePath = Join-Path $toolsDir 'ssh-x-term-windows-amd64.exe'
$targetPath = Join-Path $toolsDir 'sxt.exe'

if (Test-Path $exePath) {
  Move-Item -Path $exePath -Destination $targetPath -Force
}

# Create shim
Install-BinFile -Name 'sxt' -Path $targetPath
