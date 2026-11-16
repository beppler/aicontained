# Cross-compiles the cli-tools binary for every supported platform.
# Pure Go cross-compilation (CGO disabled) — no platform toolchains needed.
# Output lands in dist/, gitignored.
# Windows equivalent of cross-build.sh.
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')
New-Item -ItemType Directory -Force dist | Out-Null

foreach ($platform in 'windows/amd64','linux/amd64','linux/arm64','darwin/amd64','darwin/arm64') {
    $os, $arch = $platform -split '/'
    $out = "dist/cli-tools-$os-$arch"
    if ($os -eq 'windows') { $out += '.exe' }
    $env:CGO_ENABLED = '0'
    $env:GOOS = $os
    $env:GOARCH = $arch
    go build -o $out ./cmd/cli-tools
    if ($LASTEXITCODE -ne 0) { throw "go build failed for $platform" }
}

Get-ChildItem dist
