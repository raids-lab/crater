FROM golang:1.22 as build

WORKDIR /app
ENV GOPROXY=https://proxy.golang.com.cn,direct
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o webdav-server main.go

FROM alpine

COPY --from=build /app/webdav-server /webdav-server
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add tzdata
ENV TZ=Asia/Shanghai

COPY ./etc /etc

EXPOSE 7320

USER root

WORKDIR /

CMD ["/webdav-server"]