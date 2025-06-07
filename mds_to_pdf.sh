#!/bin/bash

# Converts all .md files in the mds/ directory to PDF, stores them in pdf/, appends the date, and generates SHA256 checksums.

set -e

DATE="2025-06-04"
MDS_DIR="mds"
PDF_DIR="pdf"

# Ensure output directory exists
mkdir -p "$PDF_DIR"

# Check for pandoc
if ! command -v pandoc >/dev/null 2>&1; then
  echo "Error: pandoc is not installed. Please install it first."
  exit 1
fi

# Optional: Check for a LaTeX engine (for best PDF output)
if ! command -v xelatex >/dev/null 2>&1 && ! command -v pdflatex >/dev/null 2>&1; then
  echo "Warning: No LaTeX engine found (xelatex or pdflatex). Pandoc PDF output may be limited."
fi

for mdfile in "$MDS_DIR"/*.md; do
  [ -e "$mdfile" ] || continue
  base=$(basename "$mdfile" .md)
  pdffile="$PDF_DIR/${base}_${DATE}.pdf"
  echo "Converting $mdfile → $pdffile"
  pandoc "$mdfile" -o "$pdffile"
  # Cryptographic enhancement: generate SHA256 checksum
  shasum -a 256 "$pdffile" > "$pdffile.sha256"
done

echo "All markdown files in $MDS_DIR have been converted to PDF in $PDF_DIR with cryptographic checksums."