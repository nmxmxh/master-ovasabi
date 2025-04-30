#!/bin/bash

# Script to find and assist in fixing broken links in documentation
# Usage: ./tools/fix-links.sh

echo "Running Markdown link checker..."

# Create results directory
RESULTS_DIR="docs/link-check-results"
mkdir -p "$RESULTS_DIR"

# Find all Markdown files
MARKDOWN_FILES=$(find docs -name "*.md")

# Output file for broken links
BROKEN_LINKS_FILE="$RESULTS_DIR/broken-links.txt"
> "$BROKEN_LINKS_FILE"

# Check links in each file
for file in $MARKDOWN_FILES; do
  echo "Checking $file..."
  
  # Run markdown-link-check and capture output
  RESULT=$(yarn markdown-link-check "$file" 2>&1)
  
  # Check if broken links were found
  if echo "$RESULT" | grep -q "ERROR:"; then
    echo "Broken links found in $file"
    echo -e "\n## $file" >> "$BROKEN_LINKS_FILE"
    echo "$RESULT" | grep -A 1 "\[✖\]" >> "$BROKEN_LINKS_FILE"
  fi
done

# Check if any broken links were found
if [ -s "$BROKEN_LINKS_FILE" ]; then
  echo -e "\nBroken links found. See $BROKEN_LINKS_FILE for details."
  echo "Common fixes:"
  echo "1. For local files, ensure the path is correct and the file exists"
  echo "2. For relative links, consider using '../' instead of absolute paths"
  echo "3. For GitHub links, ensure the repository URL is correct"
  echo "4. For external links, check if the URL is correctly formed and accessible"
else
  echo "No broken links found. Great job!"
  rm "$BROKEN_LINKS_FILE"
fi

# Check for missing diagrams
DIAGRAMS_DIR="docs/diagrams"
if [ ! -d "$DIAGRAMS_DIR" ]; then
  echo "Creating diagrams directory..."
  mkdir -p "$DIAGRAMS_DIR"
fi

# Check for missing image files
IMAGES_DIR="docs/assets/images"
if [ ! -d "$IMAGES_DIR" ]; then
  echo "Creating images directory..."
  mkdir -p "$IMAGES_DIR"
fi

echo "Link check completed!" 