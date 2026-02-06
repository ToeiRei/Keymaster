param(
    [switch]$Commit = $false
)

$ErrorActionPreference = 'Stop'
Write-Output "Running go test to produce coverage profile (this may take a while)..."
$testExitCode = 0
try {
    $previousEap = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & go test -coverpkg=./... ./... -coverprofile=coverage.out
    $testExitCode = $LASTEXITCODE
} finally {
    $ErrorActionPreference = $previousEap
}
if ($testExitCode -ne 0) {
    Write-Warning "go test exited with code $testExitCode; continuing to process coverage if available."
}

if ((Test-Path -Path './coverage' -PathType Leaf) -and -not (Test-Path -Path './coverage.out' -PathType Leaf)) {
    Write-Output "Normalizing 'coverage' -> 'coverage.out'"
    Move-Item -Path './coverage' -Destination './coverage.out' -Force
}

if (-not (Test-Path -Path './coverage.out')) {
    Write-Error "coverage.out file not found; aborting"
    exit 1
}

# Exclude test utility packages and legacy TUI from coverage metrics (not part of product surface).
$first = Get-Content -Path './coverage.out' -TotalCount 1
$rest = Get-Content -Path './coverage.out' | Select-Object -Skip 1
# Exclude internal test helper packages, legacy TUI, and any testdata paths from coverage
$filtered = $rest | Where-Object { ($_ -notmatch '^github.com/toeirei/keymaster/testutil') -and ($_ -notmatch '^github.com/toeirei/keymaster/tui') -and ($_ -notmatch 'testdata') }

# Drop entries that reference missing files (prevents go tool cover parse errors)
$prefix = 'github.com/toeirei/keymaster/'
$root = (Get-Location).Path
$filtered = $filtered | Where-Object {
    $line = $_
    $filePart = ($line -split ':', 2)[0]
    if ($filePart.StartsWith($prefix)) {
        $rel = $filePart.Substring($prefix.Length)
        $local = Join-Path $root $rel
        return (Test-Path -Path $local -PathType Leaf)
    }
    # Keep any non-module-path entries (defensive)
    return $true
}
# Write filtered coverage without a UTF-8 BOM so go tool cover can read it.
$outLines = @()
$outLines += $first
if ($filtered) { $outLines += $filtered }
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllLines((Join-Path (Get-Location) 'coverage.out'), $outLines, $utf8NoBom)

$func = & go tool cover -func ./coverage.out
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

# Write SVG without BOM
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText((Join-Path (Get-Location) 'coverage.svg'), $svg, $utf8NoBom)
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
