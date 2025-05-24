# 004_super_app.md

## Overview

The **Super App Campaign** aims to deliver a unified, multi-service experience to users, combining
social networking, e-commerce, messaging, payments, and mini-apps within a single platform. This
campaign orchestrates core OVASABI services to enable seamless cross-feature workflows, shared user
identity, and extensible metadata-driven integration.

---

## User Stories

1. **Unified Profile:**  
   As a user, I want a single profile for all services (social, commerce, messaging, payments).

2. **Social Feed:**  
   As a user, I can post updates, share media, comment, and like posts.

3. **Marketplace:**  
   As a user, I can browse, search, buy, and sell products/services.

4. **Messaging:**  
   As a user, I can chat with friends, create groups, and share content.

5. **Payments & Wallet:**  
   As a user, I can manage my wallet, make payments, and view transaction history.

6. **Mini-Apps:**  
   As a user, I can access third-party mini-apps (e.g., games, bookings) within the super app.

7. **Notifications:**  
   As a user, I receive real-time notifications for messages, orders, promotions, and system events.

---

## Required Backend Functionality & Service Responsibilities

| Service      | Responsibilities in Super App Campaign                                |
| ------------ | --------------------------------------------------------------------- |
| User         | Unified identity, profile, RBAC, wallet, preferences                  |
| Content      | Social posts, comments, media, feed, mini-app content                 |
| Commerce     | Product listings, orders, payments, wallet, transaction history       |
| Messaging    | Real-time chat, group messaging, content sharing                      |
| Notification | Real-time and scheduled notifications, promotions, system alerts      |
| Search       | Cross-entity search (users, posts, products, mini-apps)               |
| Analytics    | Engagement, sales, feature usage, campaign performance                |
| Mini-Apps    | Registration, integration, and orchestration of third-party mini-apps |

---

## Frontend-Backend Communication Patterns

- **REST/gRPC APIs:** For all core entity CRUD and orchestration.
- **WebSocket:** For real-time messaging, notifications, and live updates.
- **Composable Request Pattern:** All requests include `metadata` for extensibility.
- **Mini-App Integration:** Mini-apps communicate via secure APIs and metadata contracts.

---

## Extensibility and Reproducibility Notes

- All features are modular and can be enabled/disabled per user or campaign via metadata.
- New mini-apps or features can be added by registering them in the campaign and knowledge graph.
- All entities use the robust metadata pattern for future-proofing and analytics.

---

## Implementation Notes and Next Steps

1. **Define campaign in `docs/campaign/004_super_app.md`.**
2. **Register campaign in Amadeus context and knowledge graph.**
3. **Extend/compose service APIs as needed (e.g., add mini-app endpoints).**
4. **Implement unified search and notification flows.**
5. **Integrate wallet/payments with commerce and user services.**
6. **Document all new metadata fields and service interactions.**
7. **Test, monitor, and review.**

---

## Campaign-Specific API Endpoints and Event Types

- `/api/superapp/feed` — Social feed (content service)
- `/api/superapp/marketplace` — Marketplace (commerce/product service)
- `/api/superapp/chat` — Messaging (messaging service)
- `/api/superapp/wallet` — Payments and wallet (commerce/user service)
- `/api/superapp/miniapps` — Mini-app registry and launch
- `/ws/superapp/{user_id}` — Real-time updates (WebSocket)
- Event types: `new_message`, `order_placed`, `promotion`, `miniapp_event`, etc.

---

## Mapping to Existing Services and New Requirements

| Feature         | Service(s) Used   | New Requirements?               |
| --------------- | ----------------- | ------------------------------- |
| Social Feed     | Content, User     | None (extend content types)     |
| Marketplace     | Commerce, Product | None (link to campaign)         |
| Messaging       | Messaging, User   | Add messaging service if needed |
| Payments/Wallet | Commerce, User    | Extend wallet/payment flows     |
| Mini-Apps       | Mini-Apps, User   | Register/integrate mini-apps    |
| Notifications   | Notification      | None                            |
| Search          | Search            | Cross-entity search config      |

---

## Mapping to Knowledge Graph

- Register campaign, all participating services, and relationships in Amadeus knowledge graph.
- Document all new metadata fields and orchestration patterns.

---

## Example: Super App Feed Post (Composable Request)

```json
{
  "type": "feed_post",
  "fields": {
    "text": "Just bought a new bike on the marketplace!",
    "media": ["bike.jpg"]
  },
  "metadata": {
    "service_specific": {
      "campaign": {
        "campaign_id": "super_app_2024"
      },
      "content": {
        "post_type": "marketplace_purchase"
      }
    }
  }
}
```

---

## References

- [Amadeus Context](../amadeus/amadeus_context.md)
- [Service Patterns & Best Practices](../amadeus/service_patterns_and_research.md)
- [Composable Request Pattern](../amadeus/amadeus_context.md#composable-request-pattern-standard)
- [Knowledge Graph Integration](../amadeus/amadeus_context.md#knowledge-graph-structure)

---

## Amadeus Context Integration

- Register this campaign and all new relationships in the Amadeus context and knowledge graph.
- Update documentation as new features and mini-apps are added.

---

## Checklist

- [ ] Campaign file created and documented
- [ ] All services registered in knowledge graph
- [ ] APIs and endpoints documented
- [ ] Metadata fields defined and registered
- [ ] Orchestration patterns described
- [ ] Testing and monitoring in place

---

**This document is experimental and intended as a living reference for building and evolving the
Super App campaign on the OVASABI platform.**
