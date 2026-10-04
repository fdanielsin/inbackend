#!/bin/sh
# Lit sur l'entrée standard les lignes d'un hook pre-push
# (<ref locale> <sha local> <ref distante> <sha distant>) et refuse tout push vers
# refs/heads/main, suppression comprise. Seule exception : RELEASE_TAG nomme un tag
# qui pointe sur le commit poussé. À lancer depuis le dépôt concerné.
set -eu

status=0
while read -r _ local_sha remote_ref _; do
	[ "$remote_ref" = "refs/heads/main" ] || continue
	if [ -n "${RELEASE_TAG:-}" ]; then
		tagged_commit=$(git rev-parse --verify --quiet "refs/tags/${RELEASE_TAG}^{commit}" || true)
		if [ -n "$tagged_commit" ] && [ "$tagged_commit" = "$local_sha" ]; then
			continue
		fi
	fi
	echo "pre-push: push vers main refusé. Pour une release : poser le tag sur le commit poussé," >&2
	echo "pre-push: puis RELEASE_TAG=<tag> git push origin <sha>:refs/heads/main" >&2
	status=1
done
exit $status
