-- 003_cleanup.sql
-- Cleanup script for test isolation - run before each test

-- Clean up test data (keep base schema)
TRUNCATE TABLE tiktok_accounts CASCADE;
TRUNCATE TABLE browser_instances CASCADE;
TRUNCATE TABLE phone_numbers CASCADE;
TRUNCATE TABLE proxies CASCADE;
TRUNCATE TABLE content CASCADE;
TRUNCATE TABLE publish_tasks CASCADE;

-- Reset sequences
ALTER SEQUENCE tiktok_accounts_id_seq RESTART WITH 1;
ALTER SEQUENCE browser_instances_id_seq RESTART WITH 1;
ALTER SEQUENCE phone_numbers_id_seq RESTART WITH 1;
ALTER SEQUENCE proxies_id_seq RESTART WITH 1;
ALTER SEQUENCE content_id_seq RESTART WITH 1;
ALTER SEQUENCE publish_tasks_id_seq RESTART WITH 1;
