#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="$root_dir/docs-site/docs/generated/ts"

mkdir -p "$out_dir"

cd "$root_dir/web"
typedoc \
  --out "$out_dir" \
  --tsconfig "./tsconfig.json" \
  --entryPoints "src" \
  --entryPointStrategy "expand"

echo "Generated TS docs in $out_dir"
