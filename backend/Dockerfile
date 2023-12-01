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

WORKDIR /

RUN apt update && apt install -y git wget

RUN git config --global url."https://ai-portal-backend-development:***REMOVED***@gitlab.***REMOVED***".insteadof "https://gitlab.***REMOVED***" 

RUN wget https://golang.google.cn/dl/go1.19.13.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.19.13.linux-amd64.tar.gz

RUN git clone https://gitlab.***REMOVED***/act-k8s-portal-system/ai-portal-backend.git

COPY kubeconfig /root/kubeconfig

COPY start.sh /

EXPOSE 8078:8078

RUN chmod +x /start.sh

ENTRYPOINT ["/start.sh"]
