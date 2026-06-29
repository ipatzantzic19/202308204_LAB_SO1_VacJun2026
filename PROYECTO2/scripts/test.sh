#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

for module in \
  proto \
  go-d1/rest-server \
  go-d1/grpc-client \
  go-d2/grpc-server \
  go-d2/rabbit-writer \
  go-consumer \
  go-metrics-exporter; do
  echo "==> go test $module"
  (cd "$PROJECT_ROOT/$module" && go test ./...)
done

echo "==> cargo test rust-api"
(cd "$PROJECT_ROOT/rust-api" && cargo test)

echo "==> validar locustfile.py"
python3 -c 'import pathlib, sys; path = pathlib.Path(sys.argv[1]); compile(path.read_text(), str(path), "exec")' \
  "$PROJECT_ROOT/locust/locustfile.py"

echo "==> validar dashboard Grafana"
python3 -m json.tool \
  "$PROJECT_ROOT/infra/kubernetes/grafana/dashboards/quiniela-bra.json" >/dev/null
"$PROJECT_ROOT/scripts/render-grafana-cloudinit.sh" | grep -qv '__GRAFANA_DASHBOARD_BASE64__'
"$PROJECT_ROOT/scripts/render-grafana-cloudinit.sh" | python3 -c 'import sys, yaml; yaml.safe_load(sys.stdin)'

echo "==> validar que cloud-init no contenga credenciales estáticas"
if grep -Eq 'ssh_pwauth:[[:space:]]*true|--env GF_SECURITY_ADMIN_PASSWORD=[A-Za-z0-9]' \
  "$PROJECT_ROOT/infra/kubernetes/grafana/cloudinit.yaml"; then
  echo "cloud-init contiene una credencial estática o habilita SSH por contraseña" >&2
  exit 1
fi
grep -q '__GRAFANA_SSH_PUBLIC_KEY__' "$PROJECT_ROOT/infra/kubernetes/grafana/cloudinit.yaml"
