#!/bin/bash

mkdir -p gourgeist/ghastly/proto

protoc --go_out=gourgeist/ghastly/proto --go_opt=paths=source_relative \
  --go-grpc_out=gourgeist/ghastly/proto --go-grpc_opt=paths=source_relative \
  ghastly.proto

mkdir -p trevenant/ghastly/proto

python3 -m grpc_tools.protoc -I. \
  --python_out=trevenant/ghastly/proto --grpc_python_out=trevenant/ghastly/proto ghastly.proto