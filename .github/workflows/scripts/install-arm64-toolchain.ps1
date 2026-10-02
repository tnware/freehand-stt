param(
    [string]$Destination = $env:RUNNER_TEMP,
    [ValidateSet("Arm64", "X64")]
    [string]$HostArchitecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
)

$ErrorActionPreference = "Stop"
if (-not $Destination) { throw "Specify a toolchain destination" }
$hostTarget = if ($HostArchitecture -eq "Arm64") { "aarch64" } else { "x86_64" }
$version = "20260922"
$name = "llvm-mingw-$version-ucrt-$hostTarget"
$expectedHash = if ($HostArchitecture -eq "Arm64") {
    "a317514a7a63badd692032c0c2b8e165f630bbaebbe7a3254348051f43a64949"
} else {
    "e3ad77d117a4bea19a7a3b333341824d79a5a371004a10e25b8504e7b3047666"
}
New-Item -ItemType Directory -Force -Path $Destination | Out-Null
$archive = Join-Path $Destination "$name.zip"
curl.exe --fail --location --silent --show-error --output $archive "https://github.com/mstorsjo/llvm-mingw/releases/download/$version/$name.zip"
if ($LASTEXITCODE -ne 0) { throw "LLVM-MinGW download failed" }
if ((Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant() -ne $expectedHash) {
    throw "LLVM-MinGW archive checksum mismatch"
}
Expand-Archive -LiteralPath $archive -DestinationPath $Destination -Force
$toolchainBin = Join-Path $Destination "$name/bin"
$env:CC = Join-Path $toolchainBin "aarch64-w64-mingw32-clang.exe"
$env:CXX = Join-Path $toolchainBin "aarch64-w64-mingw32-clang++.exe"
$env:CGO_ENABLED = "1"
$env:PATH = "$toolchainBin;$env:PATH"
if ($env:GITHUB_PATH) { $toolchainBin | Out-File -FilePath $env:GITHUB_PATH -Encoding utf8 -Append }
if ($env:GITHUB_ENV) {
    "CC=$env:CC", "CXX=$env:CXX", "CGO_ENABLED=1" | Out-File -FilePath $env:GITHUB_ENV -Encoding utf8 -Append
}
& $env:CC --version
if ($LASTEXITCODE -ne 0) { throw "LLVM-MinGW compiler is unavailable" }
