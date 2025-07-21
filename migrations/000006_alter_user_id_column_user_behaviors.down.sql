ALTER TABLE user_behaviors ADD COLUMN user_id_temp VARCHAR(255);

ALTER TABLE user_behaviors DROP COLUMN user_id;

ALTER TABLE user_behaviors RENAME COLUMN user_id_temp TO user_id;