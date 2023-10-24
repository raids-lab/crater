FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . /app
RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o app

FROM alpine AS runner
WORKDIR /app
COPY --from=builder /app/app .
COPY --from=builder /app/test.env .
EXPOSE 8080:8080
ENTRYPOINT ["./app"]
