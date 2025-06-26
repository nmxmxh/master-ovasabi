# Notification Service Documentation

## Overview

The Notification Service provides unified, extensible delivery of notifications across multiple
channels:

- **Email** (using MJML for responsive HTML)
- **Push Notifications** (JSON payloads)
- **SMS/Text Messages** (plain text)

All notifications follow the robust metadata pattern for extensibility, analytics, and
orchestration.

---

## Supported Channels & Standards

### Email

- **Standard:** MJML (compiled to HTML), rendered with Go's `html/template`.
- **Best Practice:** Store MJML templates, compile to HTML, render with dynamic data, send via
  provider.
- **Recommended Providers:**
  - **Azure Communication Services** (production, scalable, cost-effective)
  - **Mailgun, SendGrid, Amazon SES** (alternatives)
- **Go SDKs:**
  - Use Go's `net/http` for Azure REST API
  - `github.com/jordan-wright/email` or `github.com/go-mail/mail` for SMTP

### Push Notifications

- **Standard:** JSON payloads, rendered with Go's `text/template`.
- **Best Practice:** Store push templates as Go templates, render with dynamic data, send via
  provider.
- **Recommended Providers:**
  - **Firebase Cloud Messaging (FCM)** (Android, web, cross-platform)
  - **Apple Push Notification Service (APNs)** (iOS)
  - **OneSignal, Pusher, AWS SNS** (unified/multi-platform)
- **Go SDKs:**
  - [`github.com/appleboy/go-fcm`](https://github.com/appleboy/go-fcm) for FCM
  - [`github.com/sideshow/apns2`](https://github.com/sideshow/apns2) for APNs
  - Use provider REST APIs for others

### SMS/Text Messages

- **Standard:** Plain text, rendered with Go's `text/template`.
- **Best Practice:** Store SMS templates as Go templates, render with dynamic data, send via
  provider.
- **Recommended Providers:**
  - **Twilio** (global, reliable, developer-friendly)
  - **AWS SNS** (scalable, cost-effective)
  - **Nexmo/Vonage** (alternative)
- **Go SDKs:**
  - [`github.com/twilio/twilio-go`](https://github.com/twilio/twilio-go) for Twilio
  - [`github.com/aws/aws-sdk-go`](https://github.com/aws/aws-sdk-go) for SNS
  - Use provider REST APIs for others

---

## Metadata Pattern

All notifications use the robust metadata pattern for extensibility and analytics:

- `metadata.channel`: email, push, sms, etc.
- `metadata.status`: pending, sent, delivered, failed, read
- `metadata.user_id`, `metadata.campaign_id`, etc.
- `metadata.service_specific.notification`: channel-specific extensions

Example:

```json
{
  "channel": "email",
  "status": "sent",
  "user_id": "user_123",
  "service_specific": {
    "notification": {
      "template": "welcome_email",
      "provider": "azure",
      "delivery_time": "2024-05-15T12:00:00Z"
    }
  }
}
```

---

## Example API Usage

### Send Email

```go
_, err := notificationClient.SendEmail(ctx, &notificationpb.SendEmailRequest{
    To:      "user@example.com",
    Subject: "Welcome!",
    Body:    renderedHTML,
    Html:    true,
})
```

### Send Push Notification

```go
_, err := notificationClient.SendNotification(ctx, &notificationpb.SendNotificationRequest{
    UserId:  "user_123",
    Channel: "push",
    Title:   "Welcome!",
    Body:    "Your account is ready.",
    Payload: map[string]string{"deep_link": "ovasabi://welcome"},
})
```

### Send SMS

```go
_, err := notificationClient.SendNotification(ctx, &notificationpb.SendNotificationRequest{
    UserId:  "user_123",
    Channel: "sms",
    Body:    "Hi Alice, your code is 123456",
})
```

---

## Extensibility Notes

- Add new channels by extending the proto and provider interface.
- All templates are stored and versioned for auditability.
- Metadata enables analytics, orchestration, and compliance.

---

## Azure Optimization

The Notification Service is optimized for Azure and Go for cost-effectiveness, scalability, and
maintainability:

### Email (Azure Communication Services)

- Use Azure Communication Services (ACS) Email REST API for sending emails.
- Store MJML templates (compiled to HTML) and render with Go's `html/template`.
- Host images and SVGs on Azure Blob Storage or Azure CDN for fast, reliable delivery.
- Authenticate using Azure access keys or managed identity.
- Monitor delivery and errors with Azure Monitor and Application Insights.

### Push Notifications (Azure Notification Hubs)

- Use Azure Notification Hubs for unified push delivery (supports FCM, APNs, WNS, etc.).
- Integrate with Go using REST API (`net/http`).
- Store push templates as Go text templates (JSON format) and render dynamically.
- Register and manage device tokens in your database.
- Monitor delivery status with Azure Monitor.

### SMS (Azure Communication Services SMS)

- Use Azure Communication Services SMS REST API for global SMS delivery.
- Store SMS templates as Go text templates and render with dynamic data.
- Monitor SMS delivery analytics with Azure Monitor.

### General Go/Azure Best Practices

- Store all Azure endpoints, keys, and config in environment variables or Azure Key Vault.
- Use Go's default HTTP client with connection pooling for REST calls.
- Implement retry logic for transient Azure API errors (exponential backoff).
- For high volume, queue notifications (Azure Queue Storage, Service Bus) and process with Go
  workers.
- Log all sends, failures, and delivery receipts; integrate with Azure Monitor for dashboards and
  alerts.
- Never hardcode secrets; use managed identity or Key Vault where possible.

---

## References

- [MJML](https://mjml.io/)
- [Go html/template](https://pkg.go.dev/html/template)
- [FCM Go SDK](https://github.com/appleboy/go-fcm)
- [APNs2 Go SDK](https://github.com/sideshow/apns2)
- [Twilio Go SDK](https://github.com/twilio/twilio-go)
- [AWS SNS Go SDK](https://github.com/aws/aws-sdk-go)
