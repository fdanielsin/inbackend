#!/bin/sh
# Rétention des versions du paquet GHCR (voir docs/deployment.md) : garde les versions
# taguées develop ou v*, les 10 versions taguées sha les plus récentes, et supprime le
# reste, versions sans tag comprises. Sans PRUNE_IMAGE_VERSIONS=true, n'affiche que ce
# qui serait supprimé.
# Usage : OWNER=<compte> [PRUNE_IMAGE_VERSIONS=true] prune-image-versions.sh [versions.json]
# Le JSON de versions (réponse de l'API, pages concaténées acceptées) est lu depuis le
# fichier donné (alors jamais de suppression : PRUNE_IMAGE_VERSIONS=true est refusé),
# sinon depuis gh api.
set -eu

package=${PACKAGE:-inbackend}
keep=${KEEP_SHA_VERSIONS:-10}
# Compte utilisateur : pour une organisation, remplacer users/ par orgs/.
endpoint="users/${OWNER:?OWNER is required}/packages/container/$package/versions"

dry_run=true
if [ $# -ge 1 ]; then
	if [ "${PRUNE_IMAGE_VERSIONS:-}" = "true" ]; then
		echo "ERROR: a versions file is for a dry run only; unset PRUNE_IMAGE_VERSIONS" >&2
		exit 2
	fi
	versions=$(cat "$1")
else
	versions=$(gh api --paginate "$endpoint")
	[ "${PRUNE_IMAGE_VERSIONS:-}" = "true" ] && dry_run=false
fi

# Les pages de gh api --paginate arrivent concaténées : -s les fusionne en un tableau.
doomed=$(printf '%s' "$versions" | jq -rs --argjson keep "$keep" '
	add // []
	| map({id, created_at, tags: (.metadata.container.tags // [])})
	| map(select((.tags | any(. == "develop" or startswith("v"))) | not)) as $candidates
	| ($candidates | map(select(.tags | length == 0))
		| map(. + {reason: "untagged"})) as $untagged
	| ($candidates | map(select(.tags | length > 0)) | sort_by(.created_at) | reverse
		| .[$keep:] | map(. + {reason: "older than the \($keep) newest sha versions"})) as $old
	| ($untagged + $old)[]
	| "\(.id)\t\(.created_at)\t\(.tags | join(","))\t\(.reason)"')

if [ -z "$doomed" ]; then
	echo "Nothing to delete."
	exit 0
fi

if [ "$dry_run" = true ]; then
	echo "Dry run (PRUNE_IMAGE_VERSIONS is not true, or a versions file was given). Would delete:"
	printf '%s\n' "$doomed"
	exit 0
fi

echo "Deleting:"
printf '%s\n' "$doomed"
for id in $(printf '%s\n' "$doomed" | cut -f1); do
	gh api --method DELETE "$endpoint/$id"
done
