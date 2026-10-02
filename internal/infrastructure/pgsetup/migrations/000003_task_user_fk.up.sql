-- Refuse de s'appliquer, sans rien supprimer, s'il existe des tâches dont
-- l'utilisateur n'existe pas.
DO $$
DECLARE
    orphan_tasks bigint;
BEGIN
    SELECT count(*) INTO orphan_tasks
    FROM task_models t
    WHERE NOT EXISTS (SELECT 1 FROM user_models u WHERE u.id = t.user_id);

    IF orphan_tasks > 0 THEN
        RAISE EXCEPTION 'cannot add foreign key: % task(s) reference a user that does not exist; fix or delete them manually, nothing was changed', orphan_tasks;
    END IF;
END $$;

ALTER TABLE task_models
    ADD CONSTRAINT fk_task_models_user FOREIGN KEY (user_id) REFERENCES user_models (id);

CREATE INDEX idx_task_models_user_id ON task_models (user_id);
