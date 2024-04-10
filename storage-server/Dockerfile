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


EXPOSE 7320
 
 WORKDIR /

CMD ["/webdav-server"]