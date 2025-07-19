-- Migration: Optimize user_behaviors table indexes for better performance
-- Date: 2025-07-19
-- Description: Add composite indexes and remove unused ones for engaged-time queries optimization

-- Step 1: Create main composite index for all engaged-time queries
-- This index covers: user_id + timestamp + event_type filters
-- INCLUDE adds session_id and url without affecting sort order
CREATE INDEX IF NOT EXISTS idx_user_behaviors_main_query
    ON user_behaviors (user_id, timestamp, event_type)
    INCLUDE (session_id, url);

-- Step 2: Create specialized index for active events filtering
-- This is a partial index - only for active event types
-- Dramatically speeds up minute_activity CTE
CREATE INDEX IF NOT EXISTS idx_user_behaviors_active_events
    ON user_behaviors (user_id, timestamp)
    WHERE event_type IN ('pageshow', 'click', 'focus', 'keyup', 'keydown', 'scrollend', 'pagehide', 'visibility_visible');

-- Step 3: Create index for deep work analysis (optional, for future use)
-- Partial index for deep work events with non-empty URLs
CREATE INDEX IF NOT EXISTS idx_user_behaviors_deep_work
    ON user_behaviors (user_id, timestamp)
    WHERE event_type IN ('click', 'keyup', 'scrollend') AND url IS NOT NULL AND url != '';

-- Step 4: Remove unused indexes to save space and maintenance overhead
-- These indexes have 0 usage based on pg_stat_user_indexes
DROP INDEX IF EXISTS idx_user_behaviors_url;
DROP INDEX IF EXISTS idx_user_behaviors_session_id;

-- Step 5: Update table statistics for query optimizer
ANALYZE user_behaviors;