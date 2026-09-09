#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

GITHUB_REMOTE="${PROD_REMOTE:-github}"
GITHUB_URL="${PROD_REPOSITORY_URL:-https://github.com/jefersonMarques/apresentacao-viagate.git}"
BRANCH="${PROD_BRANCH:-main}"
ENV_FILE="${PROD_ENV_FILE:-/etc/viagate-commercial/.env}"
RELEASES_DIR="${PROD_RELEASES_DIR:-/opt/viagate-commercial/releases}"
CURRENT_LINK="${PROD_CURRENT_LINK:-/opt/viagate-commercial/current}"
SERVICE="${PROD_SERVICE:-viagate-commercial}"
BACKUP_DIR="${PROD_BACKUP_DIR:-/var/backups/viagate-commercial}"
SERVICE_USER="${PROD_SERVICE_USER:-viagate}"

log() {
	printf '\n==> %s\n' "$*"
}

fail() {
	printf '\nERRO: %s\n' "$*" >&2
	exit 1
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "comando obrigatório não encontrado: $1"
}

show_status() {
	local deployed="indisponível"
	if [[ -r "$CURRENT_LINK/GIT_SHA" ]]; then
		deployed="$(cat "$CURRENT_LINK/GIT_SHA")"
	fi

	printf 'release: %s\n' "$(readlink -f "$CURRENT_LINK" 2>/dev/null || printf 'indisponível')"
	printf 'sha: %s\n' "$deployed"
	sudo systemctl --no-pager --full status "$SERVICE" || true

	printf '\nhealth:\n'
	curl -fsS -o /dev/null -w 'healthz: %{http_code}\n' http://127.0.0.1:8081/healthz || true
	curl -fsS -o /dev/null -w 'readyz: %{http_code}\n' http://127.0.0.1:8081/readyz || true
}

if [[ "${1:-}" == "status" ]]; then
	require_command sudo
	require_command curl
	show_status
	exit 0
fi

for command_name in git sudo tar curl sha256sum; do
	require_command "$command_name"
done

[[ -f "$ENV_FILE" ]] || fail "arquivo de produção não encontrado: $ENV_FILE"

if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
	fail "existem alterações locais rastreadas. Commit, reverta ou guarde as alterações antes do deploy."
fi

INITIAL_SHA="$(git rev-parse HEAD)"

log "Sincronizando $BRANCH com o GitHub"
if git remote get-url "$GITHUB_REMOTE" >/dev/null 2>&1; then
	git remote set-url "$GITHUB_REMOTE" "$GITHUB_URL"
else
	git remote add "$GITHUB_REMOTE" "$GITHUB_URL"
fi

git fetch --prune "$GITHUB_REMOTE" "$BRANCH"
git switch -C "$BRANCH" "$GITHUB_REMOTE/$BRANCH"

TARGET_SHA="$(git rev-parse HEAD)"
if [[ "${PROD_SCRIPT_SYNCED:-0}" != "1" && "$INITIAL_SHA" != "$TARGET_SHA" ]]; then
	log "Reexecutando o script atualizado"
	exec env PROD_SCRIPT_SYNCED=1 bash "$ROOT_DIR/scripts/prod.sh" "$@"
fi

VERSION="$(git rev-parse --short=12 HEAD)"
RELEASE_NAME="viagate-commercial-${VERSION}-linux-amd64"
ARCHIVE="$ROOT_DIR/dist/${RELEASE_NAME}.tar.gz"
CHECKSUM="${ARCHIVE}.sha256"
RELEASE="$RELEASES_DIR/$RELEASE_NAME"

log "Executando checks e gerando release $VERSION"
bash ./scripts/build-release.sh

[[ -f "$ARCHIVE" ]] || fail "bundle não foi gerado: $ARCHIVE"
[[ -f "$CHECKSUM" ]] || fail "checksum não foi gerado: $CHECKSUM"

(
	cd "$ROOT_DIR/dist"
	sha256sum -c "$(basename "$CHECKSUM")"
)

log "Preparando release"
sudo mkdir -p "$RELEASES_DIR"

if [[ -d "$RELEASE" ]]; then
	[[ -r "$RELEASE/GIT_SHA" ]] || fail "release existente sem GIT_SHA: $RELEASE"
	[[ "$(cat "$RELEASE/GIT_SHA")" == "$TARGET_SHA" ]] || fail "release existente aponta para outro SHA: $RELEASE"
else
	sudo tar -xzf "$ARCHIVE" -C "$RELEASES_DIR"
	sudo chown -R "$SERVICE_USER:$SERVICE_USER" "$RELEASE"
fi

[[ "$(cat "$RELEASE/GIT_SHA")" == "$TARGET_SHA" ]] || fail "GIT_SHA da release não corresponde ao código validado"

log "Executando preflight"
sudo -u "$SERVICE_USER" env RELEASE="$RELEASE" ENV_FILE="$ENV_FILE" bash -lc '
	set -Eeuo pipefail
	set -a
	source "$ENV_FILE"
	set +a
	cd "$RELEASE"
	./preflight
'

if [[ "${PROD_SKIP_BACKUP:-0}" != "1" ]]; then
	require_command pg_dump
	log "Gerando backup PostgreSQL"
	sudo install -d -o "$SERVICE_USER" -g "$SERVICE_USER" -m 750 "$BACKUP_DIR"
	BACKUP="$BACKUP_DIR/viagate-before-${VERSION}-$(date +%Y%m%d-%H%M%S).dump"
	sudo -u "$SERVICE_USER" env BACKUP="$BACKUP" ENV_FILE="$ENV_FILE" bash -lc '
		set -Eeuo pipefail
		set -a
		source "$ENV_FILE"
		set +a
		pg_dump --format=custom --file="$BACKUP" "$DATABASE_URL"
	'
	printf 'backup: %s\n' "$BACKUP"
fi

log "Aplicando migrations forward-only"
sudo -u "$SERVICE_USER" env RELEASE="$RELEASE" ENV_FILE="$ENV_FILE" bash -lc '
	set -Eeuo pipefail
	set -a
	source "$ENV_FILE"
	set +a
	cd "$RELEASE"
	./migrate up
'

PREVIOUS_RELEASE="$(readlink -f "$CURRENT_LINK" 2>/dev/null || true)"

log "Ativando release"
sudo ln -sfn "$RELEASE" "${CURRENT_LINK}.new"
sudo mv -Tf "${CURRENT_LINK}.new" "$CURRENT_LINK"
sudo systemctl restart "$SERVICE"

log "Validando serviço"
if ! sudo systemctl is-active --quiet "$SERVICE"; then
	printf 'release anterior: %s\n' "$PREVIOUS_RELEASE" >&2
	fail "serviço não ficou ativo após o restart"
fi

if ! curl -fsS http://127.0.0.1:8081/healthz -o /dev/null; then
	printf 'release anterior: %s\n' "$PREVIOUS_RELEASE" >&2
	fail "healthz falhou após o deploy"
fi

if ! curl -fsS http://127.0.0.1:8081/readyz -o /dev/null; then
	printf 'release anterior: %s\n' "$PREVIOUS_RELEASE" >&2
	fail "readyz falhou após o deploy"
fi

DEPLOYED_SHA="$(cat "$CURRENT_LINK/GIT_SHA")"
[[ "$DEPLOYED_SHA" == "$TARGET_SHA" ]] || fail "SHA ativo não corresponde ao SHA esperado"

log "Deploy concluído"
printf 'release: %s\n' "$RELEASE"
printf 'sha: %s\n' "$DEPLOYED_SHA"
printf 'healthz: ok\n'
printf 'readyz: ok\n'
