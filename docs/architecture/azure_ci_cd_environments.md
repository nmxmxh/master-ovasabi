# Azure DevOps CI/CD & Environment Strategy

## Overview

This guide describes how to structure your Azure DevOps pipelines and application for a robust
multi-environment workflow:

- **Environments:** dev, qa, staging, prod
- **Feature branch validation**
- **Best practices for environment variables, secrets, resource groups, and deployment automation**

---

## 1. Pipeline Triggers & Branch Strategy

### **A. Feature Branch Validation**

```yaml
trigger:
  branches:
    exclude:
      - develop
      - qa
      - staging
      - prod
```

- **Purpose:** Build and validate Docker images for all branches except main environments. Used for
  PRs and feature branches.

### **B. Environment-Specific Pipelines**

- Use separate pipelines (or pipeline stages) for `develop`, `qa`, `staging`, and `prod` branches.
- These pipelines build, push, and deploy images to the corresponding Azure environment.

---

## 2. Example Pipeline: Build & Validate (Feature Branches)

```yaml
trigger:
  branches:
    exclude:
      - develop
      - qa
      - staging
      - prod

resources:
  - repo: self

variables:
  dockerRegistryServiceConnection: 'YOUR-REGISTRY-SERVICE-CONNECTION-ID'
  imageRepository: 'frontend'
  containerRegistry: 'inhousecr.azurecr.io'
  dockerfilePath: 'Dockerfile'
  tag: '$(Build.SourceBranchName)'
  vmImageName: 'ubuntu-latest'

stages:
  - stage: Build
    displayName: Build and Validate Image
    jobs:
      - job: Build_and_Validate
        displayName: Build and Validate
        pool:
          vmImage: $(vmImageName)
        steps:
          - task: Docker@2
            displayName: Build
            inputs:
              command: build
              arguments: '--build-arg NODEENV="production"'
              repository: $(imageRepository)
              Dockerfile: $(dockerfilePath)
              tags: |
                $(tag)
```

---

## 3. Example Pipeline: Multi-Environment Build & Deploy

```yaml
trigger:
  - develop
  - qa
  - staging
  - prod

resources:
  - repo: self

variables:
  dockerRegistryServiceConnection: 'YOUR-REGISTRY-SERVICE-CONNECTION-ID'
  imageRepository: 'frontend'
  containerRegistry: 'inhousecr.azurecr.io'
  dockerfilePath: 'Dockerfile'
  tag: '$(Build.SourceBranchName)'
  vmImageName: 'ubuntu-latest'

stages:
  - stage: Build
    displayName: Build and push stage
    jobs:
      - job: Build_and_Push
        displayName: Build and Push
        pool:
          vmImage: $(vmImageName)
        steps:
          - task: Docker@2
            displayName: Build for develop
            condition: eq(variables['Build.SourceBranchName'], 'develop')
            inputs:
              command: build
              arguments: '--build-arg NODEENV="production" --build-arg APIURL=$(DEV_API_URL)'
              containerRegistry: $(dockerRegistryServiceConnection)
              repository: $(imageRepository)
              Dockerfile: $(dockerfilePath)
              tags: |
                $(tag)
          # Repeat for qa, staging, prod with appropriate variables
          - task: Docker@2
            displayName: Push
            inputs:
              command: push
              containerRegistry: $(dockerRegistryServiceConnection)
              repository: $(imageRepository)
              tags: |
                $(tag)
```

---

## 4. Best Practices for Azure Integration

### **A. Environment Variables & Secrets**

- Store all environment-specific config (API URLs, DB/Redis endpoints, etc.) in environment
  variables.
- Use [Azure Key Vault](https://azure.microsoft.com/en-us/products/key-vault/) for secrets and
  inject them at deploy time.
- Use variable groups in Azure DevOps for non-secret config.

### **B. Resource Groups & Isolation**

- Create separate Azure resource groups for dev, qa, staging, and prod.
- Use separate managed PostgreSQL/Redis instances per environment for isolation.

### **C. Deployment Automation**

- Use Azure DevOps Release Pipelines or GitHub Actions to automate deployment to AKS/Container Apps
  after a successful build.
- Use Helm or YAML manifests for AKS, or ARM/Bicep templates for infrastructure as code.

### **D. Logging & Monitoring**

- Output logs to stdout/stderr for Azure Monitor to collect.
- Set up alerts for build/deploy failures and resource usage.

### **E. Health Checks**

- Implement `/healthz` endpoints for all services.
- Configure AKS/Container Apps to use these for liveness/readiness probes.

---

## 5. Summary Table: Pipeline & Environment Flow

| Branch/Env | Pipeline Trigger | Build/Validate | Push to ACR | Deploy to Azure | Notes                     |
| ---------- | ---------------- | -------------- | ----------- | --------------- | ------------------------- |
| Feature/PR | Yes              | Yes            | No          | No              | For validation only       |
| develop    | Yes (separate)   | Yes            | Yes         | Yes (Dev)       | Deploy to dev environment |
| qa         | Yes (separate)   | Yes            | Yes         | Yes (QA)        | Deploy to QA environment  |
| staging    | Yes (separate)   | Yes            | Yes         | Yes (Staging)   | Deploy to staging         |
| prod       | Yes (separate)   | Yes            | Yes         | Yes (Prod)      | Deploy to production      |

---

## 6. References

- [Azure DevOps Pipelines Environments](https://learn.microsoft.com/en-us/azure/devops/pipelines/process/environments)
- [Azure Container Registry](https://learn.microsoft.com/en-us/azure/container-registry/)
- [Azure Key Vault Integration](https://learn.microsoft.com/en-us/azure/devops/pipelines/library/variable-groups?view=azure-devops&tabs=azure-portal)
- [AKS Deployment with Helm](https://learn.microsoft.com/en-us/azure/aks/quickstart-helm)

---

**Summary:**

- Use feature branch pipelines for validation only.
- Use environment-specific pipelines for build, push, and deploy.
- Store config in environment variables and secrets in Key Vault.
- Use resource groups for isolation.
- Automate deployment and monitoring for all environments.
