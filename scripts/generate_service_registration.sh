#!/bin/bash

# List of deprecated/legacy services to skip
DEPRECATED=("auth" "finance" "oldservice")

# Output file
OUT="service_registration.json"

echo "[" > "$OUT"
FIRST=1

for openapi in docs/services/*/*_openapi.json; do
  [ -e "$openapi" ] || continue
  service=$(basename "$openapi" | sed 's/_openapi.json//')
  # Skip deprecated services
  skip=0
  for dep in "${DEPRECATED[@]}"; do
    if [[ "$service" == "$dep" ]]; then
      skip=1
      break
    fi
  done
  [ "$skip" -eq 1 ] && continue

  # Optionally, you can extract capabilities from a README or set as a placeholder
  capabilities='["api", "metadata"]'

  # Output JSON object
  if [ $FIRST -eq 0 ]; then
    echo "," >> "$OUT"
  fi
  FIRST=0
  cat <<EOF >> "$OUT"
{
  "name": "${service^}Service",
  "capabilities": $capabilities,
  "schema": { "openapi": "$openapi" }
}
EOF
done

echo "]" >> "$OUT"
echo "Service registration data written to $OUT"