-- Multi-tenant database schema
-- This supports both single-server and multi-server deployments

-- Server configurations table
CREATE TABLE IF NOT EXISTS discord_servers (
    guild_id VARCHAR(20) PRIMARY KEY,
    guild_name VARCHAR(100),
    
    -- Channel configurations
    signup_channel_id VARCHAR(20),
    event_channel_id VARCHAR(20),
    leaderboard_channel_id VARCHAR(20),
    signup_message_id VARCHAR(20),
    
    -- Role configurations
    registered_role_id VARCHAR(20),
    admin_role_id VARCHAR(20),
    
    -- Settings
    signup_emoji VARCHAR(10) DEFAULT '🐍',
    settings JSONB DEFAULT '{}',
    
    -- Premium/subscription info
    subscription_tier VARCHAR(20) DEFAULT 'free',
    subscription_expires_at TIMESTAMP,
    
    -- Metadata
    bot_joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    
    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Role mappings (flexible role system)
CREATE TABLE IF NOT EXISTS server_role_mappings (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(20) REFERENCES discord_servers(guild_id) ON DELETE CASCADE,
    role_name VARCHAR(50), -- e.g., "Rattler", "Editor", "Admin"
    role_id VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(guild_id, role_name)
);

-- Server activity/usage tracking
CREATE TABLE IF NOT EXISTS server_usage (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(20) REFERENCES discord_servers(guild_id) ON DELETE CASCADE,
    
    -- Usage metrics
    rounds_created_today INTEGER DEFAULT 0,
    commands_used_today INTEGER DEFAULT 0,
    active_users_today INTEGER DEFAULT 0,
    
    -- Tracking date
    usage_date DATE DEFAULT CURRENT_DATE,
    
    UNIQUE(guild_id, usage_date)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_discord_servers_active ON discord_servers(is_active);
CREATE INDEX IF NOT EXISTS idx_discord_servers_subscription ON discord_servers(subscription_tier, subscription_expires_at);
CREATE INDEX IF NOT EXISTS idx_server_usage_guild_date ON server_usage(guild_id, usage_date);

-- Views for monitoring
CREATE OR REPLACE VIEW active_servers AS
SELECT 
    guild_id,
    guild_name,
    subscription_tier,
    bot_joined_at,
    last_activity,
    (last_activity > CURRENT_TIMESTAMP - INTERVAL '7 days') AS active_last_week
FROM discord_servers 
WHERE is_active = true;

-- Function to update last_activity
CREATE OR REPLACE FUNCTION update_server_activity(server_guild_id VARCHAR(20))
RETURNS void AS $$
BEGIN
    UPDATE discord_servers 
    SET last_activity = CURRENT_TIMESTAMP 
    WHERE guild_id = server_guild_id;
    
    -- Also update daily usage
    INSERT INTO server_usage (guild_id, commands_used_today)
    VALUES (server_guild_id, 1)
    ON CONFLICT (guild_id, usage_date)
    DO UPDATE SET commands_used_today = server_usage.commands_used_today + 1;
END;
$$ LANGUAGE plpgsql;
