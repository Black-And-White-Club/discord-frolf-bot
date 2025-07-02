# Discord Sharding for Multi-Pod (Advanced)

## When to Use Sharding:
- Bot is in 2000+ Discord servers
- Need to distribute Discord events across pods
- Want true horizontal scaling of Discord connections

## Implementation:

```go
// Shard-aware bot
func main() {
    shardID := getShardID()      // 0, 1, 2, etc.
    totalShards := getTotalShards() // 3, 4, etc.
    
    // Each pod handles different guilds
    discordSession, err := discordgo.New("Bot " + token)
    discordSession.ShardID = shardID
    discordSession.ShardCount = totalShards
    
    // This pod only receives events for its assigned guilds
}
```

## Kubernetes Deployment:

```yaml
apiVersion: apps/v1
kind: StatefulSet  # Use StatefulSet for stable shard IDs
metadata:
  name: discord-bot-shards
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: discord-bot
        env:
        - name: SHARD_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['shard-id']
        - name: TOTAL_SHARDS
          value: "3"
```

## Complexity:
- 🔴 More complex deployment
- 🔴 Requires StatefulSet
- 🔴 Shard management logic
- 🔴 Debugging is harder
- 🔴 Only needed for very large bots
