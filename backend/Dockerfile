FROM ***REMOVED***/crater/alpine:240304 AS backend

WORKDIR /

ENV GIN_MODE=release
COPY ./bin/controller .
COPY ./bin/migrate .

EXPOSE 8088:8088

# entrypoint will be replaced by the command in k8s deployment, so it's just a placeholder
ENTRYPOINT ["sh", "-c", "echo 'Use k8s deployment to start the service'"]
