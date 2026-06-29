#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

kubectl kustomize "$PROJECT_ROOT/infra/kubernetes" >/dev/null
kubectl apply -k "$PROJECT_ROOT/infra/kubernetes" --dry-run=server >/dev/null
curl --fail --silent --show-error "https://$REGISTRY/v2/" >/dev/null

kubectl get pods -n "$RABBITMQ_NAMESPACE"
kubectl get kubevirt -n kubevirt
kubectl get virtualmachine,virtualmachineinstance -n "$NAMESPACE"
kubectl get pods -n "$NAMESPACE"
kubectl get deployment/go-metrics-exporter service/go-metrics-exporter-service \
  horizontalpodautoscaler/rust-api -n "$NAMESPACE"
kubectl get gateway,httproute -n "$NAMESPACE"

curl --fail --silent --show-error \
  "https://$REGISTRY/v2/sopes1/go-metrics-exporter/tags/list" | grep -q '"v1"'

"$PROJECT_ROOT/scripts/check-grafana.sh"

# El auxiliar dispensó el OCI Artifact. Si existe, se verifica como evidencia adicional,
# pero su ausencia no invalida el flujo obligatorio.
if curl --fail --silent --show-error \
  "https://$REGISTRY/v2/sopes1/prediction-proto/tags/list" | grep -q '"v1"'; then
  "$PROJECT_ROOT/scripts/oci-artifact.sh" verify
else
  echo "Aviso: OCI Artifact no publicado (requisito dispensado por el auxiliar)." >&2
fi
