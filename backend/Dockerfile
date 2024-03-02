FROM alpine AS runner
WORKDIR /
COPY ./bin/controller .
COPY ./dbconf.yaml .

EXPOSE 8088:8088

# entrypoint will be replaced by the command in k8s deployment
ENTRYPOINT ["/controller --db-config-file /dbconf.yaml --server-port 8088"]
