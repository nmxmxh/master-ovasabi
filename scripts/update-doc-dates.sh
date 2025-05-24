#!/bin/bash
# Auto-updated by update-doc-dates.sh

set -e
CURRENT_DATE=$(date +%Y-%m-%d)
TRAINING_DATE="2024-06"
SYSTEM_DATE_REQUEST_BLOCK="> **System Date Request:** Run \
>   `date +%Y-%m-%d` \
> to get the current system date dynamically."
CONTEXT_BLOCK="> **System Date:** $CURRENT_DATE  \
> **AI Training Data Cutoff:** $TRAINING_DATE  \
>  \
> **System Date Request:** Run \
>   `date +%Y-%m-%d` \
> to get the current system date dynamically.  \
>  \
> **Information Quality & Traceability Guidance:**  \
> - All documentation, OpenAPI specs, and standards should include a version or date field reflecting the last update or generation date.  \
> - The current system date is used for traceability and should be referenced in all new or updated files.  \
> - The AI assistant's knowledge is current up to the training data cutoff; for anything after $TRAINING_DATE, always verify with the latest platform docs, code, or authoritative sources.  \
> - When in doubt, prefer the most recent file date, version field, or explicit changelog entry in service documentation.  \
> - Reference this section in onboarding and developer docs as the source of truth for date/version context and information freshness."

update_file() {
  local file="$1"
  # Remove any existing context block or version/date field at the top
  awk 'NR==1 && /^> \*\*System Date:/ {next} NR==2 && /^> \*\*AI Training Data Cutoff:/ {next} NR==3 && /^>  \\$/ {next} NR==4 && /^> \*\*System Date Request:/ {next} NR==5 && /^>   `date \+%Y-%m-%d`/ {next} NR==6 && /^>  \\$/ {next} NR==7 && /^> \*\*Information Quality/ {next} NR==8 && /^> - All documentation/ {next} NR==9 && /^> - The current system date/ {next} NR==10 && /^> - The AI assistant/ {next} NR==11 && /^> - When in doubt/ {next} NR==12 && /^> - Reference this section/ {next} NR==1 && /^version:/ {next} NR==1 && /^date:/ {next} {print}' "$file" > "$file.tmp"
  # Insert context block and version/date at the top
  if [[ "$file" == *.md ]]; then
    # Only insert a heading if the first line is not a heading
    if ! head -1 "$file.tmp" | grep -q '^# '; then
      echo -e "# Documentation\n" | cat - "$file.tmp" > "$file"
    else
      mv "$file.tmp" "$file"
    fi
  elif [[ "$file" == *.yaml ]]; then
    # Only insert schema directive and a two-line comment block if not present, and never insert more than that
    SCHEMA_LINE="# yaml-language-server: $schema=https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/schemas/v3.0/schema.json"
    if ! head -3 "$file.tmp" | grep -q "$SCHEMA_LINE"; then
      echo -e "$SCHEMA_LINE\n# Auto-updated by update-doc-dates.sh\n# version: $CURRENT_DATE" | cat - "$file.tmp" > "$file"
    else
      mv "$file.tmp" "$file"
    fi
  fi
  rm "$file.tmp"
}

export -f update_file
export CURRENT_DATE
export TRAINING_DATE
export CONTEXT_BLOCK

find docs/services docs/amadeus docs/architecture api/protos amadeus/backups amadeus docs/generated/amadeus/backups -type f \( -name '*.md' -o -name '*.yaml' -o -name '*.proto' -o -name '*.json' -o -name '*.backup' \) -exec bash -c 'update_file "$0"' {} \;

echo "All docs and OpenAPI specs updated with current date and context block ($CURRENT_DATE)." 