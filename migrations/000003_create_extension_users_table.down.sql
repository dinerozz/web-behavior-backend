-- down migration: create_extension_users_table

-- Удаление триггера
DROP TRIGGER IF EXISTS update_extension_users_updated_at_trigger ON extension_users;

-- Удаление функции триггера
DROP FUNCTION IF EXISTS update_extension_users_updated_at();

-- Удаление индексов
DROP INDEX IF EXISTS idx_extension_users_key_active;
DROP INDEX IF EXISTS idx_extension_users_last_used;
DROP INDEX IF EXISTS idx_extension_users_is_active;
DROP INDEX IF EXISTS idx_extension_users_username;
DROP INDEX IF EXISTS idx_extension_users_api_key;

-- Удаление таблицы
DROP TABLE IF EXISTS extension_users;