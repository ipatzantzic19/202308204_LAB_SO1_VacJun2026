#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

template="$PROJECT_ROOT/infra/kubernetes/grafana/cloudinit.yaml"
dashboard="$PROJECT_ROOT/infra/kubernetes/grafana/dashboards/quiniela-bra.json"
ssh_key="${GRAFANA_SSH_KEY:-$PROJECT_ROOT/.bin/grafana-vm}"

if [[ ! -f "$ssh_key" ]]; then
  mkdir -p "$(dirname "$ssh_key")"
  ssh-keygen -q -t ed25519 -N '' -C 'grafana-vm-debug' -f "$ssh_key"
fi

python3 - "$template" "$dashboard" "$ssh_key.pub" <<'PY'
import base64
import pathlib
import sys

template = pathlib.Path(sys.argv[1]).read_text()
dashboard = pathlib.Path(sys.argv[2]).read_bytes()
public_key = pathlib.Path(sys.argv[3]).read_text().strip()
replacements = {
    "__GRAFANA_DASHBOARD_BASE64__": base64.b64encode(dashboard).decode(),
    "__GRAFANA_SSH_PUBLIC_KEY__": public_key,
}
for marker, value in replacements.items():
    if template.count(marker) != 1:
        raise SystemExit(f"se esperaba exactamente un marcador {marker}")
    template = template.replace(marker, value)
print(template, end="")
PY
