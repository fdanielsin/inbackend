# Spécifications du backend

Ce dossier contient les exigences et contrats propres à `inbackend`.

## Organisation prévue

```text
specs/
├── README.md
├── functional/
├── api/
├── data/
├── non-functional/
└── testing/
```

Les sous-dossiers seront créés avec leur contenu lorsqu'un besoin réel sera
documenté. Ils ne doivent pas être remplis de fichiers vides ou de modèles
inutiles.

## Types de spécifications

- `functional/` : comportements métier attendus ;
- `api/` : routes, payloads, réponses et erreurs HTTP ;
- `data/` : modèle de données et contraintes de persistance ;
- `non-functional/` : performance, sécurité, disponibilité et observabilité ;
- `testing/` : stratégie et scénarios de validation.

## Différence avec les autres documents

| Document | Rôle |
| --- | --- |
| `specs/` | Ce que le système doit faire |
| `architecture.md` | Comment le système est organisé |
| `decisions/` | Pourquoi une décision technique a été prise |
| `roadmap.md` | Dans quel ordre les travaux sont réalisés |
| `handoff.md` | Où en est le travail et comment le reprendre |

## Règles d'évolution

- Une spécification active garde un nom stable.
- Ne pas créer `spec-v1`, `spec-v2` ou `spec-final`.
- Modifier la spécification si le même besoin évolue.
- Utiliser une ADR lorsqu'un changement de spécification impose une décision
  technique importante.
- Mettre à jour les tests et le `handoff.md` lorsque le comportement change.
