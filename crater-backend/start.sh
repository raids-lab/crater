#!/bin/bash

set -e

export KUBECONFIG="/root/kubeconfig"
git config --global url."https://ai-portal-backend-development:***REMOVED***@gitlab.***REMOVED***".insteadof "https://gitlab.***REMOVED***" 
cd /ai-portal-backend
git checkout main
git pull
/usr/local/go/bin/go build -o bin/controller main.go
bin/controller --server-port :8078 --config-file ./etc/debug-config.yaml
