#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="$root_dir/docs-site/docs/generated/go"

mkdir -p "$out_dir"

packages=(
  "ElMakina/backend/engine"
  "ElMakina/backend/server"
  "ElMakina/backend/models"
  "ElMakina/backend/client"
)

for pkg in "${packages[@]}"; do
  gomarkdoc -o "$out_dir/$(basename "$pkg").md" "$pkg"
done

echo "Generated Go docs in $out_dir"
