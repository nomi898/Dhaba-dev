# Dhaba – DevOps Deployment Project

## 1. Project Overview

Dhaba is a containerized application deployed using Docker, Amazon ECR, Kubernetes, GitHub Actions, and AWS EC2.

The project implements automated image builds, staging and production deployments, controlled access, rollback procedures, load testing, and operational health checks.

### Project Scope

- C9 – Automatic ECR credential refresh
- C10 – Restricted Kubernetes deployment permissions
- C11 – Automated staging deployment
- C12 – Production deployment with approval
- C13 – Manual rollback and Slack notifications
- C14 – Load testing
- C15 – Cluster, service, and database health checks

C16 and C17 are outside the agreed project scope and are not included.

---

## 2. Infrastructure

### AWS Infrastructure

| Component | Configuration |
|---|---|
| Cloud provider | AWS |
| Region | eu-north-1 |
| EC2 public IP | 51.20.193.202 |
| EC2 instance | i-0874e3bb71b14318a |
| Operating system | Ubuntu |
| Kubernetes | Minikube |
| Container runtime | containerd |
| Kubernetes version | v1.37.0 |
| Deployment user | dhaba-deploy |

### Kubernetes Cluster

The Minikube cluster runs three nodes on the EC2 host.

| Node | Internal IP | Role |
|---|---|---|
| minikube | 192.168.49.2 | Control plane |
| minikube-m02 | 192.168.49.3 | Worker |
| minikube-m03 | 192.168.49.4 | Worker |

**Important:** These are three Minikube nodes hosted on one EC2 instance. They do not represent three independent EC2 machines.

### Kubernetes Namespaces

| Namespace | Purpose |
|---|---|
| ops | Operational resources and ECR credential refresh |
| staging | Testing application releases |
| production | Production application |

---

## 3. Application Architecture

### Application Components

The application contains six main workloads:

1. web
2. menu-api
3. orders-api
4. kitchen-worker
5. stats-api
6. stats-nginx

Supporting infrastructure:

- PostgreSQL – StatefulSet
- Redis – StatefulSet

The kitchen-worker is a background worker and does not require a Kubernetes Service.

### Container Registry

Amazon Elastic Container Registry (ECR) stores the application images.

Repositories:

- dhaba-web
- dhaba-menu-api
- dhaba-orders-api
- dhaba-kitchen-worker
- dhaba-stats-api
- dhaba-stats-nginx

AWS account ID: 270254976774

AWS region: eu-north-1

ECR registry:

`270254976774.dkr.ecr.eu-north-1.amazonaws.com`

---

## 4. Domain Configuration

Two DuckDNS domains are configured.

| Environment | Domain |
|---|---|
| Production | devopsnktask.duckdns.org |
| Staging | devopsnktask-staging.duckdns.org |

Both domains resolve to:

`51.20.193.202`

### Traffic Flow

```text
                Internet
                    |
                    |
              DuckDNS Domains
                    |
                    v
              AWS EC2 Server
              51.20.193.202
                    |
                    v
                Nginx :80
                    |
                    v
          Minikube Ingress NodePort
                 :31742
                    |
                    v
              Kubernetes Ingress
                    |
          +---------+---------+
          |                   |
          v                   v
      Production            Staging
      Namespace             Namespace
          |                   |
          v                   v
       Web Service         Web Service
          |                   |
          v                   v
       Web Pods             Web Pods
```

Nginx uses the HTTP Host header to forward both domains to the Kubernetes Ingress.

### Nginx Configuration

File:

`/etc/nginx/sites-enabled/dhaba`

Existing configuration:

```nginx
server {
    listen 80;
    server_name devopsnktask.duckdns.org;

    location / {
        proxy_pass http://192.168.49.2:31742;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name devopsnktask-staging.duckdns.org;

    location / {
        proxy_pass http://192.168.49.2:31742;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Check Nginx configuration:

```bash
sudo nginx -t
```

Reload Nginx:

```bash
sudo systemctl reload nginx
```

---

## 5. C9 – Automatic ECR Pull Secret Refresh

### Purpose

ECR authentication tokens expire after 12 hours.

Kubernetes needs valid registry credentials to pull private application images.

We implemented a Kubernetes CronJob that refreshes the ECR pull secret every six hours.

### Files

```text
k8s/
└── ops/
    └── ecr-secret-refresh/
        ├── kustomization.yaml
        ├── rbac-cronjob.yaml
        └── refresh-ecr-secrets-k8s.py
