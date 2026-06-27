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
kubectl apply -k "$PROJECT_ROOT/infra/kubernetes"

kubectl wait virtualmachineinstance/valkey-vm -n "$NAMESPACE" \
  --for=condition=Ready --timeout=600s

for deployment in go-d2 go-d1 rust-api go-consumer; do
  kubectl rollout status "deployment/$deployment" -n "$NAMESPACE" --timeout=300s
done
