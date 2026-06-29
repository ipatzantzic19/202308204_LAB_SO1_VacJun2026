#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

USERS="${USERS:-150}"
SPAWN_RATE="${SPAWN_RATE:-25}"
DURATION="${DURATION:-90s}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
RESULTS_DIR="${RESULTS_DIR:-$PROJECT_ROOT/evidence/hpa/$RUN_ID}"
GATEWAY_ADDRESS="${GATEWAY_ADDRESS:-$(kubectl get gateway rust-api-gateway -n "$NAMESPACE" -o jsonpath='{.status.addresses[0].value}')}"
HOST="${HOST:-http://$GATEWAY_ADDRESS}"

mkdir -p "$RESULTS_DIR"
kubectl apply -f "$PROJECT_ROOT/infra/kubernetes/rust-api/hpa.yaml" >/dev/null

start_deadline=$((SECONDS + 300))
while (( SECONDS < start_deadline )); do
  current="$(kubectl get hpa rust-api -n "$NAMESPACE" -o jsonpath='{.status.currentReplicas}')"
  desired="$(kubectl get hpa rust-api -n "$NAMESPACE" -o jsonpath='{.status.desiredReplicas}')"
  [[ "$current" == "1" && "$desired" == "1" ]] && break
  sleep 10
done
if [[ "$current" != "1" || "$desired" != "1" ]]; then
  echo "el HPA no llegó al estado inicial de una réplica" >&2
  exit 1
fi

echo "timestamp_utc,current_replicas,desired_replicas,cpu" >"$RESULTS_DIR/hpa-timeline.csv"
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$RESULTS_DIR:/results" \
  "$LOCUST_IMAGE" \
  --headless --users "$USERS" --spawn-rate "$SPAWN_RATE" --run-time "$DURATION" \
  --host "$HOST" --csv /results/hpa --html /results/hpa.html \
  >"$RESULTS_DIR/locust.log" 2>&1 &
locust_pid=$!

while kill -0 "$locust_pid" 2>/dev/null; do
  printf '%s,' "$(date -u +%FT%TZ)" >>"$RESULTS_DIR/hpa-timeline.csv"
  kubectl get hpa rust-api -n "$NAMESPACE" \
    -o jsonpath='{.status.currentReplicas},{.status.desiredReplicas},{.status.currentMetrics[0].resource.current.averageUtilization}{"\n"}' \
    >>"$RESULTS_DIR/hpa-timeline.csv"
  sleep 5
done
locust_exit=0
wait "$locust_pid" || locust_exit=$?

deadline=$((SECONDS + 300))
scaled_up=false
while (( SECONDS < deadline )); do
  current="$(kubectl get hpa rust-api -n "$NAMESPACE" -o jsonpath='{.status.currentReplicas}')"
  desired="$(kubectl get hpa rust-api -n "$NAMESPACE" -o jsonpath='{.status.desiredReplicas}')"
  printf '%s,%s,%s,%s\n' "$(date -u +%FT%TZ)" "$current" \
    "$desired" \
    "$(kubectl get hpa rust-api -n "$NAMESPACE" -o jsonpath='{.status.currentMetrics[0].resource.current.averageUtilization}')" \
    >>"$RESULTS_DIR/hpa-timeline.csv"
  if [[ "$current" == "3" || "$desired" == "3" ]]; then
    scaled_up=true
  fi
  [[ "$scaled_up" == true && "$current" == "1" && "$desired" == "1" ]] && break
  sleep 10
done

max_observed="$(awk -F, 'NR > 1 {if ($2+0 > max) max=$2+0; if ($3+0 > max) max=$3+0} END {print max+0}' "$RESULTS_DIR/hpa-timeline.csv")"
final_replicas="$(kubectl get hpa rust-api -n "$NAMESPACE" -o jsonpath='{.status.currentReplicas}')"
read -r request_count failure_count failure_rate < <(python3 - "$RESULTS_DIR/hpa_stats.csv" <<'PY'
import csv
import sys
with open(sys.argv[1], newline="") as f:
    row = next(r for r in csv.DictReader(f) if r["Name"] == "Aggregated")
requests = int(row["Request Count"])
failures = int(row["Failure Count"])
rate = failures * 100 / requests if requests else 100
print(requests, failures, f"{rate:.4f}")
PY
)
printf '# Validación HPA Rust API\n\n- Máximo observado: %s réplicas\n- Réplicas al finalizar el enfriamiento: %s\n- Solicitudes: %s\n- Errores: %s (%s%%)\n- Código de salida Locust: %s\n- Objetivo: CPU 30%%, rango 1–3\n' \
  "$max_observed" "$final_replicas" "$request_count" "$failure_count" "$failure_rate" "$locust_exit" \
  >"$RESULTS_DIR/RESULTADO.md"

if [[ "$max_observed" -lt 3 || "$final_replicas" != "1" ]] || \
  ! python3 -c 'import sys; raise SystemExit(float(sys.argv[1]) >= 1)' "$failure_rate"; then
  echo "HPA no completó el ciclo esperado 1 → 3 → 1; revisar $RESULTS_DIR" >&2
  exit 1
fi
echo "HPA validado; evidencia en $RESULTS_DIR"
