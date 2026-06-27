#!/usr/bin/env bash
set -Eeuo pipefail

MIN_ROOT_GB="${MIN_ROOT_GB:-20}"

if [[ $EUID -ne 0 ]]; then
  echo "Ejecute este script con sudo." >&2
  exit 1
fi

root_partition="$(findmnt -n -o SOURCE /)"
parent_name="$(lsblk -n -o PKNAME "$root_partition")"
partition_number="$(cat "/sys/class/block/$(basename "$root_partition")/partition" 2>/dev/null || true)"
if [[ -z "$parent_name" || -z "$partition_number" ]]; then
  echo "No se pudo determinar el disco padre de $root_partition." >&2
  exit 1
fi

root_disk="/dev/$parent_name"
if ! command -v growpart >/dev/null 2>&1; then
  apt-get update
  apt-get install -y --no-install-recommends cloud-guest-utils
fi
growpart "$root_disk" "$partition_number" || true
resize2fs "$root_partition" >/dev/null

root_size_gb="$(df --block-size=1G --output=size / | tail -1 | tr -d ' ')"
if (( root_size_gb < MIN_ROOT_GB - 1 )); then
  echo "La raíz solo expone $root_size_gb GB; se esperaban aproximadamente $MIN_ROOT_GB GB." >&2
  exit 1
fi

install -d -m 0755 /var/lib/zot
echo "Filesystem raíz ampliado; /var/lib/zot persistirá en el disco de arranque."
