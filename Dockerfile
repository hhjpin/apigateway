FROM registry.henghajiang.com/hengha/go_base_image:latest
RUN apk add -U ca-certificates
ADD api_gateway_v2 /
ADD conf.yaml /
ADD conf_test.yaml /
ENTRYPOINT ["./api_gateway_v2"]