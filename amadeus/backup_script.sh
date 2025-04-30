#!/bin/bash

# Amadeus Backup Script
# This script creates backups of the Amadeus knowledge graph and modified code files

# Create timestamp for backup naming
TIMESTAMP=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="amadeus/backups/${TIMESTAMP}"

# Create backup directories
mkdir -p "$BACKUP_DIR/src"

echo "Creating Amadeus backup with timestamp: $TIMESTAMP"

# Backup the knowledge graph data file if it exists
if [ -f "amadeus/knowledge_graph.json" ]; then
    echo "Backing up knowledge graph..."
    cp "amadeus/knowledge_graph.json" "$BACKUP_DIR/knowledge_graph.json"
fi

# Backup the recently modified files
echo "Backing up modified source files..."

# CLI tool
mkdir -p "$BACKUP_DIR/src/cmd/kgcli"
cp "amadeus/cmd/kgcli/main.go" "$BACKUP_DIR/src/cmd/kgcli/"

# Knowledge Graph API
mkdir -p "$BACKUP_DIR/src/pkg/kg"
cp "amadeus/pkg/kg/knowledge_graph.go" "$BACKUP_DIR/src/pkg/kg/"

# Redis pattern files
mkdir -p "$BACKUP_DIR/src/pkg/redis"
cp "pkg/redis/pattern_executor.go" "$BACKUP_DIR/src/pkg/redis/"
cp "pkg/redis/pattern_store.go" "$BACKUP_DIR/src/pkg/redis/"

# Nexus example pattern files
mkdir -p "$BACKUP_DIR/src/internal/nexus/examples"
cp "internal/nexus/examples/pattern_examples.go" "$BACKUP_DIR/src/internal/nexus/examples/"

# Nexus service pattern store
mkdir -p "$BACKUP_DIR/src/internal/nexus/service"
cp "internal/nexus/service/pattern_store.go" "$BACKUP_DIR/src/internal/nexus/service/"

# Add backup metadata
echo "Creating backup metadata..."
cat > "$BACKUP_DIR/backup_info.txt" << EOF
Amadeus Backup - Error Handling Enhancement
Timestamp: $(date)
Files included:
- amadeus/knowledge_graph.json
- amadeus/cmd/kgcli/main.go
- amadeus/pkg/kg/knowledge_graph.go
- pkg/redis/pattern_executor.go
- pkg/redis/pattern_store.go
- internal/nexus/examples/pattern_examples.go
- internal/nexus/service/pattern_store.go

Changes: Added proper error handling to all functions that were flagged by the linter.
- Fixed error checking in CLI commands
- Fixed knowledge graph loading error handling
- Fixed pattern executor errors
- Fixed pattern store issues
- Fixed empty if branch
EOF

echo "Backup completed successfully to: $BACKUP_DIR"
echo "Use the following command to restore files if needed:"
echo "  cp -r $BACKUP_DIR/src/* ." 