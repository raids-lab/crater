#!/bin/bash
# 运行
go run main.go --db-config-file ./debug-dbconf.yaml --server-port :8099 --metrics-bind-address :8097 --health-probe-bind-address :8096 --config-file etc/debug-config.yaml