FROM registry.henghajiang.com/hengha/go_base_image:latest
RUN apk add -U ca-certificates
ADD go_tester /
ENTRYPOINT ["./go_tester"]