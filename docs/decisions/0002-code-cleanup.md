# ADR 0002 : Cleanup legacy code & documentation governance

- **Statut** : accepted
- **Périmètre** : `inbackend`
- **Date** : 2026-09-20

## Contexte

Lors d'un audit systématique du projet (Phases 1-6 post-completion), deux problèmes ont été identifiés :

1. **Code legacy** : `internal/handlers/handlers.go` contenait des handlers monolithiques du code pré-Phase 1. Ce fichier a été complètement remplacé par les quatre couches (domain/application/controllers/infrastructure) et n'est plus utilisé nulle part (0 références).

2. **Documentation invisible** : un ancien document CI n'était pas indexé et a
   ensuite été supprimé pour éviter une seconde source de vérité.

Ces problèmes violaient la gouvernance documentaire établie dans le workspace (une seule source de vérité, tout est indexé).

## Décision

1. **Supprimer `internal/handlers/` complet**
   - Fichier `handlers.go` n'a aucun usage (grep validé)
   - Dossier devient vide après suppression
   - Artifacts de code ancien

2. **Conserver la CI dans sa source canonique**
   - Le workflow réel reste dans `.github/workflows/ci.yml`
   - Les décisions de workflow restent dans l'ADR dédiée
   - Ne pas recréer `CI_CD_PIPELINE.md`

3. **Appliquer gouvernance documentaire stricte**
   - README.md = index + liens (pas de contenu dupliqué)
   - Détails techniques = dans documents spécialisés
   - Audit régulier pour éviter documents "invisibles"

## Conséquences positives

- ✅ Codebase plus propre (pas de dossiers inutilisés)
- ✅ Documentation cohérente (tout indexé & navigable)
- ✅ Nouvelles phases : modèle clair (ADR pour décisions, suppression de legacy)
- ✅ Gouvernance renforcée (une seule source de vérité)

## Conséquences acceptées

- ⚠️ Nécessite audit régulier des fichiers pour éviter regréssion
- ⚠️ README.md n'est plus la source de détails (peut surprendre)
- ⚠️ Dossiers vides doivent être détectés & nettoyés (pas automatique)

## Alternatives rejetées

- Conserver `internal/handlers/` "au cas où" → Crée confusion & dette
- Garder détails CI/CD dans README → Viole DRY & gouvernance
- Créer système de documentation "optionnelle" → Crée silos

## Validation

- Les validations historiques doivent être distinguées des validations
  exécutées lors d'une reprise ultérieure.
- L'absence actuelle de `internal/handlers/` est vérifiable dans l'arbre du
  dépôt ; elle ne prouve pas à elle seule la conformité complète de la phase.

## Notes & Références

- Le détail de l'audit historique est conservé dans Git et dans le handoff,
  mais ne constitue pas une preuve de l'état courant sans nouvelle vérification.

Voir [`handoff.md`](../handoff.md) Phase 6.1 pour détails audit.
