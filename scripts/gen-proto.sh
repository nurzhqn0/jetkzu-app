#!/usr/bin/env bash
set -euo pipefail

# Generate Go protobuf stubs via Docker (no local protoc required).
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)

docker run --rm \
  -v "$ROOT_DIR":/work \
  -w /work \
  golang:1.23-bookworm bash -c '
    set -e
    apt-get update -qq
    apt-get install -qq -y protobuf-compiler >/dev/null
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.2
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
    export PATH=$PATH:/root/go/bin
    rm -rf gen/go
    mkdir -p gen/go
    for svc in user driver ride payment notification; do
      protoc \
        --proto_path=proto \
        --go_out=gen/go \
        --go_opt=paths=source_relative \
        --go-grpc_out=gen/go \
        --go-grpc_opt=paths=source_relative \
        proto/${svc}/v1/${svc}.proto
    done
    chown -R '"$(id -u)":"$(id -g)"' gen
  '
echo "Proto generation completed."
