# Reliance Infosystems | Migration Questionnaire (Testing/Minimal Specs)

This document provides answers to the Reliance Infosystems migration questionnaire for application
hosting on Azure, focusing on the smallest specifications for testing in different environments.

---

## 1. Current Database Environment

**1.1 What types of databases are you currently using?**

- PostgreSQL (primary for all services)

**1.2 What are the versions of your databases?**

- PostgreSQL 15 (testing)

**1.3 What are the configurations of your databases?**

- The database schema includes advanced full-text search and similarity functions (trigram,
  GIN/GTRGM, etc.) and relies on custom triggers for search vector updates.
- The schema is moderately complex, with 44 tablesv(currently) and 38 custom functions (currently).

**1.4 Are there any specific database features or configurations that are critical to your
applications?**

- Trigram/pg_trgm functions (e.g., gin_extract_query_trgm, similarity, word_similarity, etc.)
- JSONB support
- Full-text search (FTS)
- Extensions: pg_trgm, uuid-ossp
- UUID functions (e.g., uuid_generate_v1, uuid_generate_v4, etc.)
- Search vector update triggers (e.g., update_campaign_search_vector, etc.)

**1.5 Do you prefer to use Azure-managed database services like Azure SQL Database or Azure Database
for PostgreSQL?**

- Yes, Azure Database for PostgreSQL (preferred for managed, minimal test setup)

---

## 2. Performance Metrics

**2.1 What are the current performance metrics for your databases?**

- Query response times: < 100ms (test)
- Throughput: Low (test data only)

**2.2 Do you have any performance baselines or benchmarks for your databases?**

- No formal baselines; test environment only

**2.3 Are there any known performance bottlenecks or issues?**

- None in test environment

**2.4 How do you currently monitor database and application performance, and are you interested in
using Azure Monitor or Container Insights for AKS to track performance metrics?**

- Basic logging only (test)
- Interested in Azure Monitor/Container Insights for production

---

## 3. Size and Complexity

**3.1 What is the total size of your databases?**

- < 2 GB (test data)

**3.2 How many tables, stored procedures, and triggers are in your databases?**

- 44 tables
- 38 custom functions (used for full-text search, similarity, UUID generation, and search vector
  updates)
- Minimal triggers (primarily for updating search vectors; no stored procedures)

**3.3 Do you have any complex database schemas or dependencies?**

- The schema is moderately complex, with 44 normalized tables, extensive use of foreign keys, and
  several JSONB fields for flexible metadata storage.
- Advanced full-text search is implemented using GIN/GTRGM indexes and custom functions from the
  pg_trgm extension.
- There are cross-entity relationships (e.g., via a master table), and some tables use triggers to
  maintain search vectors for efficient querying.
- The database depends on the pg_trgm and uuid-ossp extensions, as well as a set of custom functions
  for similarity search and UUID generation.
- No stored procedures are used in the test environment.

**3.4 What is the growth rate of your databases?**

- Negligible in test (< 100 MB/month)

**3.5 Are there specific storage requirements for your databases on AKS, such as persistent volume
needs or integration with Azure Blob Storage?**

- Minimal persistent volume (10 GB for test)
- Minimal Blob Storage integration for test

---

## 4. Servers and Applications

**4.1 How many servers and applications are you planning to migrate?**

- 1 application server, 1 database server (test)

**4.2 What is the role of each server?**

- 1 application server (app service: runs backend and frontend)
- 1 database server (postgres service: PostgreSQL 15)
- 1 cache server (redis service: Redis 8)
- 1 translation server (libretranslate for machine translation, optional for core app)
- 1 migration/init job (migrate and postgres-init, used for setup, not persistent servers)

**4.3 What operating systems are running on these servers?**

- All containers run on Linux (Alpine for Postgres, Debian/Ubuntu base for app, Redis official
  image, etc.)
- Host OS for Docker: Ubuntu Linux 22.04 LTS (test)

**4.4 What is the size (CPU, RAM, storage) of your database and application servers?**

- Application: 2 vCPU, 4 GB RAM, 20 GB storage
- Database: 2 vCPU, 4 GB RAM, 20 GB storage
- Cache/Translation: Not resource-constrained in compose; use defaults or adjust as needed

**4.5 How do you currently access the servers?**

- SSH (test)
- Exposed ports for app, database, and Redis for local/dev access

**4.6 Do you require SSL VPN for secure access?**

- Not required for test

**4.7 Do you plan to use Azure DevOps to automate the build, test, and deployment of your
applications to AKS?**

- Yes, for CI/CD

---

## 5. Application and Database Dependencies

**5.1 Which applications are integrated with the database?**

- All backend services (user, campaign, analytics, etc.)

**5.2 Do any applications have dependencies on the current database environment?**

- Yes, direct DB access and specific schema requirements

**5.3 What programming languages and frameworks are your applications built with?**

- Go (Golang)

**5.4 Are there any third-party integrations or APIs that depend on your current database setup?**

- None

**5.5 Will you need to configure Kubernetes secrets in AKS to manage database credentials securely,
potentially integrating with Azure Key Vault?**

- Yes, for production; basic secrets for test

---

## 6. Monitoring and Security

**6.1 Do you require a monitoring solution for your databases and applications? If yes, are you
interested in using Azure Monitor, Container Insights, or Microsoft Defender for Containers to
monitor your AKS clusters?**

- Not required for test; interested for production

**6.2 Is there a firewall installed in your environment? If yes, please specify the make and
model.**

- No firewall in test environment

---

## 7. Migration Strategy and Downtime Recovery

**7.1 Do you have a staging/QA environment for testing the migration?**

- None.

**7.2 What are your acceptable downtime requirements and constraints for the migration?**

- Flexible for test; < 1 hour acceptable

**7.3 Are you interested in using Azure DevOps pipelines to automate the deployment of your
applications to AKS?**

- Yes, for future automation
