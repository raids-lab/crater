#!/bin/bash
export KUBECONFIG=${PWD}/kubeconfig
go run main.go \
    --db-config-file ./debug-dbconf.yaml \
    --config-file ./etc/debug-config.yaml \
    --metrics-bind-address :8097 \
    --health-probe-bind-address :8096 \
    --server-port :8099
