#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "Arquivo .env não encontrado. Copie .env.example para .env e ajuste as configurações." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1091
source ./.env
set +a

echo "[1/4] Baixando dependências Go..."
go mod download

echo "[2/4] Gerando componentes templ..."
go run github.com/a-h/templ/cmd/templ@v0.3.943 generate

echo "[3/4] Aplicando migrations..."
go run ./cmd/migrate up

echo "[4/4] Iniciando ViaGate Commercial Platform..."
exec go run ./cmd/server
