# Discord Bot Multi-Pod Architecture

## Recommended Architecture: Gateway + Workers

### Pod Types:

1. **Discord Gateway Pod** (1 replica)
   - Handles Discord WebSocket connection
   - Processes Discord events
   - Publishes events to NATS
   - Registers/manages commands
   - NO business logic

2. **Worker Pods** (N replicas)
   - Subscribe to NATS events
   - Process business logic
   - Handle backend operations
   - Scale horizontally
   - NO Discord connection

### Benefits:
- ✅ Single Discord connection (no conflicts)
- ✅ Horizontal scaling for business logic
- ✅ Fault isolation
- ✅ Simpler deployment

### Implementation:

```yaml
# Gateway deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: discord-gateway
spec:
  replicas: 1  # ALWAYS 1
  template:
    spec:
      containers:
      - name: gateway
        image: discord-frolf-bot:latest
        env:
        - name: BOT_MODE
          value: "gateway"
        - name: DISCORD_TOKEN
          valueFrom:
            secretKeyRef:
              name: discord-secrets
              key: token

---
# Worker deployment  
apiVersion: apps/v1
kind: Deployment
metadata:
  name: discord-workers
spec:
  replicas: 3  # Scale as needed
  template:
    spec:
      containers:
      - name: worker
        image: discord-frolf-bot:latest
        env:
        - name: BOT_MODE
          value: "worker"
        # No Discord token needed
```

### Code Changes Needed:

```go
// main.go
func main() {
    mode := os.Getenv("BOT_MODE")
    
    switch mode {
    case "gateway":
        runGatewayMode(ctx)
    case "worker":
        runWorkerMode(ctx)
    default:
        runStandaloneMode(ctx) // Current single-pod mode
    }
}

func runGatewayMode(ctx context.Context) {
    // Create Discord session
    // Register commands
    // Handle Discord events
    // Publish to NATS
    // NO business logic
}

func runWorkerMode(ctx context.Context) {
    // Subscribe to NATS events
    // Process business logic
    // Handle backend operations
    // NO Discord connection
}
```
