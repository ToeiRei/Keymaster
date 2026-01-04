param(
    [switch]$Commit = $false
)

$ErrorActionPreference = 'Stop'
Write-Output "Running go test to produce coverage profile (this may take a while)..."
& go test -coverpkg=./... ./... -coverprofile=coverage.out

if (Test-Path -Path './coverage' -PathType Leaf -and -not (Test-Path -Path './coverage.out')) {
    Write-Output "Normalizing 'coverage' -> 'coverage.out'"
    Move-Item -Path './coverage' -Destination './coverage.out' -Force
}

if (-not (Test-Path -Path './coverage.out')) {
    Write-Error "coverage.out not found; aborting"
    exit 1
}

$func = & go tool cover -func=coverage.out
$line = $func | Select-String 'total:' | Select-Object -First 1
if (-not $line) {
    Write-Error "failed to parse coverage output"
    exit 1
}

$percent = ($line -split '\s+')[-1]
$percentNum = [double]($percent.TrimEnd('%'))
$width = [int](200 * $percentNum / 100)

$svg = @"
<svg xmlns="http://www.w3.org/2000/svg" width="200" height="20">
  <rect width="200" height="20" fill="#555"/>
  <rect width="$width" height="20" x="0" y="0" fill="#4c1"/>
  <text x="100" y="14" font-family="Verdana" font-size="11" fill="#fff" text-anchor="middle">coverage $percent</text>
</svg>
"@

$svg | Out-File -FilePath ./coverage.svg -Encoding utf8
Write-Output "Wrote ./coverage.svg with $percent coverage"

if ($Commit) {
    Write-Output "Committing coverage.svg to git (Commit switch enabled)"
    git add coverage.svg
    if (-not (git diff --cached --quiet)) {
        git commit -m "ci: update coverage badge"
        git push origin HEAD:main
    } else {
        Write-Output "No changes to commit"
    }
}
