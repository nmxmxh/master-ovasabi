# Find all direct metadata field accesses (for review)
grep -rnE 'meta\\[\"[a-zA-Z0-9_]+\"\\]' ./internal ./pkg ./cmd

# Find all custom metadata helpers
grep -rnE 'func (.*)Set.*Meta|func (.*)Update.*Meta' ./internal ./pkg

# Find all metadata initializations
grep -rnE 'make\\(map\\[string\\]interface\\{\\}\\)|map\\[string\\]interface\\{\\}' ./internal ./pkg