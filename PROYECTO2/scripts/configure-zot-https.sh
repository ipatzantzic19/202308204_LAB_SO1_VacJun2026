#!/usr/bin/env bash
set -euo pipefail

ZOT_DOMAIN="${ZOT_DOMAIN:-zot.35-226-224-23.sslip.io}"
CADDYFILE="${CADDYFILE:-/tmp/Caddyfile}"

if [[ ! -f "$CADDYFILE" ]]; then
  echo "No existe $CADDYFILE" >&2
  exit 1
fi

sudo docker network inspect zot-registry >/dev/null 2>&1 || \
  sudo docker network create zot-registry >/dev/null

if ! sudo docker inspect zot --format '{{json .NetworkSettings.Networks}}' | grep -q 'zot-registry'; then
  sudo docker network connect zot-registry zot
fi

sudo docker volume inspect caddy-data >/dev/null 2>&1 || sudo docker volume create caddy-data >/dev/null
sudo docker volume inspect caddy-config >/dev/null 2>&1 || sudo docker volume create caddy-config >/dev/null
sudo docker rm -f caddy >/dev/null 2>&1 || true
sudo docker run -d \
  --name caddy \
  --restart unless-stopped \
  --network zot-registry \
  -p 80:80 \
  -p 443:443 \
  -v "$CADDYFILE:/etc/caddy/Caddyfile:ro" \
  -v caddy-data:/data \
  -v caddy-config:/config \
  caddy:2.10.2 >/dev/null

echo "Caddy iniciado para https://$ZOT_DOMAIN"
