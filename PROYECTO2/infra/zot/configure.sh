#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ZOT_DOMAIN="${ZOT_DOMAIN:-zot.35-226-224-23.sslip.io}"
# Digests correspondientes a Zot v2.1.18 y Caddy 2.10.2.
ZOT_IMAGE="${ZOT_IMAGE:-ghcr.io/project-zot/zot-linux-amd64@sha256:34f18f783037f967dba10df02f9d4086c4d626f5643ef9f5e51e4a4547280a0b}"
CADDY_IMAGE="${CADDY_IMAGE:-caddy@sha256:c3d7ee5d2b11f9dc54f947f68a734c84e9c9666c92c88a7f30b9cba5da182adb}"
ZOT_MOUNT="${ZOT_MOUNT:-/var/lib/zot}"
ZOT_DATA_DIR="$ZOT_MOUNT/registry"
CONFIG_DIR="${CONFIG_DIR:-/etc/zot}"
ZOT_REQUIRE_MOUNTPOINT="${ZOT_REQUIRE_MOUNTPOINT:-true}"
LEGACY_NAME="zot-legacy"
migrated_legacy=false

rollback() {
  status=$?
  trap - ERR
  if [[ "$migrated_legacy" == true ]]; then
    echo "La nueva instancia falló; restaurando el contenedor Zot anterior..." >&2
    sudo docker rm -f zot >/dev/null 2>&1 || true
    sudo docker rename "$LEGACY_NAME" zot
    sudo docker network connect zot-registry zot >/dev/null 2>&1 || true
    sudo docker start zot >/dev/null
  fi
  exit "$status"
}
trap rollback ERR

if [[ "$ZOT_REQUIRE_MOUNTPOINT" == true ]] && ! mountpoint -q "$ZOT_MOUNT"; then
  echo "$ZOT_MOUNT no es un punto de montaje persistente." >&2
  exit 1
fi
if [[ "$ZOT_REQUIRE_MOUNTPOINT" != true && "$ZOT_REQUIRE_MOUNTPOINT" != false ]]; then
  echo "ZOT_REQUIRE_MOUNTPOINT debe ser true o false." >&2
  exit 1
fi

for file in config.json Caddyfile; do
  if [[ ! -f "$SCRIPT_DIR/$file" ]]; then
    echo "Falta $SCRIPT_DIR/$file" >&2
    exit 1
  fi
done

sudo install -d -m 0755 "$CONFIG_DIR" "$ZOT_DATA_DIR"
sudo install -m 0644 "$SCRIPT_DIR/config.json" "$CONFIG_DIR/config.json"
sudo install -m 0644 "$SCRIPT_DIR/Caddyfile" "$CONFIG_DIR/Caddyfile"
if [[ -f "$SCRIPT_DIR/cleanup-legacy.sh" ]]; then
  sudo install -m 0755 "$SCRIPT_DIR/cleanup-legacy.sh" /usr/local/sbin/zot-cleanup-legacy
fi

sudo docker network inspect zot-registry >/dev/null 2>&1 || \
  sudo docker network create zot-registry >/dev/null
sudo docker pull "$ZOT_IMAGE" >/dev/null
sudo docker pull "$CADDY_IMAGE" >/dev/null

# Primera ejecución: copiar el catálogo que vive en la capa escribible del
# contenedor antiguo. El contenedor se conserva detenido como rollback.
if sudo docker container inspect zot >/dev/null 2>&1; then
  current_storage_source="$(sudo docker inspect zot --format \
    '{{range .Mounts}}{{if eq .Destination "/var/lib/registry"}}{{.Source}}{{end}}{{end}}')"
  if [[ "$current_storage_source" != "$ZOT_DATA_DIR" ]]; then
    if sudo docker container inspect "$LEGACY_NAME" >/dev/null 2>&1; then
      echo "Ya existe $LEGACY_NAME; no se sobrescribirá el respaldo." >&2
      exit 1
    fi

    echo "Migrando el catálogo del contenedor legado al disco persistente..."
    sudo docker stop zot >/dev/null
    if [[ -z "$(sudo find "$ZOT_DATA_DIR" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
      sudo docker cp zot:/var/lib/registry/. "$ZOT_DATA_DIR/"
    else
      echo "$ZOT_DATA_DIR no está vacío; se cancela para evitar una mezcla de catálogos." >&2
      sudo docker start zot >/dev/null
      exit 1
    fi
    sudo docker rename zot "$LEGACY_NAME"
    migrated_legacy=true
  else
    sudo docker rm -f zot >/dev/null
  fi
fi

sudo docker rm -f zot >/dev/null 2>&1 || true
sudo docker run -d \
  --name zot \
  --restart unless-stopped \
  --network zot-registry \
  --mount "type=bind,src=$ZOT_DATA_DIR,dst=/var/lib/registry" \
  --mount "type=bind,src=$CONFIG_DIR/config.json,dst=/etc/zot/config.json,readonly" \
  "$ZOT_IMAGE" >/dev/null

sudo docker volume inspect caddy-data >/dev/null 2>&1 || sudo docker volume create caddy-data >/dev/null
sudo docker volume inspect caddy-config >/dev/null 2>&1 || sudo docker volume create caddy-config >/dev/null
sudo docker rm -f caddy >/dev/null 2>&1 || true
sudo docker run -d \
  --name caddy \
  --restart unless-stopped \
  --network zot-registry \
  -e "ZOT_DOMAIN=$ZOT_DOMAIN" \
  -p 80:80 \
  -p 443:443 \
  --mount "type=bind,src=$CONFIG_DIR/Caddyfile,dst=/etc/caddy/Caddyfile,readonly" \
  -v caddy-data:/data \
  -v caddy-config:/config \
  "$CADDY_IMAGE" >/dev/null

for _ in {1..30}; do
  if curl --fail --silent --show-error "https://$ZOT_DOMAIN/v2/" >/dev/null; then
    break
  fi
  sleep 2
done
curl --fail --silent --show-error "https://$ZOT_DOMAIN/v2/" >/dev/null

published_ports="$(sudo docker inspect zot --format '{{json .HostConfig.PortBindings}}')"
if [[ "$published_ports" != "{}" && "$published_ports" != "null" ]]; then
  echo "Zot aún publica puertos en la VM: $published_ports" >&2
  exit 1
fi
trap - ERR

echo "Zot disponible en https://$ZOT_DOMAIN"
echo "Datos persistentes: $ZOT_DATA_DIR"
echo "HTTP :5000 queda accesible solo dentro de la red Docker zot-registry."
if sudo docker container inspect "$LEGACY_NAME" >/dev/null 2>&1; then
  echo "Respaldo detenido: $LEGACY_NAME. Valide y luego ejecute sudo zot-cleanup-legacy."
fi
