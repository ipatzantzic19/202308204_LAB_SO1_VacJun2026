#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
command -v jq >/dev/null || {
  echo "Se requiere jq para copiar el Secret sin escribir credenciales en disco" >&2
  exit 1
}

source_secret="$(kubectl get rabbitmqcluster rabbitmq-cluster \
  -n "$RABBITMQ_NAMESPACE" \
  -o jsonpath='{.status.defaultUser.secretReference.name}')"

kubectl get secret "$source_secret" -n "$RABBITMQ_NAMESPACE" -o json | \
  jq --arg namespace "$NAMESPACE" '
    del(
      .metadata.annotations,
      .metadata.creationTimestamp,
      .metadata.labels,
      .metadata.managedFields,
      .metadata.ownerReferences,
      .metadata.resourceVersion,
      .metadata.uid
    )
    | .metadata.name = "rabbitmq-credentials"
    | .metadata.namespace = $namespace
  ' | kubectl apply -f -
