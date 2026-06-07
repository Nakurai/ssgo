#!/usr/bin/env bash
# Package each template under templates/ into a distributable .zip archive.
# Output lands alongside each template at templates/<name>.zip.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
src_dir="$repo_root/templates"
out_dir="$src_dir"

if [ ! -d "$src_dir" ]; then
  echo "no templates/ directory found at $src_dir" >&2
  exit 1
fi

found=0
for dir in "$src_dir"/*/; do
  [ -d "$dir" ] || continue
  name="$(basename "$dir")"

  if [ ! -f "$dir/base.html" ]; then
    echo "skipping $name: missing base.html" >&2
    continue
  fi

  archive="$out_dir/$name.zip"
  rm -f "$archive"
  # Zip with the template name as the top-level folder; the CLI strips the
  # common top-level directory on extraction.
  ( cd "$src_dir" && zip -rq "$archive" "$name" )
  echo "packaged $name -> ${archive#"$repo_root"/}"
  found=$((found + 1))
done

echo "done ($found template(s))"
