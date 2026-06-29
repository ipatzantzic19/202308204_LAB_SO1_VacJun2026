#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

kubectl apply -f "$PROJECT_ROOT/infra/kubernetes/namespace.yaml"
"$PROJECT_ROOT/scripts/install-kubevirt.sh"
kubectl apply -f "https://github.com/rabbitmq/cluster-operator/releases/download/v${RABBITMQ_OPERATOR_VERSION}/cluster-operator.yml"
kubectl rollout status deployment/rabbitmq-cluster-operator \
  -n "$RABBITMQ_NAMESPACE" --timeout=180s

kubectl apply -f "$PROJECT_ROOT/infra/kubernetes/rabbitmq/cluster.yaml"
kubectl rollout status statefulset/rabbitmq-cluster-server \
  -n "$RABBITMQ_NAMESPACE" --timeout=300s

"$PROJECT_ROOT/scripts/sync-rabbitmq-secret.sh"

grafana_cloudinit="$(mktemp)"
trap 'rm -f "$grafana_cloudinit"' EXIT
"$PROJECT_ROOT/scripts/render-grafana-cloudinit.sh" >"$grafana_cloudinit"
grafana_cloudinit_checksum="$(sha256sum "$grafana_cloudinit" | awk '{print $1}')"
previous_grafana_checksum="$(kubectl get virtualmachine grafana-vm -n "$NAMESPACE" \
  -o jsonpath='{.metadata.annotations.sopes1\.usac\.edu\.gt/cloudinit-checksum}' 2>/dev/null || true)"

kubectl create secret generic grafana-cloudinit -n "$NAMESPACE" \
  --from-file=userdata="$grafana_cloudinit" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -k "$PROJECT_ROOT/infra/kubernetes"

kubectl annotate virtualmachine grafana-vm -n "$NAMESPACE" \
  "sopes1.usac.edu.gt/cloudinit-checksum=$grafana_cloudinit_checksum" --overwrite
if [[ "$previous_grafana_checksum" != "$grafana_cloudinit_checksum" ]]; then
  kubectl delete virtualmachineinstance grafana-vm -n "$NAMESPACE" \
    --ignore-not-found --wait=true
fi

kubectl wait virtualmachineinstance/valkey-vm -n "$NAMESPACE" \
  --for=condition=Ready --timeout=600s

until kubectl get virtualmachineinstance/grafana-vm -n "$NAMESPACE" >/dev/null 2>&1; do
  sleep 2
done
kubectl wait virtualmachineinstance/grafana-vm -n "$NAMESPACE" \
  --for=condition=Ready --timeout=600s

"$PROJECT_ROOT/scripts/check-grafana.sh"

for deployment in go-d2 go-d1 rust-api go-consumer go-metrics-exporter; do
  kubectl rollout status "deployment/$deployment" -n "$NAMESPACE" --timeout=300s
done
