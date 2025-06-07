# OVASABI Scheduler Service: Advanced Job Management

## Overview

The Scheduler Service provides robust, production-grade job scheduling and orchestration for the
OVASABI platform. It supports dynamic job reload, dependency graphs, real-time UI hooks, and
seamless integration with orchestration patterns (Nexus).

---

## Key Features

### 1. Dynamic Reload of Jobs

- **Real-time updates:** Jobs are added, updated, or removed from the scheduler immediately when
  created, updated, or deleted via API, UI, or orchestration.
- **Event-driven:** The scheduler listens for job CRUD events and updates the active schedule
  without requiring a restart.
- **Implementation:** Uses a `sync.Map` to track active cron jobs by job ID and updates the
  robfig/cron scheduler in real time.

### 2. Dependency Graph

- **Job dependencies:** Each job can declare dependencies (other job IDs) in its metadata. Jobs only
  run when all dependencies are completed.
- **Graph validation:** The system can detect cycles in the dependency graph to prevent deadlocks
  (cycle detection stub included).
- **Metadata pattern:**
  ```json
  {
    "metadata": {
      "scheduling": {
        "dependencies": ["job_id_1", "job_id_2"]
      }
    }
  }
  ```

### 3. UI Hooks & Real-Time Events

- **Event emission:** All job lifecycle events (`created`, `updated`, `deleted`, `started`,
  `completed`, `failed`) are emitted to the event bus and WebSocket clients for real-time UI
  updates.
- **UI endpoints:**
  - `GetJobGraph`: Returns the job dependency graph for visualization.
  - `GetJobStatus`: Returns the current status of a job.
- **Example event payload:**
  ```json
  {
    "type": "scheduler.job_completed",
    "payload": { "job_id": "abc123", "status": "COMPLETED" },
    "metadata": { ... }
  }
  ```

### 4. Pattern/Orchestration Integration

- **Jobs as patterns:** Jobs can be registered as patterns in Nexus, enabling cross-service
  workflows and reusable job templates.
- **Orchestration triggers:** Jobs can be triggered as part of larger orchestration flows (e.g.,
  after a campaign launch).
- **Metadata pattern:**
  ```json
  {
    "metadata": {
      "service_specific": {
        "scheduler": {
          "pattern_id": "pattern_xyz"
        }
      }
    }
  }
  ```

---

## Usage Examples

### A. Scheduling a Cron Job with Dependencies

```json
{
  "job": {
    "name": "Weekly Report",
    "metadata": {
      "scheduling": {
        "cron": "0 9 * * 1",
        "timezone": "America/New_York",
        "dependencies": ["job_id_data_refresh"],
        "retry_policy": {
          "max_attempts": 5,
          "backoff": "exponential"
        }
      }
    }
  }
}
```

### B. Real-Time UI Integration

- Subscribe to job events via WebSocket to update job status and graph in the frontend.
- Use `GetJobGraph` to visualize job dependencies and execution order.

### C. Orchestration-Driven Job Execution

- Register jobs as patterns in Nexus for reuse in multi-step workflows.
- Trigger jobs from orchestration flows using the `TriggerJobFromOrchestration` method.

---

## Extensibility & Best Practices

- **Metadata-driven:** All advanced scheduling, dependencies, and orchestration info are stored in
  canonical metadata fields.
- **Event-driven:** Use the event bus for all job state changes and orchestration triggers.
- **UI/UX:** Expose endpoints and events for real-time, interactive job management and
  visualization.
- **Pattern registration:** Register reusable job templates and patterns for rapid onboarding and
  automation.
- **Cycle detection:** Always validate the dependency graph to prevent deadlocks.

---

## References

- [docs/amadeus/amadeus_context.md](amadeus/amadeus_context.md) (Canonical metadata and
  orchestration patterns)
- [robfig/cron](https://github.com/robfig/cron) (Cron scheduling library)
- [cenkalti/backoff](https://github.com/cenkalti/backoff) (Retry/backoff library)
- [Nexus Orchestration](../nexus/pattern/README.md)

---

For onboarding, integration, and advanced usage, see the Scheduler Service code and the Amadeus
context documentation.
