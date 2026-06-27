#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

action="${1:-build}"
if [[ "$action" != "build" && "$action" != "push" && "$action" != "all" ]]; then
  echo "Uso: $0 {build|push|all}" >&2
  exit 2
fi

build_images() {
  docker build -t "$RUST_IMAGE" "$PROJECT_ROOT/rust-api"
  docker build -t "$GO_D1_REST_IMAGE" "$PROJECT_ROOT/go-d1/rest-server"
  docker build -t "$GO_D1_GRPC_IMAGE" \
    -f "$PROJECT_ROOT/go-d1/grpc-client/Dockerfile" "$PROJECT_ROOT"
  docker build -t "$GO_D2_GRPC_IMAGE" \
    -f "$PROJECT_ROOT/go-d2/grpc-server/Dockerfile" "$PROJECT_ROOT"
  docker build -t "$GO_D2_WRITER_IMAGE" "$PROJECT_ROOT/go-d2/rabbit-writer"
  docker build -t "$LOCUST_IMAGE" "$PROJECT_ROOT/locust"
  docker pull "$RABBITMQ_SOURCE_IMAGE"
  docker tag "$RABBITMQ_SOURCE_IMAGE" "$RABBITMQ_IMAGE"
}

push_images() {
  for image in \
    "$RUST_IMAGE" \
    "$GO_D1_REST_IMAGE" \
    "$GO_D1_GRPC_IMAGE" \
    "$GO_D2_GRPC_IMAGE" \
    "$GO_D2_WRITER_IMAGE" \
    "$LOCUST_IMAGE" \
    "$RABBITMQ_IMAGE"; do
    docker push "$image"
  done
}

if [[ "$action" == "build" || "$action" == "all" ]]; then
  build_images
fi
if [[ "$action" == "push" || "$action" == "all" ]]; then
  push_images
fi
