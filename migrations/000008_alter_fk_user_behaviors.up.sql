ALTER TABLE user_behaviors
    ADD CONSTRAINT fk_user_behaviors_user_id
        FOREIGN KEY (user_id) REFERENCES extension_users(id)
            ON DELETE SET NULL
            ON UPDATE CASCADE;
