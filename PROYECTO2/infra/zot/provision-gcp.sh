#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VM_NAME="${VM_NAME:-zot-registry}"
ZONE="${ZONE:-us-central1-a}"
TARGET_BOOT_GB="${TARGET_BOOT_GB:-20}"
FREE_STANDARD_GB="${FREE_STANDARD_GB:-30}"
ZOT_DOMAIN="${ZOT_DOMAIN:-zot.35-226-224-23.sslip.io}"

if [[ ! "$TARGET_BOOT_GB" =~ ^[1-9][0-9]*$ ]]; then
  echo "TARGET_BOOT_GB debe ser un entero positivo." >&2
  exit 1
fi
if [[ ! "$ZONE" =~ ^(us-central1|us-east1|us-west1)- ]]; then
  echo "$ZONE no pertenece a una región cubierta por Always Free." >&2
  exit 1
fi

read -r boot_disk boot_device < <(gcloud compute instances describe "$VM_NAME" --zone="$ZONE" \
  --format='csv[no-heading,separator=" "](disks[0].source.basename(),disks[0].deviceName)')
disk_type="$(gcloud compute disks describe "$boot_disk" --zone="$ZONE" --format='value(type.basename())')"
current_size="$(gcloud compute disks describe "$boot_disk" --zone="$ZONE" --format='value(sizeGb)')"

if [[ "$disk_type" != "pd-standard" ]]; then
  echo "El disco $boot_disk es $disk_type; el nivel gratuito exige pd-standard." >&2
  exit 1
fi

standard_total="$(gcloud compute disks list --filter='type:pd-standard' --format='value(sizeGb)' \
  | awk '{ total += $1 } END { print total + 0 }')"
projected_total=$((standard_total - current_size + TARGET_BOOT_GB))
if (( projected_total > FREE_STANDARD_GB )); then
  echo "La ampliación dejaría $projected_total GB de pd-standard; el límite gratuito es $FREE_STANDARD_GB GB." >&2
  exit 1
fi

if (( current_size < TARGET_BOOT_GB )); then
  gcloud compute disks resize "$boot_disk" --zone="$ZONE" \
    --size="${TARGET_BOOT_GB}GB" --quiet
fi
gcloud compute instances set-disk-auto-delete "$VM_NAME" --zone="$ZONE" \
  --device-name="$boot_device" --no-auto-delete --quiet

remote_dir="/tmp/zot-config-${USER}"
gcloud compute ssh "$VM_NAME" --zone="$ZONE" --command="rm -rf '$remote_dir' && mkdir -p '$remote_dir'"
gcloud compute scp "$SCRIPT_DIR"/{Caddyfile,config.json,configure.sh,prepare-root-storage.sh,cleanup-legacy.sh} \
  "$VM_NAME:$remote_dir/" --zone="$ZONE"
gcloud compute ssh "$VM_NAME" --zone="$ZONE" --command="
  set -e
  chmod +x '$remote_dir/'*.sh
  sudo MIN_ROOT_GB='$TARGET_BOOT_GB' '$remote_dir/prepare-root-storage.sh'
  sudo docker image prune -a -f
  ZOT_DOMAIN='$ZOT_DOMAIN' ZOT_REQUIRE_MOUNTPOINT=false '$remote_dir/configure.sh'
"

echo "Zot usa el disco de arranque pd-standard de $TARGET_BOOT_GB GB."
echo "Consumo pd-standard proyectado del proyecto: $projected_total/$FREE_STANDARD_GB GB gratuitos."
echo "El auto-delete del disco $boot_disk quedó desactivado."
