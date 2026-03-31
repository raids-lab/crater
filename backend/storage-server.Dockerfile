FROM alpine:3.22

ARG BIN_DIR
ARG TARGETPLATFORM

# Add OpenContainers image metadata labels (https://github.com/opencontainers/image-spec)
LABEL org.opencontainers.image.source="https://github.com/raids-lab/crater"
LABEL org.opencontainers.image.description="Crater Storage Server"
LABEL org.opencontainers.image.licenses="Apache-2.0"

WORKDIR /

RUN apk add tzdata && ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

ENV GIN_MODE=release
COPY $BIN_DIR/bin-${TARGETPLATFORM//\//_}/storage-server .
RUN chmod +x storage-server

EXPOSE 7320

USER root

CMD ["/storage-server"]
