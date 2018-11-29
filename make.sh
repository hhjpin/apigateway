#!/usr/bin/env bash
rm api_gateway_v2
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api_gateway_v2
docker build -t registry.henghajiang.com/hengha/api_gateway_v2:$1 .
docker push registry.henghajiang.com/hengha/api_gateway_v2:$1
