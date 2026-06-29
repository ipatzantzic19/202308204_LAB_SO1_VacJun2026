#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

VIRTCTL="$PROJECT_ROOT/.bin/virtctl"
GRAFANA_LOCAL_PORT="${GRAFANA_LOCAL_PORT:-13000}"
PROMETHEUS_LOCAL_PORT="${PROMETHEUS_LOCAL_PORT:-19090}"
log_file="$(mktemp)"
health_file="$(mktemp)"
pf_pid=""

cleanup() {
  if [[ -n "$pf_pid" ]]; then
    kill "$pf_pid" >/dev/null 2>&1 || true
    wait "$pf_pid" >/dev/null 2>&1 || true
  fi
  rm -f "$log_file" "$health_file"
}
trap cleanup EXIT

[[ -x "$VIRTCTL" ]] || {
  echo "No se encontró $VIRTCTL; ejecutar make kubevirt." >&2
  exit 1
}

start_port_forward() {
  : >"$log_file"
  "$VIRTCTL" port-forward -n "$NAMESPACE" vmi/grafana-vm \
    "$GRAFANA_LOCAL_PORT:3000" "$PROMETHEUS_LOCAL_PORT:9090" >>"$log_file" 2>&1 &
  pf_pid=$!
  sleep 1
}

for _ in $(seq 1 180); do
  if [[ -z "$pf_pid" ]] || ! kill -0 "$pf_pid" >/dev/null 2>&1; then
    wait "$pf_pid" >/dev/null 2>&1 || true
    pf_pid=""
    start_port_forward
  fi
  if curl --fail --silent --show-error --connect-timeout 2 \
      "http://127.0.0.1:$GRAFANA_LOCAL_PORT/api/health" >"$health_file" 2>/dev/null \
    && curl --fail --silent --show-error --connect-timeout 2 \
      "http://127.0.0.1:$PROMETHEUS_LOCAL_PORT/-/ready" >/dev/null 2>&1; then
    version="$(jq -r '.version' "$health_file")"
    [[ "$version" == "11.5.2" ]] || {
      echo "Versión de Grafana inesperada: $version (se esperaba 11.5.2)" >&2
      exit 1
    }
    final_url="$(curl --location --silent --output /dev/null --write-out '%{url_effective}' \
      "http://127.0.0.1:$GRAFANA_LOCAL_PORT/d/quiniela-bra-202308204/quiniela-mundial-2026-brasil-bra")"
    [[ "$final_url" != *'/login'* ]] || {
      echo "Grafana redirige al login; acceso anónimo no está activo." >&2
      exit 1
    }
    echo "Grafana $version y Prometheus listos; dashboard accesible sin login."
    exit 0
  fi
  sleep 2
done

echo "Grafana/Prometheus no quedaron listos en seis minutos." >&2
cat "$log_file" >&2
exit 1
