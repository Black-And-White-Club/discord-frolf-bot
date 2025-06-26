# GCP Container Registry Setup Guide

This guide helps you set up Google Cloud Container Registry for your Discord Frolf Bot.

## Prerequisites

1. **Google Cloud Project**: You'll need a GCP project to store your containers
2. **Service Account**: A service account with appropriate permissions
3. **GitHub Secrets**: Repository secrets for authentication

## Step 1: Create a New GCP Project (Optional)

If you want a dedicated project for your containers:

```bash
# Create a new project
gcloud projects create discord-frolf-containers --name="Discord Frolf Containers"

# Set as your current project
gcloud config set project discord-frolf-containers

# Enable required APIs
gcloud services enable containerregistry.googleapis.com
```

## Step 2: Create a Service Account

```bash
# Create service account
gcloud iam service-accounts create github-actions \
    --description="Service account for GitHub Actions" \
    --display-name="GitHub Actions"

# Grant necessary permissions
gcloud projects add-iam-policy-binding discord-frolf-containers \
    --member="serviceAccount:github-actions@discord-frolf-containers.iam.gserviceaccount.com" \
    --role="roles/storage.admin"

# Create and download service account key
gcloud iam service-accounts keys create ~/github-actions-key.json \
    --iam-account=github-actions@discord-frolf-containers.iam.gserviceaccount.com
```

## Step 3: Configure GitHub Secrets

In your GitHub repository, go to Settings → Secrets and variables → Actions, and add:

1. **`GCP_PROJECT_ID`**: Your GCP project ID (e.g., `discord-frolf-containers`)
2. **`GCP_SERVICE_ACCOUNT_KEY`**: Contents of the `github-actions-key.json` file

## Step 4: Test Your Setup

After pushing to main branch, your workflow will:

1. ✅ Build your Docker image
2. ✅ Scan for vulnerabilities with Trivy
3. ✅ Push to `gcr.io/your-project-id/discord-frolf-bot:short-sha`

## Container Registry URL Format

Your images will be available at:
```
gcr.io/discord-frolf-containers/discord-frolf-bot:latest
gcr.io/discord-frolf-containers/discord-frolf-bot:abc1234
```

## Local Development

To pull images locally:

```bash
# Authenticate with gcloud
gcloud auth login
gcloud auth configure-docker gcr.io

# Pull your image
docker pull gcr.io/discord-frolf-containers/discord-frolf-bot:latest
```

## Deployment

### Option 1: Docker Compose
```yaml
services:
  discord-frolf-bot:
    image: gcr.io/discord-frolf-containers/discord-frolf-bot:latest
    # ... other configuration
```

### Option 2: Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: discord-frolf-bot
spec:
  replicas: 1
  selector:
    matchLabels:
      app: discord-frolf-bot
  template:
    metadata:
      labels:
        app: discord-frolf-bot
    spec:
      containers:
      - name: discord-frolf-bot
        image: gcr.io/discord-frolf-containers/discord-frolf-bot:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: discord-frolf-secrets
              key: database-url
```

## Cost Considerations

- Container Registry storage is charged based on the amount of data stored
- Consider setting up lifecycle policies to clean up old images
- Each image version (tag) is stored separately

## Security Best Practices

1. ✅ Use least-privilege service accounts
2. ✅ Regularly rotate service account keys
3. ✅ Enable vulnerability scanning (included in workflow)
4. ✅ Use specific image tags in production (not `latest`)

## Troubleshooting

### Authentication Issues
```bash
# Check if you're authenticated
gcloud auth list

# Re-configure Docker
gcloud auth configure-docker gcr.io
```

### Permission Issues
```bash
# Check service account permissions
gcloud projects get-iam-policy discord-frolf-containers \
    --flatten="bindings[].members" \
    --format="table(bindings.role)" \
    --filter="bindings.members:github-actions@discord-frolf-containers.iam.gserviceaccount.com"
```
