-- La mise en minuscules des emails existants n'est pas réversible.
CREATE UNIQUE INDEX idx_user_models_email ON user_models (email);
DROP INDEX idx_user_models_email_lower;
