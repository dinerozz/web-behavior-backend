-- Rollback migration: Restore original indexes

-- Step 1: Remove new optimized indexes
DROP INDEX IF EXISTS idx_user_behaviors_main_query;
DROP INDEX IF EXISTS idx_user_behaviors_active_events;
DROP INDEX IF EXISTS idx_user_behaviors_deep_work;

-- Step 2: Recreate original indexes that were removed
CREATE INDEX IF NOT EXISTS idx_user_behaviors_url ON user_behaviors(url);
CREATE INDEX IF NOT EXISTS idx_user_behaviors_session_id ON user_behaviors(session_id);

-- Step 3: Update statistics
ANALYZE user_behaviors;