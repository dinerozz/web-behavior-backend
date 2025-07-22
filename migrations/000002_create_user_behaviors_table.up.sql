CREATE TABLE IF NOT EXISTS user_behaviors (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
                            event_type VARCHAR(50) NOT NULL,
    url TEXT NOT NULL,
    user_id VARCHAR(255),
    user_name VARCHAR(255),

    -- Координаты для событий click
    x INTEGER,
    y INTEGER,

    -- Клавиша для событий keyup
    key VARCHAR(50),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_user_behaviors_session_id ON user_behaviors(session_id);
CREATE INDEX IF NOT EXISTS idx_user_behaviors_user_id ON user_behaviors(user_id);
CREATE INDEX IF NOT EXISTS idx_user_behaviors_timestamp ON user_behaviors(timestamp);
CREATE INDEX IF NOT EXISTS idx_user_behaviors_event_type ON user_behaviors(event_type);
CREATE INDEX IF NOT EXISTS idx_user_behaviors_url ON user_behaviors(url);

-- Составной индекс для частых запросов
CREATE INDEX IF NOT EXISTS idx_user_behaviors_user_session ON user_behaviors(user_id, session_id);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_user_behaviors_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_behaviors_updated_at_trigger
    BEFORE UPDATE ON user_behaviors
    FOR EACH ROW
    EXECUTE FUNCTION update_user_behaviors_updated_at();