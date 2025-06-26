-- Discord Frolf Bot Database Schema
-- Multi-tenant/Premium bot support

-- Guild configurations (per-Discord server)
CREATE TABLE guild_configs (
    guild_id VARCHAR(20) PRIMARY KEY,  -- Discord guild/server ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Basic settings
    display_name VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    
    -- Subscription/Premium settings
    subscription_tier VARCHAR(20) DEFAULT 'basic',  -- 'basic', 'premium', 'enterprise'
    subscription_expires_at TIMESTAMP NULL,
    is_trial BOOLEAN DEFAULT false,
    trial_expires_at TIMESTAMP NULL,
    
    -- Auto-discovered Discord resources
    signup_channel_id VARCHAR(20),
    signup_message_id VARCHAR(20),
    event_channel_id VARCHAR(20),
    leaderboard_channel_id VARCHAR(20),
    registered_role_id VARCHAR(20),
    admin_role_id VARCHAR(20),
    
    -- Configuration
    signup_emoji VARCHAR(10) DEFAULT '🐍',
    auto_setup_completed BOOLEAN DEFAULT false,
    setup_completed_at TIMESTAMP NULL,
    
    -- Limits and features
    max_concurrent_rounds INTEGER DEFAULT 5,
    max_participants_per_round INTEGER DEFAULT 50,
    commands_per_minute INTEGER DEFAULT 30,
    rounds_per_day INTEGER DEFAULT 20,
    
    -- Premium features
    advanced_scoring_enabled BOOLEAN DEFAULT false,
    tournament_mode_enabled BOOLEAN DEFAULT false,
    custom_leaderboards_enabled BOOLEAN DEFAULT false,
    
    INDEX idx_subscription_tier (subscription_tier),
    INDEX idx_is_active (is_active),
    INDEX idx_subscription_expires (subscription_expires_at)
);

-- Role mappings for each guild
CREATE TABLE guild_role_mappings (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    guild_id VARCHAR(20) NOT NULL,
    role_name VARCHAR(50) NOT NULL,  -- e.g., "Rattler", "Admin", "Editor"
    role_id VARCHAR(20) NOT NULL,    -- Discord role ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (guild_id) REFERENCES guild_configs(guild_id) ON DELETE CASCADE,
    UNIQUE KEY unique_guild_role (guild_id, role_name)
);

-- Usage tracking and analytics
CREATE TABLE guild_usage_stats (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    guild_id VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    
    -- Command usage
    commands_executed INTEGER DEFAULT 0,
    rounds_created INTEGER DEFAULT 0,
    participants_joined INTEGER DEFAULT 0,
    
    -- Performance metrics
    avg_response_time_ms INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (guild_id) REFERENCES guild_configs(guild_id) ON DELETE CASCADE,
    UNIQUE KEY unique_guild_date (guild_id, date)
);

-- Bot instance tracking (for multi-instance deployments)
CREATE TABLE bot_instances (
    instance_id VARCHAR(100) PRIMARY KEY,
    hostname VARCHAR(255),
    version VARCHAR(20),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    
    -- Deployment info
    kubernetes_pod VARCHAR(255),
    kubernetes_namespace VARCHAR(100),
    deployment_env VARCHAR(20),  -- 'development', 'staging', 'production'
    
    INDEX idx_is_active (is_active),
    INDEX idx_last_heartbeat (last_heartbeat)
);

-- Subscription billing events (if implementing your own billing)
CREATE TABLE subscription_events (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    guild_id VARCHAR(20) NOT NULL,
    event_type VARCHAR(50) NOT NULL,  -- 'trial_started', 'subscribed', 'upgraded', 'cancelled', 'expired'
    old_tier VARCHAR(20),
    new_tier VARCHAR(20),
    amount_cents INTEGER,  -- If tracking revenue
    currency VARCHAR(3) DEFAULT 'USD',
    external_subscription_id VARCHAR(100),  -- Stripe/PayPal ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (guild_id) REFERENCES guild_configs(guild_id) ON DELETE CASCADE,
    INDEX idx_event_type (event_type),
    INDEX idx_created_at (created_at)
);

-- Feature flags (for gradual rollouts or A/B testing)
CREATE TABLE feature_flags (
    flag_name VARCHAR(100) PRIMARY KEY,
    description TEXT,
    is_enabled BOOLEAN DEFAULT false,
    rollout_percentage INTEGER DEFAULT 0,  -- 0-100, for gradual rollouts
    target_tiers JSON,  -- JSON array of subscription tiers
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Initial feature flags
INSERT INTO feature_flags (flag_name, description, is_enabled, rollout_percentage, target_tiers) VALUES
('tournament_mode', 'Advanced tournament bracket system', false, 0, '["premium", "enterprise"]'),
('advanced_scoring', 'Detailed scoring with handicaps', false, 50, '["premium", "enterprise"]'),
('custom_leaderboards', 'User-defined leaderboard categories', false, 0, '["premium", "enterprise"]'),
('beta_features', 'Early access to new features', true, 10, '["premium", "enterprise"]');

-- Default guild configuration template
INSERT INTO guild_configs 
(guild_id, display_name, subscription_tier, max_concurrent_rounds, max_participants_per_round, commands_per_minute, rounds_per_day) 
VALUES 
('DEFAULT_TEMPLATE', 'Default Configuration', 'basic', 5, 50, 30, 20);
