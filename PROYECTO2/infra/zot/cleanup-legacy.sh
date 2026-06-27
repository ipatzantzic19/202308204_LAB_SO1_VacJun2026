#!/usr/bin/env bash
set -Eeuo pipefail

ZOT_DOMAIN="${ZOT_DOMAIN:-zot.35-226-224-23.sslip.io}"

curl --fail --silent --show-error "https://$ZOT_DOMAIN/v2/_catalog"
echo
sudo docker inspect zot --format 'Zot activo: imagen={{.Config.Image}} mounts={{json .Mounts}} puertos={{json .HostConfig.PortBindings}}'

if ! sudo docker container inspect zot-legacy >/dev/null 2>&1; then
  echo "No existe el contenedor zot-legacy."
  exit 0
fi

if [[ "${CONFIRM_DELETE_LEGACY:-}" != "yes" ]]; then
  echo "Catálogo accesible. Para liberar la capa antigua use:" >&2
  echo "CONFIRM_DELETE_LEGACY=yes $0" >&2
  exit 1
fi

sudo docker rm zot-legacy >/dev/null
echo "Contenedor legado eliminado; el catálogo activo permanece en el disco persistente."
