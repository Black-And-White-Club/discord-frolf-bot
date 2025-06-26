# Deployment Guide

This guide covers deploying the Discord Frolf Bot using modern container practices.

## Prerequisites

- Docker and Docker Compose installed
- Discord bot token and application configured
- (Optional) Google Cloud Platform account for container registry
- (Optional) Kubernetes cluster for production deployment

## Configuration Options

The bot supports two configuration modes:

### 1. File-based Configuration (Single Server)
- Use for single Discord server deployments
- Configuration stored in `config.yaml` file
- Simpler setup, good for development and small deployments

### 2. Database-backed Configuration (Multi-tenant)
- Use for multiple Discord servers
- Configuration stored in PostgreSQL database
- Supports runtime configuration updates
- Better for production and scaling

## Local Development

1. Copy the example configuration:
   ```bash
   cp config.example.yaml config.yaml
   ```

2. Edit `config.yaml` with your Discord bot credentials

3. Start the development environment:
   ```bash
   docker-compose up -d postgres nats
   docker-compose up discord-frolf-bot
   ```

## Production Deployment

### Option 1: Docker Compose (Recommended for small deployments)

1. Create production configuration:
   ```bash
   # For file-based config
   cp config.example.yaml config.yaml
   # Edit with production values
   
   # For database config, set environment variables:
   export DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=require"
   export DISCORD_GUILD_ID="your-guild-id"
   ```

2. Deploy:
   ```bash
   docker-compose -f docker-compose.yml up -d
   ```

### Option 2: Kubernetes (Recommended for production)

1. Build and push image:
   ```bash
   docker build -t your-registry/discord-frolf-bot:latest .
   docker push your-registry/discord-frolf-bot:latest
   ```

2. Create Kubernetes manifests (see k8s/ directory in infrastructure repo)

3. Deploy:
   ```bash
   kubectl apply -f k8s/
   ```

## Environment Variables

### Required
- `DISCORD_TOKEN`: Discord bot token
- `DISCORD_GUILD_ID`: Discord server ID (for database config only)

### Optional
- `DATABASE_URL`: PostgreSQL connection string (enables database config)
- `NATS_URL`: NATS server URL (default: nats://localhost:4222)
- `METRICS_ADDRESS`: Metrics server address (default: :8080)
- `LOKI_URL`: Loki logging endpoint
- `TEMPO_ENDPOINT`: Tempo tracing endpoint

## Configuration Management

### Initial Setup
Run the setup command to initialize bot configuration:

```bash
# For Docker
docker run --rm -v $(pwd)/config.yaml:/config.yaml \
  your-registry/discord-frolf-bot:latest setup YOUR_GUILD_ID

# For Kubernetes
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
   - Check `/ready` endpoint
   - Verify Discord token and permissions
   - Check logs for connection errors

2. **Configuration not loading**:
   - Verify file permissions
   - Check environment variables
   - Review database connection (if using DB config)

3. **Performance issues**:
   - Check metrics endpoint
   - Review resource limits
   - Monitor database performance

### Logs
Access container logs:
```bash
# Docker Compose
docker-compose logs discord-frolf-bot

# Kubernetes
kubectl logs -f deployment/discord-frolf-bot
```

### Debug Mode
Enable verbose logging by setting environment variable:
```bash
LOG_LEVEL=debug
```

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
- **Horizontal**: Not recommended (Discord bots are stateful)
- **Vertical**: Scale CPU/memory based on server activity
- **Database**: Use connection pooling for database-backed config

## Backup and Recovery

### File-based Configuration
- Backup `config.yaml` file
- Store in version control (without secrets)

### Database-backed Configuration
- Regular PostgreSQL backups
- Point-in-time recovery capability
- Configuration export/import tools

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
- NATS 2.8+
