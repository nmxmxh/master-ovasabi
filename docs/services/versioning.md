# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14

## 1. Purpose

Versioning is essential for:

- Ensuring backward compatibility and safe evolution of APIs, services, and metadata.
- Enabling reproducibility, traceability, and impact analysis across the platform.
- Supporting multi-version orchestration, migrations, and rollbacks.

## 2. Versioning Pattern

### a. API & Proto Versioning

- Directory Structure: Each API version lives in its own directory, e.g. `api/protos/user/v1/`,
  `api/protos/campaign/v1/`.
- Semantic Versioning: Use `v1`, `v2`, etc. for breaking changes. Use minor/patch for
  backward-compatible changes in documentation and code comments.
- Message/Service Naming: Do not include version in message/service names; version is in the package
  and directory.

### b. Metadata Versioning

- Knowledge Graph: The knowledge graph JSON includes a top-level `"version"` field and
  `"last_updated"`.
- Service Metadata: Each service/entity can include a `version` field in its metadata (e.g.,
  `metadata.service_specific.{service}.version`).
- Schema Versioning: For extensible fields (e.g., `service_specific`), include a `version` key to
  indicate the schema or logic version for that extension.

### c. Database Versioning

- Migration Tracking: Use migration tools that track schema version (e.g., `schema_migrations`
  table).
- Table Versioning: For tables with evolving schemas, include a `version` column if needed for
  compatibility.

### d. Documentation Versioning

- Docs: All major docs (API, onboarding, standards) should indicate the version they describe.
- Changelog: Maintain a changelog for each service and the knowledge graph.

### e. Service Registration & Orchestration

- Service Registration: Each service registers its version in the knowledge graph and with the
  DI/Provider.
- Orchestration: Nexus and orchestration logic should be version-aware, supporting multi-version
  flows if needed.

---

## 3. Canonical Version Fields

### a. Knowledge Graph

```json
{
  "version": "2.0.0",
  "last_updated": "2025-05-11T00:00:00Z",
  ...
}
```

### b. Service Registration

```go
type ServiceRegistration struct {
    Name         string   `json:"name"`
    Version      string   `json:"version"`
    ...
}
```

### c. Metadata (Proto/JSON)

```json
{
  "metadata": {
    "service_specific": {
      "user": {
        "version": "1.2.0",
        ...
      }
    }
  }
}
```

### d. API/Proto

- Package path: `api/protos/{service}/v1/`
- Example: `package user.v1;`

---

## 4. Versioning Best Practices

- Increment version for any breaking change (API, DB, metadata schema).
- Document all version changes in a changelog and in the Amadeus context.
- Include version in all service registration and orchestration records.
- Expose version in all health/metrics endpoints for observability.
- Use versioned OpenAPI specs and publish them for consumers.
- For extensible metadata, always include a `version` field in the service-specific section.

---

## 5. Example: Versioned Metadata Pattern

```json
{
  "metadata": {
    "features": ["referral", "notification"],
    "service_specific": {
      "campaign": {
        "version": "1.1.0",
        "priority": "high"
      },
      "user": {
        "version": "2.0.0",
        "login_source": "mobile_app"
      }
    }
  }
}
```

---

## 6. Version Evolution & Backups

- Backups: All knowledge graph and critical data are backed up with version and timestamp.
- Historical Retention: Retain historical versions for audit, rollback, and reproducibility.
- Evolution Tracking: Track version evolution in the Amadeus context and knowledge graph.

---

## 7. Versioning Checklist

- [ ] All APIs and protos are versioned by directory/package.
- [ ] All metadata includes a version field where extensible.
- [ ] All services register their version in the knowledge graph.
- [ ] All docs and OpenAPI specs are versioned and published.
- [ ] All migrations and DB schemas are versioned and tracked.
- [ ] All backups include version and timestamp.
- [ ] All orchestration and DI logic is version-aware.

---

## 8. Connecting User, System, and Environment Versioning

### a. Unified Versioning & Environment Field

To enable full traceability, analytics, and environment-aware orchestration, use a unified field in
metadata and user records:

```json
{
  "versioning": {
    "system_version": "2.0.0", // Platform/system version
    "service_version": "1.2.0", // Service or API version
    "user_version": "1.0.0", // User profile or schema version
    "environment": "prod", // dev, test, qa, prod, beta, etc.
    "user_type": "admin", // admin, beta_user, qa_user, regular_user, etc.
    "feature_flags": ["new_ui", "beta_feature"],
    "last_migrated_at": "2025-05-11T00:00:00Z"
  }
}
```

- **system_version:** The current deployed platform version (from knowledge graph or deployment
  config)
- **service_version:** The version of the service handling the request/entity
- **user_version:** The version of the user profile/schema (for migrations, A/B, etc.)
- **environment:** The environment context (dev, test, qa, prod, beta, etc.)
- **user_type:** The type of user (admin, beta, QA, regular, etc.)
- **feature_flags:** List of enabled features for this user/session/entity
- **last_migrated_at:** Timestamp of last migration or version update

### b. Where to Use This Field

- In all extensible metadata (e.g., `metadata.service_specific.{service}.versioning`)
- In user records (for user_type, user_version, feature_flags, etc.)
- In service registration and orchestration records
- In analytics and audit logs

### c. Example: User Metadata with Versioning

```json
{
  "metadata": {
    "service_specific": {
      "user": {
        "versioning": {
          "system_version": "2.0.0",
          "service_version": "1.2.0",
          "user_version": "1.0.0",
          "environment": "beta",
          "user_type": "beta_user",
          "feature_flags": ["new_ui"],
          "last_migrated_at": "2025-05-11T00:00:00Z"
        },
        "login_source": "mobile_app"
      }
    }
  }
}
```

### d. Best Practices for Versioning & Environment Fields

- Always set these fields at entity creation and update them on migration or environment change.
- Use environment and user_type for feature gating, A/B testing, and analytics.
- Expose versioning info in all health, metrics, and debug endpoints.
- Use feature_flags for progressive rollout and experimentation.
- Store last_migrated_at for audit and rollback.

---

## 9. Implementation & Orchestration Guidance

- Use the unified versioning field in all new and refactored services.
- Document versioning and environment fields in your OpenAPI and onboarding docs.
- Ensure all orchestration, analytics, and compliance logic is version/environment aware.
- Reference the Amadeus context and onboarding template for updates.

---

## 10. References

- [Amadeus Context: Version Control](../amadeus/amadeus_context.md#evolution-tracking)
- [Service List: Versioned APIs](./service_list.md)
- [Knowledge Graph: Version Field](../amadeus/knowledge_graph.md)
- [Onboarding Template](./onboarding_template.md)
- [API Proto README](../../api/protos/README.md)

---

**This standard ensures every part of the platform is versioned, traceable, and future-proof. Update
this document as new versioning needs or patterns emerge.**
