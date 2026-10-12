-- Supprimer un utilisateur supprime ses tâches (USER-R13). Une seule instruction,
-- donc atomique ; le nom de la contrainte est conservé, le code s'en sert pour
-- reconnaître la violation (user introuvable à la création d'une tâche).
ALTER TABLE task_models
    DROP CONSTRAINT fk_task_models_user,
    ADD CONSTRAINT fk_task_models_user FOREIGN KEY (user_id) REFERENCES user_models (id) ON DELETE CASCADE;
