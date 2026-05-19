DROP INDEX IF EXISTS conv_user_idx;
ALTER TABLE conversations DROP COLUMN IF EXISTS user_id;

DROP INDEX IF EXISTS users_email_idx;
DROP TABLE IF EXISTS users;
