#!/usr/bin/env bash
function check_fail() { if [ $? != 0 ]; then echo -e "\nFail！\n"; exit 1; fi }
rm api_gateway_v2
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api_gateway_v2; check_fail
docker build -t registry.henghajiang.com/hengha/api_gateway_v2:$1 .; check_fail
docker push registry.henghajiang.com/hengha/api_gateway_v2:$1; check_fail
rm api_gateway_v2
echo "Success";