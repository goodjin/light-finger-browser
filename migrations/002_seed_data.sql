-- 002_seed_data.sql
-- Seed test data for integration tests

-- Insert test accounts
INSERT INTO tiktok_accounts (id, username, email, status, account_level, account_group, created_at, updated_at) VALUES
  ('test-account-001', 'test_user_001', 'test001@example.com', 'active', 1, 'test-group', NOW(), NOW()),
  ('test-account-002', 'test_user_002', 'test002@example.com', 'active', 1, 'test-group', NOW(), NOW()),
  ('test-account-003', 'test_user_003', 'test003@example.com', 'pending', 0, 'test-group', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert test instances
INSERT INTO browser_instances (id, status, fingerprint_json, account_id, port, group_name, started_at, last_active_at, created_at) VALUES
  ('test-instance-001', 'running', '{"browser":"chrome","version":"120"}', 'test-account-001', 34921, 'test-group', NOW(), NOW(), NOW()),
  ('test-instance-002', 'running', '{"browser":"chrome","version":"120"}', 'test-account-002', 34922, 'test-group', NOW(), NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert test phone numbers
INSERT INTO phone_numbers (id, number, country, type, provider, status, created_at, last_used_at) VALUES
  ('test-phone-001', '+1234567890', 'US', 'disposable', 'smsactivate', 'available', NOW(), NOW()),
  ('test-phone-002', '+1234567891', 'US', 'disposable', 'smsactivate', 'used', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert test proxies
INSERT INTO proxies (id, ip, port, country, city, type, username, password, status, success_rate, latency, provider, created_at, last_check_at) VALUES
  ('test-proxy-001', '192.168.1.1', 8080, 'US', 'New York', 'residential', 'user1', 'pass1', 'available', 0.95, 100, 'brightdata', NOW(), NOW()),
  ('test-proxy-002', '192.168.1.2', 8080, 'US', 'Los Angeles', 'residential', 'user2', 'pass2', 'available', 0.90, 150, 'oxylabs', NOW(), NOW()),
  ('test-proxy-003', '192.168.1.3', 8080, 'UK', 'London', 'datacenter', 'user3', 'pass3', 'dead', 0.20, 200, 'brightdata', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