```

Additional script:

```text
scripts/refresh-ecr-secrets.sh
```

### Implementation

- CronJob name: ecr-secret-refresh
- Namespace: ops
- Schedule: Every six hours
- ServiceAccount: ecr-secret-refresher
- Target namespaces: staging and production

CronJob schedule:

```yaml
schedule: "0 */6 * * *"
```

The Python script obtains an ECR authentication token using AWS CLI and updates the existing `ecr-pull-secret` in both application namespaces.

### Security

- EC2 uses an IAM role called DhabaEC2ECRRole.
- No permanent AWS access keys are stored in Kubernetes.
- The ServiceAccount has permission to update only the named ECR secret in staging and production.
- hostNetwork is enabled so the refresher can access EC2 instance metadata.
- IMDSv2 hop limit is configured to 2.

### Kustomize Configuration

`k8s/ops/ecr-secret-refresh/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - rbac-cronjob.yaml

configMapGenerator:
  - name: ecr-secret-refresh-script
    namespace: ops
    files:
      - refresh-ecr-secrets-k8s.py
```

### Test the CronJob

Create a test Job:

```bash
kubectl create job ecr-secret-refresh-test \
  --from=cronjob/ecr-secret-refresh \
  -n ops
```

Check logs:

```bash
kubectl logs job/ecr-secret-refresh-test -n ops
```

Expected output:

```text
staging: refreshed successfully (200)
production: refreshed successfully (200)
ECR credentials refreshed in both namespaces.
```

Delete the test Job:

```bash
kubectl delete job ecr-secret-refresh-test -n ops
```

---

## 6. C10 – Restricted Kubernetes Deployment Permissions

### Purpose

The GitHub deployment pipeline should not use cluster-admin permissions.

We created a dedicated Linux user and Kubernetes ServiceAccount.

### Linux Deployment User

Username:

`dhaba-deploy`

Repository:

`/home/dhaba-deploy/Dhaba-dev`

The deployment user does not have sudo or Docker group permissions.

Switch from ubuntu:

```bash
sudo su - dhaba-deploy
```

Go to the repository:

```bash
cd /home/dhaba-deploy/Dhaba-dev
```

### Kubernetes ServiceAccount

Name:

`dhaba-deployer`

Namespace:

`ops`

RBAC file:

`k8s/rbac/deployer.yaml`

The deployment identity has namespace-scoped permissions for the required deployment resources.

Permissions include:

- Deployments
- StatefulSets
- Services
- ConfigMaps
- Ingresses
- Pod inspection
- ReplicaSet inspection
- Limited access to postgres-secret
- Jobs and CronJobs in ops

It cannot delete namespaces or access the ECR pull secret.

### Apply RBAC

```bash
kubectl apply -f k8s/rbac/deployer.yaml
```

### Verify Permissions

Check deployment creation:

```bash
kubectl auth can-i create deployments -n staging
```

Check production deployment updates:

```bash
kubectl auth can-i patch deployments -n production
```

Check database secret access:

```bash
kubectl auth can-i get secret postgres-secret -n staging
```

Check that ECR credentials are protected:

```bash
kubectl auth can-i get secret ecr-pull-secret -n staging
```

Check namespace deletion restriction:

```bash
kubectl auth can-i delete namespaces
```

Expected results:

- Required deployment operations: yes
- ECR pull secret access: no
- Namespace deletion: no

### GitHub Secrets

The deployment workflow uses:

- DHABA_DEPLOY_SSH_KEY

The deployment user authenticates to EC2 using SSH.

---

## 7. C11 – Automated Staging Deployment

### Workflow

`.github/workflows/deploy-staging.yml`

### Trigger

The staging deployment starts after the Build and Push Images to ECR workflow completes successfully on the main branch.

### Deployment Process

1. GitHub Actions waits for the image build workflow.
2. The workflow connects to EC2 using SSH.
3. It checks out the exact commit and extracts Kubernetes manifests and scripts.
4. It refreshes ECR credentials using the existing CronJob.
5. It applies the staging Kubernetes overlay.
6. It deploys the required application images.
7. It waits for all six Deployments to roll out.
8. It runs the web health endpoint smoke test.
9. It runs the load test.
10. It creates an immutable release ConfigMap after successful verification.

### Verify Staging

```bash
kubectl get pods -n staging
kubectl get deployments -n staging
kubectl get services -n staging
kubectl get ingress -n staging
```

Check rollout:

```bash
kubectl rollout status deployment/web -n staging
```

Repeat for the other application Deployments if required.

### Health Check

```bash
curl -i http://devopsnktask-staging.duckdns.org/health
```

Expected response:

```json
{"status":"ok","service":"web"}
```

---

## 8. C12 – Production Deployment With Approval

### Workflow

`.github/workflows/deploy-production.yml