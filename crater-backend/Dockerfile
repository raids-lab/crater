# FROM golang:1.21-alpine AS builder
# WORKDIR /app
# COPY . /app
# RUN go env -w GO111MODULE=on
# RUN go env -w GOPROXY=https://goproxy.cn,direct
# RUN go mod download
# RUN CGO_ENABLED=0 go build -o bin/controller main.go

# FROM alpine AS runner
# WORKDIR /
# COPY --from=builder /app/bin/controller .
# COPY --from=builder /app/dbconf.yaml .
# EXPOSE 8088:8088
# ENTRYPOINT ["/controller --db-config-file /dbconf.yaml --server-port 8078 --metrics-bind-address 8077 --health-probe-bind-address 8076"]

FROM ubuntu:22.04

COPY ./bin/controller /controller
COPY ./etc/config.yaml /etc/config.yaml
RUN chmod +x /controller
EXPOSE 8078:8078
ENTRYPOINT ["/controller --server-port :8078 --config-file /etc/config.yaml"]
