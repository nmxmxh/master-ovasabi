This project is well-structured for containerized deployment, making it a great candidate for Azure
Kubernetes Service (AKS). Given your `Makefile`, various `Dockerfile`s, and `nginx.conf`, the setup
clearly points to a microservices architecture. Let's break down a thorough deployment strategy for
AKS, excluding testing for now.

The core services identified for deployment are:

1.  **`master-ovasabi` (Main Application):** The primary Go application.
2.  **`nexus`:** The orchestration service.
3.  **`media-streaming`:** The media streaming service.
4.  **`ws-gateway`:** The WebSocket gateway.
5.  **`frontend` (Nginx + WASM):** The static frontend served by Nginx, including the WASM module.
6.  **Data Stores (`PostgreSQL`, `Redis`):** While they run as Docker containers locally, for a
    robust AKS setup, these should typically be managed Azure services.

### Azure Kubernetes Service (AKS) Deployment Strategy

Here's a step-by-step guide for setting up and deploying your project to AKS:

#### 1. Prerequisites

Before you begin, ensure you have the following installed and configured:

- **Azure CLI:** For interacting with Azure.
- **`kubectl`:** The Kubernetes command-line tool.
- **`helm`:** The Kubernetes package manager, used for deploying your application charts.
- **Docker Desktop (or similar):** For local image building.

#### 2. Azure Setup: AKS Cluster and Azure Container Registry (ACR)

First, you'll need an AKS cluster and an ACR instance to store your Docker images.

**2.1. Create a Resource Group:**

```bash
az group create --name ovasabi-rg --location eastus
```

**2.2. Create Azure Container Registry (ACR):**

```bash
az acr create --resource-group ovasabi-rg --name ovasabiregistry --sku Basic --admin-enabled true
```

**2.3. Create Azure Kubernetes Service (AKS) Cluster:**

```bash
az aks create \
  --resource-group ovasabi-rg \
  --name ovasabi-aks \
  --node-count 2 \
  --enable-managed-identity \
  --generate-ssh-keys \
  --attach-acr ovasabiregistry
```

- `--node-count`: Start with 2 nodes; scale as needed.
- `--enable-managed-identity`: Recommended for secure access to other Azure resources.
- `--attach-acr`: Directly integrates your ACR with AKS for easy image pulling.

**2.4. Get AKS Credentials:**

```bash
az aks get-credentials --resource-group ovasabi-rg --name ovasabi-aks
```

This configures `kubectl` to connect to your new AKS cluster.

#### 3. Build and Push Docker Images to ACR

Your `Makefile` has a `docker-build` target that uses `docker-compose` to build all images. You'll
use this, then tag and push each required image to your ACR.

```bash
# Log in to your ACR
az acr login --name ovasabiregistry

# Step 1: Build all images locally using docker-compose
# This will build images defined in docker-compose.yml (app, nexus, media-streaming, ws-gateway, nginx)
make docker-build

# Step 2: Tag and push each image to ACR
# Replace 'ovasabi/master-ovasabi' with the actual image name from your docker-compose.yml output
# and 'ovasabiregistry.azurecr.io' with your ACR login server.
# You might need to inspect the output of `docker images` to get the exact local image names.

ACR_LOGIN_SERVER="ovasabiregistry.azurecr.io" # Replace with your ACR login server

# Main Application
docker tag ovasabi/master-ovasabi:latest $ACR_LOGIN_SERVER/ovasabi/master-ovasabi:$(VERSION)
docker push $ACR_LOGIN_SERVER/ovasabi/master-ovasabi:$(VERSION)

# Nexus Service
docker tag ovasabi/nexus:latest $ACR_LOGIN_SERVER/ovasabi/nexus:$(VERSION)
docker push $ACR_LOGIN_SERVER/ovasabi/nexus:$(VERSION)

# Media Streaming Service
docker tag ovasabi/media-streaming:latest $ACR_LOGIN_SERVER/ovasabi/media-streaming:$(VERSION)
docker push $ACR_LOGIN_SERVER/ovasabi/media-streaming:$(VERSION)

# WebSocket Gateway
docker tag ovasabi/ws-gateway:latest $ACR_LOGIN_SERVER/ovasabi/ws-gateway:$(VERSION)
docker push $ACR_LOGIN_SERVER/ovasabi/ws-gateway:$(VERSION)

# Frontend Nginx (assuming your docker-compose.yml builds this from `deployments/docker/frontend/Dockerfile` or similar)
# If Nginx is built from a custom Dockerfile within `deployments/docker/frontend/`, adjust this.
# Otherwise, you might use a generic Nginx image from Docker Hub and configure it via ConfigMap.
docker tag ovasabi/nginx:latest $ACR_LOGIN_SERVER/ovasabi/nginx:$(VERSION)
docker push $ACR_LOGIN_SERVER/ovasabi/nginx:$(VERSION)

# For WASM: The Dockerfile.wasm likely builds the WASM binary. This binary is then copied to `frontend/public`.
# The Nginx container will serve this as a static asset. You typically don't deploy the WASM binary itself as a separate pod,
# but rather serve it from your frontend Nginx.
```

