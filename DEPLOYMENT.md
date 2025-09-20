# Deployment Guide

This guide covers deploying the Discord Frolf Bot using modern container practices with multi-pod, event-driven architecture.

## Prerequisites

- Docker and Docker Compose installed
- Discord bot token and application configured
- NATS/JetStream server for event processing
- PostgreSQL database for backend storage
- (Optional) Google Cloud Platform account for container registry
- (Optional) Kubernetes cluster for production deployment

## Architecture Overview

The bot supports three deployment modes:

1. **Standalone Mode**: Single pod handling both Discord and backend processing
2. **Gateway Mode**: Dedicated pod for Discord interactions (exactly 1 replica)
3. **Worker Mode**: Scalable pods for backend event processing (N replicas)

## Configuration Options

### Runtime Configuration
The bot dynamically loads guild configurations at runtime - no config file restarts required.

### Environment Variables

**Required:**
- `DISCORD_TOKEN`: Discord bot token
- `NATS_URL`: NATS server URL (default: nats://localhost:4222)

**Optional:**
- `BOT_MODE`: `standalone`, `gateway`, or `worker` (default: standalone)
- `DISCORD_GUILD_ID`: Single guild ID (for development/testing)
- `DATABASE_URL`: PostgreSQL connection string
- `ENVIRONMENT`: Environment name for queue isolation (default: development)
- `METRICS_ADDRESS`: Metrics server address (default: :8080)
- `LOKI_URL`: Loki logging endpoint
- `TEMPO_ENDPOINT`: Tempo tracing endpoint

## Local Development

1. Start infrastructure services:
   ```bash
   docker-compose up -d postgres nats
   ```

2. Run in standalone mode:
   ```bash
   # Option 1: Using Go directly
   export DISCORD_TOKEN=your_token
   go run main.go

   # Option 2: Using Docker
   docker run -e DISCORD_TOKEN=your_token \
              -e NATS_URL=nats://host.docker.internal:4222 \
              discord-frolf-bot:latest
   ```

3. For testing multi-pod architecture locally:
   ```bash
   # Terminal 1: Gateway mode
   export BOT_MODE=gateway
   export DISCORD_TOKEN=your_token
   go run main.go

   # Terminal 2: Worker mode
   export BOT_MODE=worker
   go run main.go
   ```

## Production Deployment

### Option 1: Docker Compose (Recommended for small deployments)

Create a `docker-compose.prod.yml`:

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: discord_frolf_bot
      POSTGRES_USER: frolf_user
      POSTGRES_PASSWORD: secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  nats:
    image: nats:latest
    command: ["--jetstream"]
    volumes:
      - nats_data:/data

  gateway:
    image: your-registry/discord-frolf-bot:latest
    environment:
      BOT_MODE: gateway
      DISCORD_TOKEN: ${DISCORD_TOKEN}
      NATS_URL: nats://nats:4222
      DATABASE_URL: postgres://frolf_user:secure_password@postgres:5432/discord_frolf_bot?sslmode=disable
      ENVIRONMENT: production
    depends_on:
      - postgres
      - nats
    deploy:
      replicas: 1  # MUST be 1 - Discord allows only one connection

  worker:
    image: your-registry/discord-frolf-bot:latest
    environment:
      BOT_MODE: worker
      NATS_URL: nats://nats:4222
      DATABASE_URL: postgres://frolf_user:secure_password@postgres:5432/discord_frolf_bot?sslmode=disable
      ENVIRONMENT: production
    depends_on:
      - postgres
      - nats
    deploy:
      replicas: 3  # Scale based on load

volumes:
  postgres_data:
  nats_data:
```

Deploy:
```bash
docker-compose -f docker-compose.prod.yml up -d
```

### Option 2: Kubernetes (Recommended for production scale)

1. **Build and push image**:
   ```bash
   docker build -t your-registry/discord-frolf-bot:latest .
   docker push your-registry/discord-frolf-bot:latest
   ```

2. **Create Kubernetes manifests**:

```yaml
# gateway-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: discord-frolf-bot-gateway
  labels:
    app: discord-frolf-bot
    mode: gateway
spec:
  replicas: 1  # CRITICAL: Must be exactly 1
  selector:
    matchLabels:
      app: discord-frolf-bot
      mode: gateway
  template:
    metadata:
      labels:
        app: discord-frolf-bot
        mode: gateway
    spec:
      containers:
      - name: gateway
        image: your-registry/discord-frolf-bot:latest
        env:
        - name: BOT_MODE
          value: "gateway"
        - name: DISCORD_TOKEN
          valueFrom:
            secretKeyRef:
              name: discord-secrets
              key: token
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: database-secrets
              key: url
        - name: ENVIRONMENT
          value: "production"
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
---
# worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: discord-frolf-bot-worker
  labels:
    app: discord-frolf-bot
    mode: worker
spec:
  replicas: 3  # Scale based on load
  selector:
    matchLabels:
      app: discord-frolf-bot
      mode: worker
  template:
    metadata:
      labels:
        app: discord-frolf-bot
        mode: worker
    spec:
      containers:
      - name: worker
        image: your-registry/discord-frolf-bot:latest
        env:
        - name: BOT_MODE
          value: "worker"
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: database-secrets
              key: url
        - name: ENVIRONMENT
          value: "production"
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
```

3. **Deploy**:
   ```bash
   kubectl apply -f gateway-deployment.yaml
   kubectl apply -f worker-deployment.yaml
   ```

## Environment Variables

### Required
- `DISCORD_TOKEN`: Discord bot token
- `NATS_URL`: NATS server URL (default: nats://localhost:4222)

### Optional
- `BOT_MODE`: Deployment mode (`standalone`, `gateway`, `worker`)
- `DISCORD_GUILD_ID`: Single guild ID (development/testing only)
- `DATABASE_URL`: PostgreSQL connection string
- `ENVIRONMENT`: Environment name for queue isolation (default: development)
- `METRICS_ADDRESS`: Metrics server address (default: :8080)
- `LOKI_URL`: Loki logging endpoint
- `TEMPO_ENDPOINT`: Tempo tracing endpoint

## Configuration Management

### Guild Setup
Initialize bot configuration for a new guild:

```bash
# Using Go directly
go run main.go setup YOUR_GUILD_ID

# Using Docker (standalone mode)
docker run --rm \
  -e DISCORD_TOKEN=your_token \
  -e NATS_URL=nats://host.docker.internal:4222 \
  your-registry/discord-frolf-bot:latest \
  setup YOUR_GUILD_ID

# Using Kubernetes job
kubectl run setup-guild --rm -it --restart=Never \
  --image=your-registry/discord-frolf-bot:latest \
  --env="DISCORD_TOKEN=your_token" \
  --env="NATS_URL=nats://nats:4222" \
  -- setup YOUR_GUILD_ID
```

### Runtime Configuration
- Guild configurations are loaded dynamically from the database
- No restarts required for new guild configurations
- Backend workers automatically pick up new guild events
kubectl run discord-setup --rm -it --restart=Never \
  --image=your-registry/discord-frolf-bot:latest \
  --env="DATABASE_URL=$DATABASE_URL" \
  --env="DISCORD_GUILD_ID=$GUILD_ID" \
  -- setup $GUILD_ID
```

### Configuration Updates
- **File-based**: Update `config.yaml` and restart container
- **Database-backed**: Use Discord commands or update database directly

## Health Checks

The application provides health endpoints:

- `GET /health`: Basic health check
- `GET /ready`: Readiness check (Discord connection status)

Use these for:
- Docker health checks
- Kubernetes probes
- Load balancer health checks

## Monitoring

The bot includes comprehensive observability:

- **Metrics**: Prometheus metrics on `:8080/metrics`
- **Logging**: Structured JSON logs to stdout
- **Tracing**: OpenTelemetry traces to Tempo

Configure your monitoring stack to collect from these endpoints.

## Security Considerations

1. **Secrets Management**:
   - Use Kubernetes secrets or Docker secrets for sensitive data
   - Never commit tokens to version control
   - Use environment variables for configuration

2. **Network Security**:
   - Run containers as non-root user (automatically done)
   - Use distroless base image (minimal attack surface)
   - Configure firewalls appropriately

3. **Container Security**:
   - Regularly update base images
   - Scan images for vulnerabilities
   - Use signed images when possible

## Troubleshooting

### Common Issues

1. **Bot not responding**:
   - Verify exactly 1 gateway pod is running
   - Check `/ready` endpoint on gateway pod
   - Verify Discord token and permissions
   - Check gateway pod logs for Discord connection errors

2. **Events not processing**:
   - Check worker pod logs for event processing
   - Verify NATS connectivity between gateway and workers
   - Check queue group isolation (environment settings)
   - Monitor NATS for message accumulation

3. **Multi-guild isolation issues**:
   - Verify `guild_id` is included in all events
   - Check database for guild configuration conflicts
   - Review event handler guild scoping

4. **Performance issues**:
   - Scale worker pods based on event processing lag
   - Check database connection pooling
   - Monitor NATS message throughput
   - Review resource limits on pods

### Pod-specific Troubleshooting

**Gateway Pod Issues:**
```bash
# Check Discord connection status
kubectl logs -f deployment/discord-frolf-bot-gateway

# Verify health endpoints
kubectl port-forward deployment/discord-frolf-bot-gateway 8080:8080
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

**Worker Pod Issues:**
```bash
# Check event processing logs
kubectl logs -f deployment/discord-frolf-bot-worker

# Monitor specific worker pod
kubectl logs -f discord-frolf-bot-worker-<pod-id>
```

**NATS Issues:**
```bash
# Check NATS connectivity
kubectl exec -it deployment/discord-frolf-bot-gateway -- nc -zv nats 4222

# Monitor NATS streams (if NATS box is available)
kubectl exec -it nats-box -- nats stream ls
kubectl exec -it nats-box -- nats consumer ls
```

### Logs
Access container logs:
```bash
# Docker Compose
docker-compose logs discord-frolf-bot-gateway
docker-compose logs discord-frolf-bot-worker

# Kubernetes - Gateway logs
kubectl logs -f deployment/discord-frolf-bot-gateway

# Kubernetes - Worker logs
kubectl logs -f deployment/discord-frolf-bot-worker

# Kubernetes - Specific pod logs
kubectl logs -f <pod-name>
```

### Debug Mode
Enable verbose logging by setting environment variable:
```bash
LOG_LEVEL=debug
```

### Monitoring Multi-Pod Architecture

Monitor the separation between gateway and worker responsibilities:

**Gateway Pod Metrics:**
- Discord API response times
- Command registration/processing rates
- Discord connection status
- Event publishing rates

**Worker Pod Metrics:**
- Event processing lag
- Database operation times
- Queue processing rates
- Guild-specific event throughput

## CI/CD Pipeline

The included GitHub Actions workflow provides:

1. **Testing**: Automated unit and integration tests
2. **Security**: Vulnerability scanning with Trivy
3. **Building**: Multi-architecture Docker images
4. **Publishing**: Automated container registry publishing
5. **Deployment**: Optional infrastructure repository updates

Configure the following secrets in your GitHub repository:
- `CODECOV_TOKEN`: For code coverage reporting
- `GCP_SA_KEY`: For Google Cloud Artifact Registry (if using)
- `INFRA_REPO_TOKEN`: For infrastructure repository updates

## Performance Tuning

### Resource Requirements
- **CPU**: 0.1-0.5 cores (typical)
- **Memory**: 128-512MB (depending on server size)
- **Storage**: Minimal (logs and config only)

### Scaling
- **Gateway pods**: MUST be exactly 1 replica (Discord connection limitation)
- **Worker pods**: Scale horizontally based on event processing load
- **Database**: Use connection pooling and read replicas for high load
- **NATS**: Use clustering for high availability

### Resource Requirements

**Gateway Pod:**
- **CPU**: 0.1-0.3 cores (Discord API interactions)
- **Memory**: 128-256MB
- **Storage**: Minimal (logs only)

**Worker Pods:**
- **CPU**: 0.2-0.5 cores (event processing)
- **Memory**: 256-512MB (depending on concurrent processing)
- **Storage**: Minimal (logs only)

**Infrastructure:**
- **PostgreSQL**: 1-2 cores, 2-4GB RAM (depending on guild count)
- **NATS**: 0.5-1 core, 512MB-1GB RAM

## Backup and Recovery

### Event-Driven Architecture Backups
- **Database**: Regular PostgreSQL backups for guild configurations and data
- **NATS**: JetStream persistence backups for event durability
- **Configuration**: Environment-specific configuration backups

### Multi-Tenant Data Isolation
- Each guild's data is isolated by `guild_id`
- Selective restore capabilities per guild
- Event replay for specific guilds if needed

### Disaster Recovery
- **Gateway pod failure**: Kubernetes automatically restarts (Discord reconnection)
- **Worker pod failure**: Event processing continues with remaining workers
- **Database failure**: Point-in-time recovery with event replay capability
- **NATS failure**: JetStream durability ensures no event loss

## Support

For issues and questions:
1. Check this deployment guide
2. Review application logs
3. Check GitHub issues
4. Contact development team

## Version Compatibility

This deployment guide is compatible with:
- Docker 20.10+
- Docker Compose 2.0+
- Kubernetes 1.20+
- PostgreSQL 12+
- NATS 2.8+ with JetStream enabled
- Go 1.21+ (for local development)

## Multi-Pod Architecture Benefits

- **Resilience**: Gateway and worker pods can fail independently
- **Scalability**: Worker pods scale based on event processing load
- **Discord Compliance**: Single gateway ensures no Discord API conflicts
- **Event Durability**: NATS JetStream ensures no event loss
- **Multi-Tenant**: Unlimited guilds with proper isolation
- **Zero Downtime**: Rolling deployments possible for worker pods
