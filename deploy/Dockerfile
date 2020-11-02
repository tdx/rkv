FROM golang:1.15-alpine AS build
WORKDIR /go/src/rkv
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/rkvd ./cmd/rkvd
RUN GRPC_HEALTH_PROBE_VERSION=v0.3.2 && \
    wget -qO/go/bin/grpc_health_probe \
    https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/\
${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /go/bin/grpc_health_probe

FROM nicolaka/netshoot
COPY --from=build /go/bin/rkvd /bin/rkvd
COPY --from=build /go/bin/grpc_health_probe /bin/grpc_health_probe
COPY example/cluster/put /bin/put
COPY example/cluster/get /bin/get
COPY example/cluster/del /bin/del
COPY example/cluster/join /bin/join
ENTRYPOINT ["/bin/rkvd"]