#### 4. Kubernetes Manifests (Helm Charts) Adaptation

Your `Makefile` already has `helm` commands and refers to `deployments/kubernetes`. This is the
ideal way to deploy. You'll need to create or adapt Helm charts for each of your services.

**4.1. Helm Chart Structure:**

For each service (e.g., `master-ovasabi`, `nexus`, `media-streaming`, `ws-gateway`,
`frontend-nginx`), you would typically have a Helm chart directory like
`deployments/kubernetes/charts/master-ovasabi/`.

Inside each chart, you'd find:

- `Chart.yaml`: Metadata about the chart.
- `values.yaml`: Default configurable values (e.g., image name, tag, replica count, ports).
- `templates/`: Kubernetes YAML definitions.
  - `deployment.yaml`: Defines the Pods and ReplicaSet.
  - `service.yaml`: Defines how to access the Pods internally.
  - `_helpers.tpl`: Reusable YAML snippets.
  - `ingress.yaml` (optional, for frontend): Defines external access.

**4.2. Example: `master-ovasabi` Helm Chart (`deployments/kubernetes/charts/master-ovasabi/`)**

- **`values.yaml`**:

  ```yaml
  # deployments/kubernetes/charts/master-ovasabi/values.yaml
  image:
    repository: ovasabiregistry.azurecr.io/ovasabi/master-ovasabi
    tag: 'latest' # This will be overridden by `helm upgrade --set image.tag=$(VERSION)`
    pullPolicy: IfNotPresent

  service:
    type: ClusterIP
    port: 50051 # gRPC port
    httpPort: 9090 # HTTP/REST port

  # Environment variables (from .env)
  env:
    DB_USER: 'yourdbuser'
    DB_PASSWORD: 'yourdbpassword'
    DB_NAME: 'master_ovasabi'
    DB_HOST: 'postgres-svc' # Or your Azure PostgreSQL FQDN
    REDIS_HOST: 'redis-svc' # Or your Azure Cache for Redis FQDN
    GIN_MODE: 'release'
  ```

