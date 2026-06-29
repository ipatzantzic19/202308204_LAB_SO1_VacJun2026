#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

ORAS_VERSION="${ORAS_VERSION:-1.3.2}"
ORAS_BIN="$PROJECT_ROOT/.bin/oras"
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_ROOT/.artifacts/prediction-proto}"
SOURCE_FILE="$PROJECT_ROOT/proto/prediction.proto"

install_oras() {
  if [[ -x "$ORAS_BIN" ]]; then
    return
  fi

  local arch archive base tmp expected
  case "$(uname -m)" in
    x86_64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo "Arquitectura no soportada por este instalador de ORAS: $(uname -m)" >&2; exit 1 ;;
  esac
  archive="oras_${ORAS_VERSION}_linux_${arch}.tar.gz"
  base="https://github.com/oras-project/oras/releases/download/v${ORAS_VERSION}"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN
  curl --fail --location --silent --show-error "$base/$archive" -o "$tmp/$archive"
  curl --fail --location --silent --show-error "$base/oras_${ORAS_VERSION}_checksums.txt" -o "$tmp/checksums.txt"
  expected="$(awk -v file="$archive" '$2 == file {print $1}' "$tmp/checksums.txt")"
  [[ -n "$expected" ]] || { echo "No se encontró checksum para $archive" >&2; exit 1; }
  printf '%s  %s\n' "$expected" "$tmp/$archive" | sha256sum --check --status
  mkdir -p "$(dirname "$ORAS_BIN")"
  tar -xzf "$tmp/$archive" -C "$tmp" oras
  install -m 0755 "$tmp/oras" "$ORAS_BIN"
  rm -rf "$tmp"
  trap - RETURN
}

pull_artifact() {
  local destination="$1"
  rm -rf "$destination"
  mkdir -p "$destination"
  "$ORAS_BIN" pull "$OCI_ARTIFACT" --output "$destination"
}

verify_artifact() {
  local tmp
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN
  pull_artifact "$tmp"
  cmp "$SOURCE_FILE" "$tmp/prediction.proto"
  echo "Artefacto OCI verificado: $OCI_ARTIFACT"
  rm -rf "$tmp"
  trap - RETURN
}

install_oras
case "${1:-verify}" in
  publish)
    (
      cd "$PROJECT_ROOT/proto"
      "$ORAS_BIN" push "$OCI_ARTIFACT" \
        --artifact-type application/vnd.sopes1.prediction.proto.v1 \
        "prediction.proto:application/vnd.google.protobuf"
    )
    verify_artifact
    ;;
  pull)
    pull_artifact "$OUTPUT_DIR"
    echo "prediction.proto descargado en $OUTPUT_DIR"
    ;;
  verify)
    verify_artifact
    ;;
  *)
    echo "Uso: $0 {publish|pull|verify}" >&2
    exit 2
    ;;
esac
