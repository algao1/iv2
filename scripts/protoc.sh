#!/bin/bash

mkdir -p gourgeist/pkg/ghastly/proto

protoc --go_out=gourgeist/pkg/ghastly/proto --go_opt=paths=source_relative \
  --go-grpc_out=gourgeist/pkg/ghastly/proto --go-grpc_opt=paths=source_relative \
  ghastly.proto

mkdir -p trevenant/ghastly/proto

python3 -m grpc_tools.protoc -I. \
  --python_out=trevenant/ghastly/proto --grpc_python_out=trevenant/ghastly/proto ghastly.proto

# Replace import with relative import.
sed -i 's/^import .*_pb2 as/from . \0/' trevenant/ghastly/proto/ghastly_pb2_grpc.py