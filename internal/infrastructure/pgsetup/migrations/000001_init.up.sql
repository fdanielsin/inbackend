-- Schéma initial : identique à celui que créait GORM AutoMigrate, idempotent pour
-- les bases déjà créées par lui.
CREATE TABLE IF NOT EXISTS user_models (
    id    bigserial PRIMARY KEY,
    name  text,
    email text
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_models_email ON user_models (email);

CREATE TABLE IF NOT EXISTS task_models (
    id          bigserial PRIMARY KEY,
    title       text,
    description text,
    user_id     bigint,
    completed   boolean
);
