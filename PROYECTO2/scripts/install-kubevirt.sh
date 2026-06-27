#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

CLUSTER_NAME="${CLUSTER_NAME:-sopes1-p2-cluster}"
CLUSTER_ZONE="${CLUSTER_ZONE:-us-central1-a}"
KUBEVIRT_NODE_POOL="${KUBEVIRT_NODE_POOL:-kubevirt-pool}"
KUBEVIRT_NODES="${KUBEVIRT_NODES:-1}"

if ! gcloud container node-pools describe "$KUBEVIRT_NODE_POOL" \
  --cluster "$CLUSTER_NAME" --zone "$CLUSTER_ZONE" >/dev/null 2>&1; then
  echo "==> creando node pool N1 con virtualización anidada"
  gcloud container node-pools create "$KUBEVIRT_NODE_POOL" \
    --cluster "$CLUSTER_NAME" \
    --zone "$CLUSTER_ZONE" \
    --machine-type n1-standard-4 \
    --image-type COS_CONTAINERD \
    --disk-type pd-balanced \
    --disk-size 50 \
    --num-nodes "$KUBEVIRT_NODES" \
    --enable-nested-virtualization \
    --node-labels nested-virtualization=enabled \
    --node-taints kubevirt.io/dedicated=virtualization:NoSchedule
fi

echo "==> instalando KubeVirt $KUBEVIRT_VERSION"
kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml"
# El manifiesto upstream exige labels worker/control-plane que GKE no asigna a sus nodos.
kubectl patch deployment virt-operator -n kubevirt --type=json \
  -p='[{"op":"remove","path":"/spec/template/spec/affinity"}]'
kubectl rollout status deployment/virt-operator -n kubevirt --timeout=300s
kubectl apply -f "$PROJECT_ROOT/infra/kubernetes/kubevirt/kubevirt-cr.yaml"
kubectl wait kubevirt/kubevirt -n kubevirt --for=condition=Available --timeout=600s

mkdir -p "$PROJECT_ROOT/.bin"
if [[ ! -x "$PROJECT_ROOT/.bin/virtctl" ]]; then
  curl -fsSL -o "$PROJECT_ROOT/.bin/virtctl" \
    "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/virtctl-${KUBEVIRT_VERSION}-linux-amd64"
  chmod +x "$PROJECT_ROOT/.bin/virtctl"
fi
