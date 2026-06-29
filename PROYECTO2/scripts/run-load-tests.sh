#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

USERS="${USERS:-50}"
SPAWN_RATE="${SPAWN_RATE:-10}"
DURATION="${DURATION:-60s}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
RESULTS_DIR="${RESULTS_DIR:-$PROJECT_ROOT/evidence/locust/$RUN_ID}"
GATEWAY_ADDRESS="${GATEWAY_ADDRESS:-$(kubectl get gateway rust-api-gateway -n "$NAMESPACE" -o jsonpath='{.status.addresses[0].value}')}"
HOST="${HOST:-http://$GATEWAY_ADDRESS}"

mkdir -p "$RESULTS_DIR"
ORIGINAL_GO_D2_REPLICAS="$(kubectl get deployment go-d2 -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')"

restore_state() {
  kubectl patch hpa rust-api -n "$NAMESPACE" --type merge \
    -p '{"spec":{"minReplicas":1,"maxReplicas":3}}' >/dev/null 2>&1 || true
  kubectl scale deployment/go-d2 -n "$NAMESPACE" \
    --replicas="$ORIGINAL_GO_D2_REPLICAS" >/dev/null 2>&1 || true
}
trap restore_state EXIT

# Se fija Rust en una réplica para aislar la variable del experimento. La prueba
# independiente de validate-hpa.sh demuestra el escalamiento obligatorio 1 -> 3 -> 1.
kubectl patch hpa rust-api -n "$NAMESPACE" --type merge \
  -p '{"spec":{"minReplicas":1,"maxReplicas":1}}' >/dev/null
kubectl wait hpa/rust-api -n "$NAMESPACE" \
  --for=jsonpath='{.status.currentReplicas}'=1 --timeout=180s

run_case() {
  local replicas="$1"
  local prefix="go-d2-$replicas"
  echo "==> Locust con Go D2 en $replicas réplica(s)"
  kubectl scale deployment/go-d2 -n "$NAMESPACE" --replicas="$replicas" >/dev/null
  kubectl rollout status deployment/go-d2 -n "$NAMESPACE" --timeout=180s

  docker run --rm \
    --user "$(id -u):$(id -g)" \
    -v "$RESULTS_DIR:/results" \
    "$LOCUST_IMAGE" \
    --headless \
    --users "$USERS" \
    --spawn-rate "$SPAWN_RATE" \
    --run-time "$DURATION" \
    --host "$HOST" \
    --csv "/results/$prefix" \
    --html "/results/$prefix.html"
}

run_case 1
run_case 2

python3 "$PROJECT_ROOT/scripts/compare-locust.py" \
  "$RESULTS_DIR/go-d2-1_stats.csv" \
  "$RESULTS_DIR/go-d2-2_stats.csv" >"$RESULTS_DIR/COMPARACION.md"

restore_state
trap - EXIT
echo "Evidencia guardada en $RESULTS_DIR"
