-- Schéma initial : identique à celui que créerait GORM AutoMigrate pour les modèles
-- de pgstore (user_models, task_models), idempotent pour une base créée ainsi. Les
-- tables users et tasks de la 1.0.0 ne sont pas migrées : pgsetup refuse de
-- démarrer tant qu'elles existent (docs/troubleshooting.md).
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
