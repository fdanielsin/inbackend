-- Refuse de s'appliquer, sans rien modifier, si deux utilisateurs ne diffèrent
-- que par la casse ou les espaces de leur email. Le message ne contient que le
-- nombre de groupes : un email n'a pas à finir dans les logs.
DO $$
DECLARE
    duplicate_groups bigint;
BEGIN
    SELECT count(*) INTO duplicate_groups FROM (
        SELECT 1 FROM user_models
        WHERE email IS NOT NULL
        GROUP BY lower(btrim(email, E' \t\n\r\f\v'))
        HAVING count(*) > 1
    ) AS duplicates;

    IF duplicate_groups > 0 THEN
        RAISE EXCEPTION 'cannot normalize emails: % group(s) of users share the same email once trimmed and lowercased; resolve them manually, nothing was changed', duplicate_groups;
    END IF;
END $$;

UPDATE user_models
SET email = lower(btrim(email, E' \t\n\r\f\v'))
WHERE email <> lower(btrim(email, E' \t\n\r\f\v'));

CREATE UNIQUE INDEX idx_user_models_email_lower ON user_models (lower(email));
DROP INDEX idx_user_models_email;
