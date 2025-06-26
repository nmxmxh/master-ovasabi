# Campaign Definition Template

## 1. Campaign Overview

- **Campaign Name:** [e.g., "Spring Game Launch", "Corporate Website", "Content Strategy Q3"]
- **Type:** [Game | Website | Content Management | ...]
- **Description:** [Brief summary of campaign goals and scope]
- **Owner:** [Team or user responsible]

---

## 2. Campaign Communication Levels

- **User-to-User:** [e.g., chat, forums, multiplayer]
- **User-to-Service:** [e.g., content submission, quiz participation, purchases]
- **Service-to-Service:** [e.g., notifications, analytics, moderation, payments]
- **Admin-to-User:** [e.g., announcements, moderation actions]

---

## 3. Campaign Communication Types

- **Synchronous:** [REST/gRPC APIs, direct calls]
- **Asynchronous:** [WebSocket, event bus, notifications]
- **Batch:** [Scheduled reports, data exports]
- **Real-Time:** [Live updates, multiplayer state, live chat]

---

## 4. Campaign Deployed Resources

- **Frontend:** [Web app, mobile app, kiosk, etc.]
- **Backend Services:** [List of required services: user, content, commerce, notification,
  analytics, etc.]
- **Databases:** [Postgres, Redis, etc.]
- **External Integrations:** [Payment gateways, social media, etc.]
- **Orchestration/Workflow:** [Nexus patterns, scheduled jobs, event triggers]

---

## 5. Campaign Rules & Orchestration

- **Access Control:** [Who can participate, roles, RBAC]
- **Content Moderation:** [Automated/manual, service integration]
- **Localization:** [Supported locales, i18n strategy]
- **Scheduling:** [Start/end dates, time-based triggers]
- **Resource Limits:** [Rate limits, quotas, max users/content]
- **Event Triggers:** [What events trigger workflows or notifications?]
- **Audit & Compliance:** [Logging, audit trails, compliance requirements]

---

## 6. Service & Orchestration Mapping

| Service      | Role in Campaign      | Communication Type | Orchestration/Pattern        |
| ------------ | --------------------- | ------------------ | ---------------------------- |
| User         | Auth, profiles, RBAC  | REST/gRPC          | Nexus registration           |
| Content      | Posts, quizzes, media | REST/WebSocket     | Event-driven, scheduled jobs |
| Commerce     | Purchases, payments   | REST/gRPC          | Payment workflow, audit      |
| Notification | Alerts, updates       | WebSocket, email   | Event-driven                 |
| Analytics    | Engagement, reporting | Batch, REST        | Scheduled jobs, event bus    |
| ...          | ...                   | ...                | ...                          |

---

## 7. Example: Campaign Metadata (for registration)

```json
{
  "type": "game",
  "communication": {
    "levels": ["user-to-user", "user-to-service", "service-to-service"],
    "types": ["real-time", "asynchronous"]
  },
  "resources": {
    "frontend": ["web", "mobile"],
    "services": ["user", "content", "notification", "analytics"],
    "databases": ["postgres", "redis"]
  },
  "rules": {
    "access_control": "RBAC",
    "moderation": "automated",
    "localization": ["en", "fr", "es"],
    "scheduling": {
      "start": "2025-06-01T00:00:00Z",
      "end": "2025-09-01T00:00:00Z"
    }
  }
}
```

---

## 8. References

- [Service Patterns & Best Practices](../amadeus/service_patterns_and_research.md)
- [Nexus Orchestration Docs](../architecture/nexus_future.md)
- [Scheduler Service Docs](../architecture/scheduler_service.md)
- [Amadeus Context](../amadeus/amadeus_context.md)

---

**This template ensures every campaign is well-defined, orchestrated, and ready for rapid deployment
and scaling across the OVASABI platform.**
