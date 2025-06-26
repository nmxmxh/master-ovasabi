# Security Service Documentation

## Overview

The **Security Service** is the platform's central authority for:

- Authentication (services, workloads, non-human actors)
- Authorization (policy enforcement, RBAC/ABAC, Zero Trust)
- Secrets & Key Management (API keys, certificates, dynamic credentials)
- Security Policy Management (OPA/Rego, YAML, versioned)
- Security Analytics & Audit (event logging, anomaly detection)

**It does NOT handle user profile, user CRUD, or user session management.** Those remain the
exclusive domain of the **User Service**.

---

## Separation of Concerns: Security Service vs User Service

| Responsibility         | User Service                         | Security Service                               |
| ---------------------- | ------------------------------------ | ---------------------------------------------- |
| User CRUD              | ✅ (create, update, delete, profile) | ❌                                             |
| User Authentication    | ✅ (password, OIDC, SSO, MFA)        | ❌ (delegates to User Service for human users) |
| Service/Workload AuthN | ❌                                   | ✅ (mTLS, JWT, SPIFFE, SVID, API keys)         |
| Authorization          | ❌ (delegates to Security Service)   | ✅ (RBAC, ABAC, Zero Trust, policy as code)    |
| Secrets Management     | ❌                                   | ✅ (API keys, certs, dynamic secrets)          |
| Policy Management      | ❌                                   | ✅ (OPA/Rego, YAML, versioned policies)        |
| Security Analytics     | ❌                                   | ✅ (event logging, audit, risk scoring)        |
| User Metadata          | ✅ (roles, preferences, profile)     | ❌ (except for security-specific metadata)     |
| Workload Metadata      | ❌                                   | ✅ (identity, risk, audit, spiffe_id, etc.)    |

**Key Rule:**

- **User Service** is for human users and their lifecycle.
- **Security Service** is for platform-wide security, including non-human actors.

---

## API & Proto Summary

- **Authenticate:** For services, workloads, and non-human actors. Delegates to User Service for
  human users.
- **Authorize:** All services (including User Service) use this to check permissions.
- **IssueSecret:** For API keys, mTLS certs, SVIDs, etc. (never for user passwords).
- **ValidateCredential:** For tokens, certs, SVIDs, etc.
- **QueryEvents:** For security analytics, audit, and compliance.
- **Get/SetPolicy:** For managing security policies.

---

## Security Metadata Pattern

Extend the platform's metadata with a **security-specific namespace**:

```json
{
  "service_specific": {
    "security": {
      "risk_score": 0.92,
      "auth_method": "mTLS",
      "identity_type": "workload",
      "spiffe_id": "spiffe://example.org/ns/default/sa/myservice",
      "policy_version": "2024-05-16",
      "zero_trust": true,
      "audit": {
        "event_id": "evt_123",
        "actor": "service_abc",
        "action": "authorize",
        "resource": "db:orders",
        "timestamp": "2024-05-16T12:00:00Z"
      }
    }
  }
}
```

- **User Service** should only use `service_specific.user` for user metadata.
- **Security Service** should only use `service_specific.security` for security metadata.

---

## Integration Patterns

- **User Service** calls SecurityService.Authorize for all sensitive actions (e.g., admin actions,
  data access).
- **All services** (including User Service) use SecurityService for secrets, policy, and audit.
- **No direct overlap:**
  - User Service does not issue or manage secrets, policies, or workload identity.
  - Security Service does not manage user profiles, passwords, or sessions.

---

## Best Practices & Inspirations

- [HashiCorp Vault](https://developer.hashicorp.com/vault/docs): Centralized secrets, dynamic
  credentials, audit
- [Istio Security](https://istio.io/latest/docs/concepts/security/): Service mesh security, mTLS,
  policy, workload identity
- [SPIFFE/SPIRE](https://spiffe.io/): Workload identity, SVID, federated trust
- [Cloudflare Zero Trust](https://www.cloudflare.com/en-gb/learning/security/glossary/what-is-zero-trust/):
  Identity-based access, least privilege
- [Uber M3](https://www.uber.com/en-NG/blog/m3/),
  [Google SRE Monitoring](https://sre.google/sre-book/monitoring-distributed-systems/): Distributed
  monitoring, audit, and incident response

---

## Summary Table: Security Service Responsibilities

| Capability         | Description                                              | Integration Points                |
| ------------------ | -------------------------------------------------------- | --------------------------------- |
| Identity           | Service, workload, device identity issuance & validation | Service, Mesh, SPIFFE             |
| Authentication     | mTLS, JWT, SVID, API keys                                | All services, mesh, API gateway   |
| Authorization      | RBAC, ABAC, Zero Trust, policy as code                   | All services, mesh, API gateway   |
| Secrets Management | Centralized, dynamic, auditable secrets                  | All services, CI/CD, mesh         |
| Policy Enforcement | mTLS, JWT, OPA, rate limits, audit                       | All services, mesh, API gateway   |
| Security Analytics | Event logging, anomaly detection, metrics, dashboards    | SRE, SIEM, compliance, monitoring |

---

## Change Management

- All changes to this service must be reviewed for overlap with the User Service.
- Any new field in `service_specific.security` must be documented here and referenced in the
  metadata standard.

---

**This documentation ensures a robust, extensible, and non-overlapping security architecture for
your platform.**
