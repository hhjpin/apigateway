#!/usr/bin/env bash
rm go_tester
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o go_tester
docker build -t registry.henghajiang.com/hengha/api_gateway_tester:$1 .
docker push registry.henghajiang.com/hengha/api_gateway_tester:$1
