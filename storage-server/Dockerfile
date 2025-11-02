FROM alpine

ARG BIN_DIR
ARG TARGETPLATFORM

WORKDIR /

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add tzdata
ENV TZ=Asia/Shanghai

COPY $BIN_DIR/bin-${TARGETPLATFORM//\//_}/storage-server .

COPY ./etc /etc 

RUN chmod +x storage-server

EXPOSE 7320 

USER root 

CMD ["/storage-server"]
