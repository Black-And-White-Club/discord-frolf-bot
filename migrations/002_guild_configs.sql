-- Simple guild configuration table
-- This stores the Discord setup configuration for each guild
-- Future: Can be evolved to support multiple guilds or different tiers

CREATE TABLE IF NOT EXISTS guild_configs (
    guild_id VARCHAR(20) PRIMARY KEY,
    guild_name VARCHAR(100) NOT NULL,
    event_channel_id VARCHAR(20) NOT NULL,
    event_channel_name VARCHAR(100) NOT NULL,
    leaderboard_channel_id VARCHAR(20) NOT NULL,
    leaderboard_channel_name VARCHAR(100) NOT NULL,
    signup_channel_id VARCHAR(20) NOT NULL,
    signup_channel_name VARCHAR(100) NOT NULL,
    role_mappings JSONB NOT NULL DEFAULT '{}',
    registered_role_id VARCHAR(20),
    admin_role_id VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_guild_configs_guild_id ON guild_configs(guild_id);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_guild_configs_updated_at 
    BEFORE UPDATE ON guild_configs 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Future evolution: This table can easily be extended to support:
-- - tier VARCHAR(10) DEFAULT 'free' -- for premium features
-- - is_active BOOLEAN DEFAULT true   -- for soft deletes
-- - additional_settings JSONB        -- for future features
-- - billing_info JSONB               -- for premium billing
