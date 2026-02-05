$ErrorActionPreference = 'Stop'

$packageName = 'ssh-x-term'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# Remove shim
Uninstall-BinFile -Name 'sxt' -Path (Join-Path $toolsDir 'sxt.exe')
