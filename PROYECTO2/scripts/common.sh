#!/usr/bin/env bash

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REGISTRY="${REGISTRY:-zot.35-226-224-23.sslip.io}"
NAMESPACE="${NAMESPACE:-sopes1-p2}"
RABBITMQ_NAMESPACE="${RABBITMQ_NAMESPACE:-rabbitmq-system}"
RABBITMQ_OPERATOR_VERSION="${RABBITMQ_OPERATOR_VERSION:-2.21.1}"

RUST_IMAGE="$REGISTRY/sopes1/rust-api:v3"
GO_D1_REST_IMAGE="$REGISTRY/sopes1/go-d1-rest:v3"
GO_D1_GRPC_IMAGE="$REGISTRY/sopes1/go-d1-grpc-client:v3"
GO_D2_GRPC_IMAGE="$REGISTRY/sopes1/go-d2-grpc-server:v2"
GO_D2_WRITER_IMAGE="$REGISTRY/sopes1/go-d2-rabbit-writer:v4"
LOCUST_IMAGE="$REGISTRY/sopes1/locust:v1"
RABBITMQ_SOURCE_IMAGE="rabbitmq:3.13-management"
RABBITMQ_IMAGE="$REGISTRY/library/rabbitmq:3.13-management"
