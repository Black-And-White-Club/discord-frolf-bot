# Simple Discord Bot Activation Guide

This guide shows how to easily activate your Discord Frolf Bot on new servers using the built-in `/frolf-setup` slash command.

## The Easy Way: Use `/frolf-setup` Slash Command

Your bot already has an automated setup system! No need for manual kubectl commands or pre-configuration.

### Step 1: Deploy Bot to Kubernetes

```yaml
# discord-bot-deployment.yaml
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
        image: discord-frolf-bot:latest
        ports:
        - containerPort: 8080
        env:
        - name: DISCORD_TOKEN
          valueFrom:
            secretKeyRef:
              name: discord-bot-secret
              key: token
        # Optional: Add database for persistence across restarts
        # - name: DATABASE_URL
        #   value: "postgres://user:pass@host:5432/frolf_bot"
        # Optional: Specify guild ID for database-backed config
        # - name: DISCORD_GUILD_ID  
        #   value: "your_guild_id_here"
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi" 
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
```

Deploy it:
```bash
# Create the bot token secret
kubectl create secret generic discord-bot-secret \
  --from-literal=token='YOUR_BOT_TOKEN_HERE'

# Deploy the bot
kubectl apply -f discord-bot-deployment.yaml
```

**🎉 That's it! No DISCORD_GUILD_ID needed!** The bot will register commands globally and work in any server.

### Step 2: Invite Bot to Discord Server

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Select your bot application
3. Go to "OAuth2" → "URL Generator"
4. Select scopes: **`bot`** and **`applications.commands`**
5. Select permissions: **`Administrator`** (for easy setup)
6. Copy the generated URL and open it
7. Select your Discord server and authorize

### Step 3: Run Setup Command in Discord

Once the bot is online in your server:

1. **In any Discord channel, type:**
   ```
   /frolf-setup
   ```

2. **The bot will automatically:**
   - ✅ Create `#frolf-events` channel
   - ✅ Create `#frolf-leaderboard` channel  
   - ✅ Create `#frolf-signup` channel
   - ✅ Create `@Frolf Player` role
   - ✅ Create `@Frolf Admin` role
   - ✅ Set up channel permissions
   - ✅ Respond with success message

3. **Done!** Your server is ready for frolf events.

## That's It! 🎉

No kubectl commands, no manual config editing, no database required. Server admins can set up the bot themselves using the slash command.

## Adding Bot to Multiple Servers

To add the bot to multiple Discord servers:

1. **Use the same invite URL** for each server
2. **Each server admin runs** `/frolf-setup` in their server
3. **Each server gets its own setup** automatically

The bot can handle multiple servers simultaneously.

## Troubleshooting

### Bot doesn't respond to `/frolf-setup`
- Check bot is online: `kubectl get pods -l app=discord-frolf-bot`
- Check logs: `kubectl logs -l app=discord-frolf-bot`
- Ensure bot has `applications.commands` scope when invited

### Setup command fails
- Ensure the user running `/frolf-setup` has **Administrator** permissions
- Check bot has **Administrator** permissions in the server
- Check logs for detailed error messages

### Bot appears offline
- Verify `DISCORD_TOKEN` secret is correct
- Check pod status: `kubectl describe pod -l app=discord-frolf-bot`
- Ensure bot is invited to the server

## Optional: Add Database for Persistence

If you want to persist configurations across bot restarts:

1. **Deploy PostgreSQL:**
   ```bash
   # Use your preferred method (Helm chart, operator, etc.)
   ```

2. **Add DATABASE_URL to deployment:**
   ```yaml
   env:
   - name: DATABASE_URL
     value: "postgres://user:pass@postgres-service:5432/frolf_bot"
   ```

3. **Run database migrations:**
   ```bash
   kubectl exec -it discord-bot-pod -- sh
   # Inside pod: apply migrations/002_guild_configs.sql to your database
   ```

Without database, the bot still works perfectly - it just doesn't persist setup configs across restarts.

## Benefits of This Approach

✅ **Self-Service**: Server admins can set up the bot themselves  
✅ **No kubectl access needed**: Everything done through Discord  
✅ **Works immediately**: No additional infrastructure required  
✅ **Multi-server ready**: Each server gets independent setup  
✅ **User-friendly**: Provides helpful setup confirmation messages

## Next Steps

After setup, users can:
- React to signup messages in `#frolf-signup` to get the player role
- Use `/createround` to create new events
- Use `/claimtag` to claim leaderboard positions
- Admins can post in `#frolf-events` for announcements
