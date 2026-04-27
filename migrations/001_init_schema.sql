-- 001_init_schema.sql
-- Initialize test database schema

-- TikTok Accounts table
CREATE TABLE IF NOT EXISTS tiktok_accounts (
    id VARCHAR(64) PRIMARY KEY,
    username VARCHAR(255),
    email VARCHAR(255) UNIQUE NOT NULL,
    email_password VARCHAR(255),
    phone_id VARCHAR(64),
    phone_number VARCHAR(32),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    account_level INTEGER DEFAULT 0,
    account_group VARCHAR(64),
    instance_id VARCHAR(64),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Browser Instances table
CREATE TABLE IF NOT EXISTS browser_instances (
    id VARCHAR(64) PRIMARY KEY,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    fingerprint_json TEXT,
    proxy_id VARCHAR(64),
    account_id VARCHAR(64),
    cdp_endpoint TEXT,
    pid INTEGER,
    port INTEGER,
    user_data_dir TEXT,
    group_name VARCHAR(64),
    started_at TIMESTAMP,
    last_active_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Phone Numbers table
CREATE TABLE IF NOT EXISTS phone_numbers (
    id VARCHAR(64) PRIMARY KEY,
    number VARCHAR(32) NOT NULL,
    country VARCHAR(8) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'disposable',
    provider VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'available',
    external_id VARCHAR(128),
    rent_expires_at TIMESTAMP,
    instance_id VARCHAR(64),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Proxies table
CREATE TABLE IF NOT EXISTS proxies (
    id VARCHAR(64) PRIMARY KEY,
    ip VARCHAR(64) NOT NULL,
    port INTEGER NOT NULL,
    country VARCHAR(8) NOT NULL,
    city VARCHAR(128),
    type VARCHAR(32) NOT NULL DEFAULT 'residential',
    username VARCHAR(128),
    password VARCHAR(128),
    status VARCHAR(32) NOT NULL DEFAULT 'available',
    bind_id VARCHAR(64),
    bound_at TIMESTAMP,
    last_check_at TIMESTAMP,
    success_rate DOUBLE PRECISION DEFAULT 1.0,
    latency INTEGER DEFAULT 0,
    provider VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Content table
CREATE TABLE IF NOT EXISTS content (
    id VARCHAR(64) PRIMARY KEY,
    title VARCHAR(512) NOT NULL,
    description TEXT,
    tags TEXT[],
    video_path VARCHAR(512),
    created_by VARCHAR(64),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Publish Tasks table
CREATE TABLE IF NOT EXISTS publish_tasks (
    id VARCHAR(64) PRIMARY KEY,
    content_id VARCHAR(64) NOT NULL,
    account_ids TEXT[],
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    scheduled_at TIMESTAMP NOT NULL,
    results JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_accounts_email ON tiktok_accounts(email);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON tiktok_accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_group ON tiktok_accounts(account_group);

CREATE INDEX IF NOT EXISTS idx_instances_status ON browser_instances(status);
CREATE INDEX IF NOT EXISTS idx_instances_account ON browser_instances(account_id);
CREATE INDEX IF NOT EXISTS idx_instances_proxy ON browser_instances(proxy_id);

CREATE INDEX IF NOT EXISTS idx_phones_status ON phone_numbers(status);
CREATE INDEX IF NOT EXISTS idx_phones_country ON phone_numbers(country);
CREATE INDEX IF NOT EXISTS idx_phones_provider ON phone_numbers(provider);

CREATE INDEX IF NOT EXISTS idx_proxies_status ON proxies(status);
CREATE INDEX IF NOT EXISTS idx_proxies_country ON proxies(country);
CREATE INDEX IF NOT EXISTS idx_proxies_bind ON proxies(bind_id);

CREATE INDEX IF NOT EXISTS idx_content_created_by ON content(created_by);
CREATE INDEX IF NOT EXISTS idx_publish_status ON publish_tasks(status);
