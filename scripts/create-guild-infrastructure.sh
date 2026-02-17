#!/bin/bash
# Script to create per-guild infrastructure

set -euo pipefail

GUILD_ID=${1:-}
GUILD_NAME=${2:-}
DB_PASSWORD=${DB_PASSWORD:-}
BOT_IMAGE_REF=${BOT_IMAGE_REF:-frolf-bot:latest}
BACKEND_IMAGE_REF=${BACKEND_IMAGE_REF:-frolf-backend:latest}
MIGRATIONS_IMAGE_REF=${MIGRATIONS_IMAGE_REF:-frolf-migrations:latest}

if [ -z "$GUILD_ID" ]; then
    echo "Usage: $0 <guild_id> <guild_name>"
    exit 1
fi

if [ -z "$DB_PASSWORD" ]; then
    echo "Error: DB_PASSWORD environment variable is required"
    exit 1
fi

for image_ref in "$BOT_IMAGE_REF" "$BACKEND_IMAGE_REF" "$MIGRATIONS_IMAGE_REF"; do
    if [[ "$image_ref" == *":latest" ]]; then
        echo "Error: image ref '$image_ref' uses mutable ':latest'. Use a pinned tag or digest."
        exit 1
    fi
done

# Create namespace for this guild
kubectl create namespace "frolf-guild-${GUILD_ID}"

# Create dedicated PostgreSQL instance
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: frolf-guild-${GUILD_ID}
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
      guild: ${GUILD_ID}
  template:
    metadata:
      labels:
        app: postgres
        guild: ${GUILD_ID}
    spec:
      containers:
      - name: postgres
        image: postgres:15
        env:
        - name: POSTGRES_DB
          value: frolf_${GUILD_ID}
        - name: POSTGRES_USER
          value: frolf_user_${GUILD_ID}
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "200m"
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 20Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: frolf-guild-${GUILD_ID}
spec:
  selector:
    app: postgres
    guild: ${GUILD_ID}
  ports:
  - port: 5432
    targetPort: 5432
---
# Dedicated Discord Bot instance
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frolf-bot
  namespace: frolf-guild-${GUILD_ID}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: frolf-bot
      guild: ${GUILD_ID}
  template:
    metadata:
      labels:
        app: frolf-bot
        guild: ${GUILD_ID}
    spec:
      containers:
      - name: discord-bot
        image: ${BOT_IMAGE_REF}
        env:
        - name: INFRASTRUCTURE_MODE
          value: "single-server"
        - name: GUILD_ID
          value: "${GUILD_ID}"
        - name: DATABASE_URL
          value: "postgresql://frolf_user_${GUILD_ID}:${DB_PASSWORD}@postgres:5432/frolf_${GUILD_ID}"
        - name: NATS_URL
          value: "nats://nats:4222"
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "100m"
      - name: backend-api
        image: ${BACKEND_IMAGE_REF}
        env:
        - name: GUILD_ID
          value: "${GUILD_ID}"
        - name: DATABASE_URL
          value: "postgresql://frolf_user_${GUILD_ID}:${DB_PASSWORD}@postgres:5432/frolf_${GUILD_ID}"
        - name: NATS_URL
          value: "nats://nats:4222"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
---
# Per-guild NATS (optional, or use shared)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nats
  namespace: frolf-guild-${GUILD_ID}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nats
      guild: ${GUILD_ID}
  template:
    metadata:
      labels:
        app: nats
        guild: ${GUILD_ID}
    spec:
      containers:
      - name: nats
        image: nats:2.9
        ports:
        - containerPort: 4222
        resources:
          requests:
            memory: "64Mi"
            cpu: "25m"
          limits:
            memory: "128Mi"
            cpu: "50m"
EOF

# Run database migrations for this guild
kubectl run migration-job-${GUILD_ID} \
  --namespace=frolf-guild-${GUILD_ID} \
  --image="${MIGRATIONS_IMAGE_REF}" \
  --env="DATABASE_URL=postgresql://frolf_user_${GUILD_ID}:${DB_PASSWORD}@postgres:5432/frolf_${GUILD_ID}" \
  --restart=Never \
  --rm -i

echo "Guild ${GUILD_ID} infrastructure created successfully!"
echo "Namespace: frolf-guild-${GUILD_ID}"
echo "Database: frolf_${GUILD_ID}"