- **`templates/deployment.yaml`**:

  ```yaml
  # deployments/kubernetes/charts/master-ovasabi/templates/deployment.yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: {{ include "master-ovasabi.fullname" . }}
    labels:
      {{- include "master-ovasabi.labels" . | nindent 4 }}
  spec:
    replicas: {{ .Values.replicaCount }}
    selector:
      matchLabels:
        {{- include "master-ovasabi.selectorLabels" . | nindent 6 }}
    template:
      metadata:
        labels:
          {{- include "master-ovasabi.selectorLabels" . | nindent 8 }}
      spec:
        containers:
          - name: {{ .Chart.Name }}
            image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
            imagePullPolicy: {{ .Values.image.pullPolicy }}
            ports:
              - name: grpc
                containerPort: {{ .Values.service.port }}
              - name: http
                containerPort: {{ .Values.service.httpPort }}
            env:
              {{- range $key, $value := .Values.env }}
              - name: {{ $key }}
                value: {{ $value | quote }}
              {{- end }}
            # Liveness and Readiness probes from Dockerfile comments
            livenessProbe:
              exec:
                command: ["/usr/local/bin/grpc_health_probe", "-addr=:{{ .Values.service.port }}"]
              initialDelaySeconds: 10
              periodSeconds: 10
            readinessProbe:
              exec:
                command: ["/usr/local/bin/grpc_health_probe", "-addr=:{{ .Values.service.port }}"]
              initialDelaySeconds: 5
              periodSeconds: 5
            # ... other resource limits and requests
  ```

- **`templates/service.yaml`**:
  ```yaml
  # deployments/kubernetes/charts/master-ovasabi/templates/service.yaml
  apiVersion: v1
  kind: Service
  metadata:
    name: { { include "master-ovasabi.fullname" . } }
    labels: { { - include "master-ovasabi.labels" . | nindent 4 } }
  spec:
    type: { { .Values.service.type } }
    ports:
      - port: { { .Values.service.port } }
        targetPort: grpc
        protocol: TCP
        name: grpc
      - port: { { .Values.service.httpPort } }
        targetPort: http
        protocol: TCP
        name: http
    selector: { { - include "master-ovasabi.selectorLabels" . | nindent 4 } }
  ```

**4.3. Frontend Nginx and WASM Deployment:**

- **`Dockerfile.wasm`:** This builds the WASM binary. The `Makefile` copies this to
  `frontend/public/`.
- **`nginx.conf`:** This config file needs to be mounted into an Nginx container.
- **Frontend Assets:** The contents of `frontend/public/` (including the WASM binary) need to be
  served.

You can create a Helm chart for `frontend-nginx` that:

1.  Uses an `nginx` Docker image (built locally as `ovasabi/nginx` and pushed to ACR).
2.  Mounts `nginx.conf` as a `ConfigMap`.
3.  Mounts your `frontend/public` assets (e.g., from an `initContainer` that copies from a volume,
    or baked into your `ovasabi/nginx` image).
