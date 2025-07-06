-- down migration: create_user_behaviors_table

-- Удаление триггера
DROP TRIGGER IF EXISTS update_user_behaviors_updated_at_trigger ON user_behaviors;

-- Удаление функции триггера
DROP FUNCTION IF EXISTS update_user_behaviors_updated_at();

-- Удаление индексов
DROP INDEX IF EXISTS idx_user_behaviors_user_session;
DROP INDEX IF EXISTS idx_user_behaviors_url;
DROP INDEX IF EXISTS idx_user_behaviors_event_type;
DROP INDEX IF EXISTS idx_user_behaviors_timestamp;
DROP INDEX IF EXISTS idx_user_behaviors_user_id;
DROP INDEX IF EXISTS idx_user_behaviors_session_id;

-- Удаление таблицы
DROP TABLE IF EXISTS user_behaviors;

