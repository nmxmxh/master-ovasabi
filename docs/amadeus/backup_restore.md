# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14


This document provides detailed instructions on backing up and restoring the Amadeus Knowledge Graph
system.

## 1. Backup Methods

Amadeus offers multiple methods for backing up the knowledge graph and related components:

### 1.1. Using the CLI Tool

The most convenient way to create a backup is using the `kgcli` command-line tool:

```bash
bin/kgcli backup --desc "Pre-deployment backup"

bin/kgcli list-backups

bin/kgcli list-backups --format json

bin/kgcli restore --path amadeus/backups/knowledge_graph_20230615_120000.json
```

### 1.2. Using the Backup Script

For more comprehensive backups including source code, use the backup script:

```bash
./amadeus/backup_script.sh
```

This script creates a timestamped backup directory containing:

- The knowledge graph JSON file
- Source code files of key components
- Backup metadata

### 1.3. Programmatic Backup

To create backups programmatically from your Go code:

```go
import "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"

func createBackup() {
    // Get the knowledge graph
    graph := kg.DefaultKnowledgeGraph()

    // Create a backup
    info, err := graph.Backup("Programmatic backup")
    if err != nil {
        // Handle error
    }

    // Use backup info
    fmt.Printf("Backup created: %s\n", info.FilePath)
}
```

## 2. Backup Contents

### 2.1. Knowledge Graph Data

Backups include the complete knowledge graph data:

- System components
- Repository structure
- Services and their relationships
- Patterns
- Database practices
- Redis practices
- Amadeus integration metadata

### 2.2. Source Code Backups

When using the backup script, the following source files are included:

- `amadeus/cmd/kgcli/main.go`: CLI tool
- `amadeus/pkg/kg/knowledge_graph.go`: Knowledge Graph API
- Pattern-related files:
  - `pkg/redis/pattern_executor.go`
  - `pkg/redis/pattern_store.go`
  - `internal/nexus/examples/pattern_examples.go`
  - `internal/nexus/service/pattern_store.go`

## 3. Backup Automation

### 3.1. Scheduled Backups

Set up a cron job to run automatic backups:

```bash
0 2 * * * cd /path/to/repository && bin/kgcli backup --desc "Daily automatic backup"
```

### 3.2. Pre-Deployment Backups

Add backups to your deployment pipeline:

```yaml
- name: Backup Knowledge Graph
  run: |
    bin/kgcli backup --desc "Pre-deployment backup for version ${{ github.ref }}"
```

### 3.3. Event-Triggered Backups

Create backups before major changes:

```go
// Before updating multiple services
func updateServices() {
    // Create backup first
    kg.DefaultKnowledgeGraph().Backup("Pre-service update backup")

    // Proceed with updates
    // ...
}
```

## 4. Restore Process

### 4.1. Restoring via CLI

To restore the knowledge graph from a backup:

```bash
bin/kgcli restore --path amadeus/backups/knowledge_graph_20230615_120000.json
```

### 4.2. Restoring Source Code

To restore source code from a script-generated backup:

```bash
cp -r amadeus/backups/20230615_120000/src/* .
```

### 4.3. Programmatic Restore

To restore programmatically:

```go
import "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"

func restoreFromBackup(backupPath string) error {
    // Get the knowledge graph
    graph := kg.DefaultKnowledgeGraph()

    // Restore from backup
    if err := graph.RestoreFromBackup(backupPath); err != nil {
        return err
    }

    // Save the restored graph
    return graph.Save("amadeus/knowledge_graph.json")
}
```

## 5. Managing Backups

### 5.1. Backup Rotation

Implement a backup rotation strategy to manage disk space:

```bash
find amadeus/backups -name "knowledge_graph_*.json" | sort -r | tail -n +11 | xargs rm -f
```

### 5.2. Backup Verification

Verify backups to ensure they are usable:

```go
func verifyBackup(backupPath string) bool {
    // Try loading the backup
    _, err := kg.LoadFromFile(backupPath)
    return err == nil
}
```

### 5.3. External Storage

Copy important backups to external storage for disaster recovery:

```bash
aws s3 cp amadeus/backups/knowledge_graph_20230615_120000.json s3://your-bucket/backups/
```

## 6. Best Practices

1. **Regular Backups**: Schedule automatic daily backups
2. **Pre-Change Backups**: Create backups before major changes
3. **Versioned Backups**: Include version information in backup descriptions
4. **Backup Testing**: Periodically verify backups can be restored
5. **External Storage**: Store critical backups in multiple locations
6. **Rotation Policy**: Implement a backup rotation policy
7. **Backup Documentation**: Document when backups were taken and why

## 7. Troubleshooting

### 7.1. Common Issues

| Issue                 | Resolution                                          |
| --------------------- | --------------------------------------------------- |
| Backup file not found | Check the path and ensure backups directory exists  |
| Permission denied     | Ensure proper file permissions on backups directory |
| Invalid backup format | Verify the backup file is a valid JSON file         |
| Restore conflict      | Backup current state before restoring from backup   |

### 7.2. Emergency Recovery

If all else fails, you can recreate the knowledge graph from scratch:

1. Create an empty knowledge graph:

   ```go
   newGraph := &kg.KnowledgeGraph{
       Version: "1.0.0",
       // Initialize other fields
   }
   ```

2. Save the new knowledge graph:

   ```go
   newGraph.Save("amadeus/knowledge_graph.json")
   ```

3. Rebuild the knowledge graph by scanning services and patterns

## 8. Further Reading

- [Amadeus Knowledge Graph Implementation Guide](implementation_guide.md)
- [Consistent Update Guide](consistent_updates.md)
- [API Reference](api_reference.md) for programmatic backup/restore
