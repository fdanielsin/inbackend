-- Retour au comportement de 000003 : la suppression d'un utilisateur qui a des
-- tâches est refusée. Aucune ligne n'est touchée.
ALTER TABLE task_models
    DROP CONSTRAINT fk_task_models_user,
    ADD CONSTRAINT fk_task_models_user FOREIGN KEY (user_id) REFERENCES user_models (id);
