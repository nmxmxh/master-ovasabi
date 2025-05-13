# Azure Deployment & Integration Guide (OVASABI Platform)

## Overview

This guide describes how to deploy and operate the current OVASABI stack on Microsoft Azure,
focusing on:

- The core backend services: **App (Go microservices), PostgreSQL, Redis, LibreTranslate**
- Strategies for deploying multiple frontends (web, mobile, etc.) that leverage this backend
- Cost-saving and operational best practices

---

## 1. Core Backend Services on Azure

### **A. App (Go Microservices)**

- **Recommended Azure Service:**
  - [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/en-us/products/kubernetes-service/)
    for production-grade orchestration and scaling.
  - [Azure Container Apps](https://azure.microsoft.com/en-us/products/container-apps/) for simpler,
    event-driven, or scale-to-zero workloads.
- **How to Deploy:**
  - Build Docker images for each microservice.
  - Push to
    [Azure Container Registry](https://azure.microsoft.com/en-us/products/container-registry/) (ACR)
    or Docker Hub.
  - Deploy using Helm charts or YAML manifests (AKS), or via Azure Portal (Container Apps).
- **Cost Tips:**
  - Use spot/preemptible nodes for dev/test.
  - Scale down or use scale-to-zero for non-critical services.

### **B. PostgreSQL**

- **Recommended Azure Service:**
  - [Azure Database for PostgreSQL Flexible Server](https://azure.microsoft.com/en-us/products/postgresql/flexible-server/)
- **How to Deploy:**
  - Provision a managed PostgreSQL instance.
  - Enable required extensions (`uuid-ossp`, `pg_trgm`).
  - Configure VNet integration for security.
- **Cost Tips:**
  - Start with a burstable or basic tier.
  - Use auto-stop for dev/test environments.

### **C. Redis**

- **Recommended Azure Service:**
  - [Azure Cache for Redis](https://azure.microsoft.com/en-us/products/cache/)
- **How to Deploy:**
  - Provision a managed Redis instance.
  - Connect from your app using the Redis endpoint and credentials.
- **Cost Tips:**
  - Use Basic/Standard tier for most workloads.
  - Monitor usage and scale only if needed.

### **D. LibreTranslate**

- **Recommended Azure Service:**
  - [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/en-us/products/kubernetes-service/)
    or [Azure Container Apps](https://azure.microsoft.com/en-us/products/container-apps/)
- **How to Deploy:**
  - Use the official LibreTranslate Docker image.
  - Deploy as a service in your AKS cluster or as a standalone Container App.
  - Expose via internal service (for backend-only use) or public endpoint (if needed).
- **Cost Tips:**
  - Use scale-to-zero or minimal replicas for low-usage scenarios.
  - Monitor resource usage and right-size pods/containers.

---

## 2. Deploying Multiple Frontends (Web, Mobile, etc.)

### **A. Azure Static Web Apps**

- **Service:**
  [Azure Static Web Apps](https://azure.microsoft.com/en-us/products/app-service/static/)
- **Use for:** React, Vue, Angular, Svelte, or static HTML/JS frontends.
- **How to Deploy:**
  - Connect your GitHub repo for CI/CD.
  - Configure build and deployment via Azure Portal or YAML.
  - Set up custom domains and SSL easily.
- **Integration:**
  - Frontends call backend APIs (Go microservices, LibreTranslate) via HTTPS endpoints.
  - Use environment variables for API endpoints.
- **Cost Tips:**
  - Free tier available for most small/medium projects.

### **B. Azure App Service (Web Apps)**

- **Service:** [Azure App Service](https://azure.microsoft.com/en-us/products/app-service/)
- **Use for:** Server-rendered apps (Next.js, Nuxt.js, Django, etc.) or APIs.
- **How to Deploy:**
  - Deploy via GitHub Actions, Azure DevOps, or ZIP upload.
  - Supports custom domains, SSL, and scaling.
- **Integration:**
  - Web apps call backend APIs via internal or public endpoints.

### **C. Azure Front Door or API Management**

- **Service:** [Azure Front Door](https://azure.microsoft.com/en-us/products/frontdoor/) or
  [API Management](https://azure.microsoft.com/en-us/products/api-management/)
- **Use for:**
  - Global load balancing, SSL termination, and API gateway features.
  - Securely expose backend APIs to multiple frontends.
- **How to Deploy:**
  - Configure routing rules to direct traffic to the correct backend service.
  - Add authentication, rate limiting, and monitoring as needed.

### **D. Mobile Apps (React Native, Flutter, etc.)**

- **How to Integrate:**
  - Mobile apps communicate with backend APIs over HTTPS.
  - Use Azure App Center for build, test, and distribution.
  - Store assets (images, files) in Azure Blob Storage if needed.

---

## 3. Security & Cost Optimization

- Use [Azure Key Vault](https://azure.microsoft.com/en-us/products/key-vault/) for secrets
  management (DB credentials, API keys).
- Use VNet integration and private endpoints for secure backend communication.
- Monitor usage with [Azure Monitor](https://azure.microsoft.com/en-us/products/monitor/) and set up
  alerts for cost and performance.
- Use serverless and pay-per-use options (Container Apps, Static Web Apps) to minimize idle costs.

---

## 4. Example Architecture Diagram

```
+-------------------+         +-------------------+         +-------------------+
|   Static Web App  | <-----> |   Azure Front Door| <-----> |   AKS/Container   |
|   (React, etc.)   |         |   or API Mgmt     |         |   Apps (Go, LT)   |
+-------------------+         +-------------------+         +-------------------+
         |                                                        |
         |                                                        |
         v                                                        v
+-------------------+         +-------------------+         +-------------------+
|   Mobile App      | <-----> |   Azure Front Door| <-----> |   PostgreSQL      |
|   (ReactNative)   |         |   or API Mgmt     |         |   Azure Redis     |
+-------------------+         +-------------------+         +-------------------+
```

---

## 5. References & Further Reading

- [Azure Kubernetes Service](https://azure.microsoft.com/en-us/products/kubernetes-service/)
- [Azure Container Apps](https://azure.microsoft.com/en-us/products/container-apps/)
- [Azure Database for PostgreSQL](https://azure.microsoft.com/en-us/products/postgresql/)
- [Azure Cache for Redis](https://azure.microsoft.com/en-us/products/cache/)
- [Azure Static Web Apps](https://azure.microsoft.com/en-us/products/app-service/static/)
- [Azure Front Door](https://azure.microsoft.com/en-us/products/frontdoor/)
- [Azure Key Vault](https://azure.microsoft.com/en-us/products/key-vault/)
- [LibreTranslate Docker](https://hub.docker.com/r/libretranslate/libretranslate)

---

**Summary:**

- Use managed Azure services for PostgreSQL and Redis.
- Deploy Go microservices and LibreTranslate as containers (AKS or Container Apps).
- Use Static Web Apps or App Service for multiple frontends.
- Secure, monitor, and scale with Azure's built-in tools.
- Optimize costs by starting small, using serverless/containerized options, and scaling as needed.
