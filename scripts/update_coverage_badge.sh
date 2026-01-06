#!/usr/bin/env bash
set -euo pipefail

echo "Running go test to produce a unified coverage profile..."
go test -coverpkg=./... ./... -coverprofile=coverage.out || true
# Some platforms (PowerShell or other shells) may produce a file named 'coverage'.
# Normalize that to 'coverage.out' before processing so filtering works uniformly.
if [[ -f coverage && ! -f coverage.out ]]; then
  echo "Normalizing coverage -> coverage.out"
  mv coverage coverage.out || true
fi

# Exclude test utility packages from coverage metrics (not part of product surface).
if [[ -f coverage.out ]]; then
  head -n1 coverage.out > coverage.filtered.out || true
  tail -n +2 coverage.out | grep -v -E '(^github.com/toeirei/keymaster/internal/testutil|testdata)' >> coverage.filtered.out || true
  mv coverage.filtered.out coverage.out || true
else
  echo "coverage.out not found" >&2
  exit 1
fi

pct=$(go tool cover -func=coverage.out | awk '/total:/ {print $3}')
if [[ -z "$pct" ]]; then
  echo "Failed to parse coverage percent" >&2
  exit 1
fi

pctnum=${pct%%%}
width=$(awk -v p="$pctnum" 'BEGIN{printf("%d", 200*p/100)}')

cat > coverage.svg <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="200" height="20">
  <rect width="200" height="20" fill="#555"/>
  <rect width="$width" height="20" x="0" y="0" fill="#4c1"/>
  <text x="100" y="14" font-family="Verdana" font-size="11" fill="#fff" text-anchor="middle">coverage $pct</text>
</svg>
EOF

echo "Wrote coverage.svg with $pct"
