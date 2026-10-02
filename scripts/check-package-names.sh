#!/bin/sh
# Refuse un paquet du module dont le nom est celui d'un paquet de la bibliothèque
# standard ou d'une dépendance (ADR 0008). Une exception se déclare, avec sa
# justification, dans scripts/package-name-exceptions.txt.
set -eu
cd "$(dirname "$0")/.."

exceptions=scripts/package-name-exceptions.txt
module=$(go list -m)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# Chaque go list s'exécute seul : dans un pipeline, son échec passerait inaperçu.
go list -f '{{.Name}}' std > "$work/std.raw"
go list -deps -test -f '{{if not .Standard}}{{.Name}} {{.ImportPath}}{{end}}' ./... > "$work/deps.raw"
go list -f '{{.ImportPath}} {{.Name}}' ./... > "$work/own.raw"

sort -u "$work/std.raw" > "$work/std"
awk -v module="$module" '$2 != module && index($2, module "/") != 1' "$work/deps.raw" | sort -u > "$work/deps"
awk '$2 != "main"' "$work/own.raw" > "$work/own"

awk -F: '
	/^[[:space:]]*(#|$)/ { next }
	{
		path = $1; gsub(/^[[:space:]]+|[[:space:]]+$/, "", path)
		reason = substr($0, index($0, ":") + 1); gsub(/^[[:space:]]+|[[:space:]]+$/, "", reason)
		if (index($0, ":") == 0 || reason == "") { print "ERROR: exception without justification: " $0 > "/dev/stderr"; bad = 1; next }
		print path
	}
	END { exit bad }
' "$exceptions" > "$work/excepted"

status=0
while read -r path name; do
	collides=""
	if grep -qx "$name" "$work/std"; then collides="the standard library"; fi
	if [ -z "$collides" ] && awk -v n="$name" '$1 == n { found = 1 } END { exit !found }' "$work/deps"; then
		collides="a dependency ($(awk -v n="$name" '$1 == n { print $2 }' "$work/deps" | head -1))"
	fi
	if [ -n "$collides" ]; then
		if grep -qx "$path" "$work/excepted"; then
			echo "$path" >> "$work/used"
		else
			echo "ERROR: package $path is named $name, like a package of $collides; rename it or justify it in $exceptions"
			status=1
		fi
	fi
done < "$work/own"

while read -r path; do
	if ! grep -qx "$path" "$work/used" 2>/dev/null; then
		echo "ERROR: exception for $path in $exceptions matches no collision; remove it"
		status=1
	fi
done < "$work/excepted"

exit "$status"
