# Knowledge Graph Sync: Keeping Local and Cloud in Sync

This guide explains how to keep the Amadeus knowledge graph (`knowledge_graph.json`) and backups
synchronized between your local development environment and your cloud deployment.

---

## Why Sync?

- **Local updates** (e.g., during development) are not automatically reflected in the cloud.
- **Cloud updates** (e.g., from running services) are not automatically reflected locally.
- Keeping both in sync ensures consistency, easier debugging, and reliable backups.

---

## Approaches to Sync

### 1. Manual Sync (Simple, but Manual)

- Download the latest `knowledge_graph.json` and backups from the cloud to your local machine (or
  upload local changes to the cloud) as needed.
- Use `scp`, `rsync`, or cloud storage tools (e.g., AWS S3 CLI, Google Cloud Storage CLI).

### 2. Shared Storage Backend

- Mount a shared storage solution (e.g., AWS EFS, Google Filestore, NFS, or a cloud bucket via FUSE)
  in both your local and cloud environments.
- Both local and cloud apps read/write to the same files.

### 3. Automated Sync with Cloud Storage

- Store the knowledge graph and backups in a cloud bucket (e.g., S3, GCS, Azure Blob).
- Use a sync tool or script to:
  - Download the latest from the bucket before starting the app locally or in the cloud.
  - Upload the updated files back to the bucket after changes.
- Automate this with a Makefile, Docker entrypoint, or CI/CD pipeline.

#### Sample AWS S3 Sync Script

```bash
# Download before starting app
aws s3 cp s3://my-bucket/knowledge_graph.json amadeus/knowledge_graph.json
aws s3 sync s3://my-bucket/backups/ amadeus/backups/

# ... run your app ...

# Upload after update or shutdown
aws s3 cp amadeus/knowledge_graph.json s3://my-bucket/knowledge_graph.json
aws s3 sync amadeus/backups/ s3://my-bucket/backups/
```

### 4. GitOps Approach

- Commit the knowledge graph and backups to a Git repository.
- Pull/push changes as part of your deployment or development workflow.
- **Note:** Best for infrequent updates and small files.

---

## Comparison Table

| Method               | Pros                        | Cons                        |
| -------------------- | --------------------------- | --------------------------- |
| Manual sync          | Simple, no infra needed     | Not automatic               |
| Shared storage mount | Always in sync              | Needs networked storage     |
| Cloud storage sync   | Works anywhere, automatable | Needs scripts/tooling       |
| GitOps               | Versioned, auditable        | Not for high-frequency data |

---

## Best Practice

- For most teams: **Use a cloud storage bucket and sync scripts** for both local and cloud
  environments.
- For high-availability or multi-instance: **Use a shared network filesystem or a database** as the
  backend for the knowledge graph.

---

_If you need a ready-to-use sync script, Docker entrypoint, or Makefile target for your setup, see
the examples above or ask for a custom solution!_
