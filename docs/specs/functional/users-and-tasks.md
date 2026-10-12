# Spécification fonctionnelle — utilisateurs et tâches

## Statut

`active` — comportement actuellement implémenté et couvert par les routes et
les tests existants.

## Utilisateurs

- Un utilisateur possède un nom et un email non vides.
- Un utilisateur peut être créé, listé, récupéré par son identifiant, renommé (`PUT`, le nom seul : l'e-mail et l'identifiant ne changent jamais) ou supprimé avec toutes ses tâches (`DELETE`).
- Un utilisateur absent lors d'un renommage ou d'une suppression renvoie `404 Not Found`, et un renommage ne recrée jamais un utilisateur.
- Une création valide renvoie `201 Created`.
- Une requête invalide renvoie `400 Bad Request`.
- Un utilisateur absent renvoie `404 Not Found`.

## Tâches

- Une tâche appartient à un utilisateur identifié par `user_id`.
- Une tâche possède un titre non vide, une description et un état `completed`.
- Une tâche peut être créée, mise à jour, supprimée ou listée par utilisateur.
- Une création valide renvoie `201 Created`.
- Une mise à jour valide renvoie `200 OK`.
- Une suppression valide renvoie `200 OK`.
- Une tâche absente lors d'une mise à jour renvoie `404 Not Found`.
- Les identifiants de route invalides renvoient `400 Bad Request`.

## Statistiques et santé

- `/api/v1/stats` renvoie le nombre d'utilisateurs et de tâches.
- `/health` renvoie l'état `ok`, la version du service et un identifiant UUID
  généré pour la réponse.

## Limites connues

- L'email est seulement vérifié comme non vide; aucun format email n'est
  imposé par le domaine.
- Aucune authentification ou autorisation n'est actuellement définie.
- La mise à jour d'une tâche rejette un titre explicitement vide (`Q-01`).
