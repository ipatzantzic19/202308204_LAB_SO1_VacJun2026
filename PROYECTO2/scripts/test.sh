#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

for module in \
  proto \
  go-d1/rest-server \
  go-d1/grpc-client \
  go-d2/grpc-server \
  go-d2/rabbit-writer \
  go-consumer; do
  echo "==> go test $module"
  (cd "$PROJECT_ROOT/$module" && go test ./...)
done

echo "==> cargo test rust-api"
(cd "$PROJECT_ROOT/rust-api" && cargo test)

echo "==> validar locustfile.py"
python3 -c 'import pathlib, sys; path = pathlib.Path(sys.argv[1]); compile(path.read_text(), str(path), "exec")' \
  "$PROJECT_ROOT/locust/locustfile.py"
