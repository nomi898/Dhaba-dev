## C9 – Automatic ECR Image Pull Secret Refresh

### Overview

Amazon Elastic Container Registry (ECR) uses temporary authentication tokens to allow Kubernetes Pods to pull private Docker images.

These authentication tokens are valid for approximately **12 hours**. If the token expires, Kubernetes may fail to pull images from ECR, resulting in `ImagePullBackOff` errors.

To prevent this, Dhaba implements an automated Kubernetes CronJob that refreshes ECR credentials every 6 hours.

### Architecture

The refresh process works as follows:

1. Kubernetes CronJob runs every 6 hours in the `ops` namespace.
2. A Job creates a temporary Pod using the `amazon/aws-cli` image.
3. The Pod accesses the EC2 instance's IAM role through the EC2 Instance Metadata Service (IMDSv2).
4. AWS CLI obtains a temporary ECR authentication token.
5. The refresh script communicates with the Kubernetes API.
6. The script updates the existing `ecr-pull-secret` in both `staging` and `production` namespaces.
7. Kubernetes uses the refreshed credentials when pulling private images from ECR.

### Configuration

**CronJob details**

| Setting | Value |
|---|---|
| CronJob name | `ecr-secret-refresh` |
| Namespace | `ops` |
| Schedule | `0 */6 * * *` |
| Container image | `amazon/aws-cli:latest` |
| AWS Region | `eu-north-1` |
| AWS Authentication | EC2 IAM Role |
| Authentication method | IMDSv2 |
| Target namespaces | `staging`, `production` |
| Secret name | `ecr-pull-secret` |

The CronJob uses `hostNetwork: true` and `dnsPolicy: ClusterFirstWithHostNet` to access the EC2 metadata service from the Pod.

### IAM Authentication

The CronJob uses the EC2 IAM role:

`DhabaEC2ECRRole`

This role provides the permissions required to authenticate with Amazon ECR.

The implementation does not store permanent AWS access keys or secret access keys inside Kubernetes Secrets or environment variables.

Instead, it obtains temporary credentials through IMDSv2.

### Kubernetes RBAC

A dedicated ServiceAccount named `ecr-secret-refresher` runs the refresh process in the `ops` namespace.

It has restricted permissions through Kubernetes Roles and RoleBindings.

Its permissions are limited to:

- Accessing the `ecr-pull-secret` Secret in the `staging` namespace.
- Updating and patching that Secret.
- Accessing the `ecr-pull-secret` Secret in the `production` namespace.
- Updating and patching that Secret.

It does not require cluster-admin permissions.

### Deployment Pipeline Integration

The staging and production GitHub Actions workflows trigger the ECR refresh CronJob manually before deploying application workloads.

They create a Job from the existing CronJob:

```bash
kubectl create job "$JOB_NAME" \
  --from=cronjob/ecr-secret-refresh \
  -n ops
```

The workflows wait for the Job to complete successfully before continuing with deployment.

This ensures that the ECR credentials are refreshed before Kubernetes attempts to pull the new application images.

The deployment pipeline uses the separate `dhaba-deployer` ServiceAccount, while the refresh Job uses `ecr-secret-refresher`.

This separation prevents the deployment pipeline from needing direct permission to manage ECR pull credentials.

### Manual Testing

To manually trigger the refresh process:

```bash
kubectl create job ecr-secret-refresh-test \
  --from=cronjob/ecr-secret-refresh \
  -n ops
```

Check the Job status:

```bash
kubectl get jobs -n ops
```

Check the logs:

```bash
kubectl logs job/ecr-secret-refresh-test -n ops
```

Expected successful output:

```text
staging: refreshed successfully (200)
production: refreshed successfully (200)
ECR credentials refreshed in both namespaces.
```

Clean up the test Job:

```bash
kubectl delete job ecr-secret-refresh-test -n ops
```

### Security Considerations and Risks

1. **No permanent AWS credentials:** The implementation uses the EC2 IAM role instead of storing permanent AWS credentials in Kubernetes.

2. **IMDSv2 protection:** The EC2 metadata service uses IMDSv2 to obtain temporary credentials. The instance metadata hop limit must permit access from the host-networked Pod.

3. **Host networking:** The CronJob requires host networking to access the EC2 metadata service. This increases the Pod's network access, so the workload must remain restricted and its image and permissions should be reviewed.

4. **Least privilege:** The refresh ServiceAccount only receives permission to update the specifically named ECR pull Secret in the two application namespaces.

5. **Token expiration:** ECR authentication tokens are valid for approximately 12 hours. Refreshing every 6 hours reduces the risk of token expiration during deployments.

6. **Image security:** The CronJob uses the AWS CLI container image. Its version should be pinned and updated regularly rather than relying indefinitely on the `latest` tag.

### Benefits

- Prevents expired ECR credentials from interrupting deployments.
- Removes the need for manually refreshing image pull secrets.
- Avoids storing permanent AWS credentials in Kubernetes.
- Uses a dedicated ServiceAccount with restricted permissions.
- Supports both staging and production environments.
- Integrates directly with the GitHub Actions deployment pipeline.