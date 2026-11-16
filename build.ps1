# Deprecated: builds the images and installs the compiled binary via
# `cli-tools build`. Kept until the busybox-launcher migration cutover
# (issue #41); run `cli-tools build` directly instead.
param(
    [switch]$ScriptsOnly,
    [switch]$PruneImages,
    [Parameter(ValueFromRemainingArguments = $true)][string[]]$Tools
)
$ErrorActionPreference = "Stop"
if (-not (Get-Command cli-tools -ErrorAction SilentlyContinue)) {
    Write-Error "cli-tools is not installed. Build and install it first: go build -o `"$env:LOCALAPPDATA\Programs\bin\cli-tools.exe`" ./cmd/cli-tools ; then run: cli-tools build"
}
$CliArgs = @("build")
if ($ScriptsOnly) { $CliArgs += "--launcher-only" }
if ($PruneImages) { $CliArgs += "--prune-images" }
if ($Tools) { $CliArgs += $Tools }
& cli-tools @CliArgs
exit $LASTEXITCODE
