$ErrorActionPreference = "Stop"

if (Get-Command makensis -ErrorAction SilentlyContinue) {
    makensis /VERSION
    exit 0
}

$version = "3.12"
$expectedHash = "3BC2B06253A7E4957111BE152AC6A536E0C7478A706E19DA814038DB5D706495"
$installer = Join-Path $env:RUNNER_TEMP "nsis-$version-setup.exe"
$url = "https://pilotfiber.dl.sourceforge.net/project/nsis/NSIS%203/$version/nsis-$version-setup.exe"

curl.exe --fail --location --silent --show-error --output $installer $url
$actualHash = (Get-FileHash -Algorithm SHA256 $installer).Hash
if ($actualHash -ne $expectedHash) {
    throw "NSIS installer hash mismatch: expected $expectedHash, got $actualHash"
}

$process = Start-Process -FilePath $installer -ArgumentList "/S" -Wait -PassThru
if ($process.ExitCode -ne 0) {
    throw "NSIS installation failed with exit code $($process.ExitCode)"
}

$nsisPath = Join-Path ${env:ProgramFiles(x86)} "NSIS"
$nsisPath | Out-File -FilePath $env:GITHUB_PATH -Encoding utf8 -Append
$env:PATH = "$nsisPath;$env:PATH"
if (-not (Get-Command makensis -ErrorAction SilentlyContinue)) {
    throw "makensis was unavailable after installing NSIS $version"
}
makensis /VERSION
