#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

kubectl kustomize "$PROJECT_ROOT/infra/kubernetes" >/dev/null
kubectl apply -k "$PROJECT_ROOT/infra/kubernetes" --dry-run=server >/dev/null
curl --fail --silent --show-error "https://$REGISTRY/v2/" >/dev/null

kubectl get pods -n "$RABBITMQ_NAMESPACE"
kubectl get pods -n "$NAMESPACE"
kubectl get gateway,httproute -n "$NAMESPACE"
