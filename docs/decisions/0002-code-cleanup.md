# ADR 0002 : Cleanup legacy code & documentation governance

- **Statut** : accepté
- **Périmètre** : `inbackend`
- **Date** : 2026-09-20

## Contexte

Lors d'un audit systématique du projet (Phases 1-6 post-completion), deux problèmes ont été identifiés :

1. **Code legacy** : `internal/handlers/handlers.go` contenait des handlers monolithiques du code pré-Phase 1. Ce fichier a été complètement remplacé par les quatre couches (domain/application/controllers/infrastructure) et n'est plus utilisé nulle part (0 références).

2. **Documentation invisible** : `CI_CD_PIPELINE.md` existait à la racine mais n'était pas indexé dans `docs/README.md`, le rendre invisible pour les nouveaux contributeurs.

Ces problèmes violaient la gouvernance documentaire établie dans le workspace (une seule source de vérité, tout est indexé).

## Décision

1. **Supprimer `internal/handlers/` complet**
   - Fichier `handlers.go` n'a aucun usage (grep validé)
   - Dossier devient vide après suppression
   - Artifacts de code ancien

2. **Indexer `CI_CD_PIPELINE.md` dans `docs/README.md`**
   - Déplacer détails CI/CD vers document dédié
   - Conserver `docs/README.md` concis (index + liens)
   - Respecter principe "une seule source de vérité"

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

- ✅ `go build .` : succès
- ✅ `go test -v -short ./internal/...` : tous les tests passent
- ✅ `gofmt -s -l .` : format OK
- ✅ `go mod tidy` : modules OK
- ✅ Grep `internal/handlers` : 0 références trouvées
- ✅ Grep `handlers.Handler` : 0 références trouvées

## Notes & Références

- Audit systématique effectué : 51+ fichiers explorés ligne par ligne
- Checklist exhaustive : domain/app/controllers/infrastructure/logger/models/shared/tests/docs
- Anomalies trouvées : CI_CD_PIPELINE.md (invisible) + handlers.go (unused)
- Corrections appliquées & validées en même session

Voir [`handoff.md`](../handoff.md) Phase 6.1 pour détails audit.
