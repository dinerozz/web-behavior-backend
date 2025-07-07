-- up migration: create_extension_users_table
CREATE TABLE IF NOT EXISTS extension_users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) NOT NULL UNIQUE,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP NULL
    );

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_extension_users_api_key ON extension_users(api_key);
CREATE INDEX IF NOT EXISTS idx_extension_users_username ON extension_users(username);
CREATE INDEX IF NOT EXISTS idx_extension_users_is_active ON extension_users(is_active);
CREATE INDEX IF NOT EXISTS idx_extension_users_last_used ON extension_users(last_used_at);

-- Составной индекс для аутентификации
CREATE INDEX IF NOT EXISTS idx_extension_users_key_active ON extension_users(api_key, is_active);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_extension_users_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_extension_users_updated_at_trigger
    BEFORE UPDATE ON extension_users
    FOR EACH ROW
    EXECUTE FUNCTION update_extension_users_updated_at();