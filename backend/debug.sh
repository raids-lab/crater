#!/bin/bash
export KUBECONFIG=${PWD}/kubeconfig
go run main.go \
    --config-file ./etc/debug-config.yaml \
    --server-port :8099