4.  Exposes Nginx on port 80/443.
5.  Defines an `Ingress` resource to expose it externally via an Ingress Controller (like Nginx
    Ingress Controller, which you'd deploy separately in AKS).

**4.4. Database and Redis (Managed Services Recommendation):**

Instead of running PostgreSQL and Redis inside AKS, it's highly recommended for robustness and ease
of management to use Azure's managed services:

- **Azure Database for PostgreSQL:** Create a flexible server instance.
- **Azure Cache for Redis:** Create a Redis cache instance.

You'll then update the `DB_HOST` and `REDIS_HOST` environment variables in your Kubernetes
Deployments (or Helm `values.yaml`) to point to the FQDNs of these Azure managed services. Securely
inject credentials using Kubernetes Secrets.

#### 5. Deploying with Helm

Once your Helm charts are prepared, you can deploy your application.

**5.1. Install Nginx Ingress Controller (if not already present in AKS):**

```bash
# Add the Nginx Ingress Controller Helm repository
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

# Install the Nginx Ingress Controller
helm install nginx-ingress ingress-nginx/ingress-nginx \
  --create-namespace --namespace ingress-nginx \
  --set controller.service.loadBalancerIP="" # Azure will assign a public IP
```

**5.2. Deploy Each Service using Helm:**

From your `deployments/kubernetes/` directory (or wherever your chart is):

```bash
# Deploy master-ovasabi
helm install master-ovasabi deployments/kubernetes/charts/master-ovasabi \
  --namespace ovasabi \
  --set image.tag=$(VERSION) \
  --set env.DB_PASSWORD="your_secure_db_password" # Use actual secure methods later (Key Vault)

# Deploy nexus
helm install nexus deployments/kubernetes/charts/nexus \
  --namespace ovasabi \
  --set image.tag=$(VERSION)

# Deploy media-streaming
helm install media-streaming deployments/kubernetes/charts/media-streaming \
  --namespace ovasabi \
  --set image.tag=$(VERSION)

# Deploy ws-gateway
helm install ws-gateway deployments/kubernetes/charts/ws-gateway \
  --namespace ovasabi \
  --set image.tag=$(VERSION)

# Deploy frontend (Nginx)
helm install ovasabi-frontend deployments/kubernetes/charts/frontend-nginx \
  --namespace ovasabi \
  --set image.tag=$(VERSION) # If you built a custom Nginx image
  # If using a generic Nginx image, you'd only set configmap/volume mounts.
```

Your `Makefile` already provides `helm-install` and `helm-upgrade` targets, which would wrap these
commands. For example:

```makefile
# Example of how your Makefile's helm-install could look (simplified)
helm-install:
	$(KUBECTL) create namespace $(K8S_NAMESPACE) --dry-run=client -o yaml | $(KUBECTL) apply -f -
	helm install master-ovasabi deployments/kubernetes/charts/master-ovasabi -n $(K8S_NAMESPACE) --set image.tag=$(VERSION)
	helm install nexus deployments/kubernetes/charts/nexus -n $(K8S_NAMESPACE) --set image.tag=$(VERSION)
	# ... similar for other services
	helm install ovasabi-frontend deployments/kubernetes/charts/frontend-nginx -n $(K8S_NAMESPACE) --set image.tag=$(VERSION)
```

#### 6. Post-Deployment Checks

- **Check Pods:** `kubectl get pods -n ovasabi` (ensure all are Running)
- **Check Services:** `kubectl get svc -n ovasabi`
- **Check Deployments:** `kubectl get deployments -n ovasabi`
- **Check Ingress:** `kubectl get ingress -n ovasabi` (get the public IP/hostname of your Nginx
  Ingress Controller)
- **View Logs:** `kubectl logs -f deployment/master-ovasabi -n ovasabi`

#### 7. Important AKS Considerations for Production

- **Secrets Management:** Never hardcode sensitive information. Use Kubernetes Secrets, preferably
  backed by Azure Key Vault using CSI Driver for Secret Store.
- **Networking:**
  - **VNet Integration:** For private network access to Azure managed databases.
  - **Azure CNI:** For advanced networking and more IP addresses per pod.
- **Scaling:**
  - **Horizontal Pod Autoscaler (HPA):** To automatically scale pods based on CPU/memory usage.
  - **Cluster Autoscaler:** To automatically scale AKS nodes.
- **Monitoring and Logging:**
  - Enable Azure Monitor for AKS to collect metrics and logs.
  - Integrate with Prometheus and Grafana for deeper insights.
  - Use Azure Log Analytics for centralized logging.
- **Resource Limits and Requests:** Define CPU and memory limits/requests for all containers in your
  deployments for better resource management and stability.
- **Health Probes:** Ensure all deployments have robust `liveness` and `readiness` probes for
  reliable restarts and traffic routing, as seen in the `Dockerfile` comments.
- **Ingress Controller:** Utilize an Ingress Controller (like Nginx Ingress or Azure Application
  Gateway Ingress Controller - AGIC) for external HTTP/S routing, SSL termination, and load
  balancing.
- **External IP:** Your `nginx.conf` and `Ingress` setup will likely expose your frontend and API
  gateway. The Ingress Controller will provision a public Load Balancer IP.
- **CI/CD Pipeline (Azure DevOps):** While you're skipping tests for now, for a full production
  setup, integrate these Helm deployments into an Azure DevOps pipeline to automate the build, image
  push, and deployment process upon code changes.

This comprehensive approach leverages your project's existing containerization and Helm readiness,
adapting it for a robust and scalable Azure Kubernetes environment.
