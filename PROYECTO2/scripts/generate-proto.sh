#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
PROTO_MODULE="github.com/ipatzantzic19/202308204_LAB_SO1_VacJun2026/PROYECTO2/proto"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

if command -v protoc >/dev/null && \
   command -v protoc-gen-go >/dev/null && \
   command -v protoc-gen-go-grpc >/dev/null; then
  (
    cd "$PROJECT_ROOT/proto"
    protoc \
      --go_out="$tmp_dir" --go_opt="module=$PROTO_MODULE" \
      --go-grpc_out="$tmp_dir" --go-grpc_opt="module=$PROTO_MODULE" \
      prediction.proto
  )
else
  command -v docker >/dev/null || {
    echo "Se requiere protoc con sus plugins de Go, o Docker" >&2
    exit 1
  }
  tool_image="proyecto2-protobuf-tools:local"
  docker build -q -t "$tool_image" \
    -f "$PROJECT_ROOT/scripts/protobuf.Dockerfile" "$PROJECT_ROOT" >/dev/null
  docker run --rm \
    --user "$(id -u):$(id -g)" \
    -v "$PROJECT_ROOT/proto:/workspace:ro" \
    -v "$tmp_dir:/generated" \
    "$tool_image" \
    -I/workspace \
    --go_out=/generated --go_opt="module=$PROTO_MODULE" \
    --go-grpc_out=/generated --go-grpc_opt="module=$PROTO_MODULE" \
    /workspace/prediction.proto
fi

cp "$tmp_dir/prediction.pb.go" "$PROJECT_ROOT/proto/prediction.pb.go"
cp "$tmp_dir/prediction_grpc.pb.go" "$PROJECT_ROOT/proto/prediction_grpc.pb.go"

echo "Módulo Go compartido regenerado desde proto/prediction.proto"
