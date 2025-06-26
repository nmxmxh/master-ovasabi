# Scheduler Service Documentation

## Overview

The Scheduler Service is a first-class, production-grade component responsible for orchestrating
background jobs, maintenance tasks, and event-driven workflows across the OVASABI platform. It is
deeply integrated with Nexus (for orchestration and eventing), the security service (for RBAC and
audit), and the knowledge graph (for discoverability and impact analysis).

---

## Key Features & Requirements

- **Service Abstraction**: Dedicated, discoverable, and orchestrated via Nexus. Registered in the
  knowledge graph and exposes gRPC/REST APIs for job management.
- **Job Definition & Metadata**: Standardized proto messages for jobs (e.g., `ScheduledJob`,
  `MaintenanceJob`) with rich metadata for configuration, audit, and security context.
- **Integration with Nexus**: Registers itself and all jobs with Nexus, emits/consumes events for
  job lifecycle, supports event-driven and time-based triggers.
- **Security & Access Control**: Enforces RBAC and audit logging for all job management APIs. All
  executions are logged as security events.
- **Observability & Health**: Exposes Prometheus metrics, structured logs, and OpenTelemetry traces.
  Health endpoints for liveness/readiness.
- **Concurrency & Worker Pools**: Dynamic worker pools for parallel job execution, configurable
  concurrency, rate limiting, and backpressure. Idempotent and safe job execution.
- **Persistence & State**: Persists job definitions, schedules, and state in a reliable store
  (Postgres, Redis, or both). Supports recovery and replay.
- **Extensibility**: Allows new job types and handlers to be registered dynamically. Supports both
  cron/time-based and event-based triggers.

---

## Proto Example: Scheduled Job

```protobuf
message ScheduledJob {
  string id = 1;
  string type = 2; // e.g., "cleanup", "archive", "notify"
  string owner_service = 3;
  google.protobuf.Timestamp schedule = 4;
  string status = 5; // "pending", "running", "success", "failed"
  common.Metadata metadata = 6;
  google.protobuf.Timestamp last_run = 7;
  google.protobuf.Timestamp next_run = 8;
  string created_by = 9;
  string updated_by = 10;
}
```

---

## Integration Flow

1. Job is registered with Scheduler (and Nexus).
2. Job metadata includes schedule, owner, and security context.
3. Nexus orchestrates dependencies and triggers jobs (time-based or event-based).
4. Scheduler executes job using worker pool, logs execution, and emits events.
5. Security service audits all job actions and logs to `service_security_event`.
6. Observability tools track job health, performance, and failures.

---

## Best Practices

- Use Go's scheduler and goroutines for efficient, concurrent background jobs.
- Integrate with gRPC service lifecycle for control and observability.
- Use protos and metadata for extensibility, auditing, and analytics.
- Add metrics, logging, and tracing for production-grade reliability.
- Ensure jobs are idempotent and safe for concurrent execution.
- Use context cancellation and graceful shutdown for lifecycle management.
- Persist job state and support recovery after failures.
- Enforce RBAC and audit logging for all job actions.

---

## Final Checklist

- [x] Scheduler is a first-class, registered service.
- [x] All jobs and executions are tracked in metadata and knowledge graph.
- [x] RBAC and audit logging are enforced for all job actions.
- [x] Nexus orchestrates and tracks all scheduled/event-driven jobs.
- [x] Observability and health are built-in.
- [x] Extensible for new job types, triggers, and integrations.

---

## References

- [Go Scheduler Deep Dive (Ardan Labs)](https://www.ardanlabs.com/blog/2018/08/scheduling-in-go-part2.html)
- [Temporal Workflow Engine](https://temporal.io/)
- [robfig/cron](https://github.com/robfig/cron)
- [gRPC-Go Examples](https://github.com/grpc/grpc-go/tree/master/examples)
