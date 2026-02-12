$ErrorActionPreference = 'Stop'

$packageName = 'ssh-x-term'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$version = '2.0.3'

$packageArgs = @{
  packageName    = $packageName
  fileFullPath   = Join-Path $toolsDir 'sxt.exe'
  url64bit       = "https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v$version/ssh-x-term-windows-amd64.exe"
  checksum64     = 'PLACEHOLDER_CHECKSUM'
  checksumType64 = 'sha256'
}

Get-ChocolateyWebFile @packageArgs
