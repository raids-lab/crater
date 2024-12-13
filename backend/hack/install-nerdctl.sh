#!/bin/bash
set -e

# Usage: bash hack/install-nerdctl.sh
NERDCTL_VERSION=2.0.1
REGISTERY=***REMOVED***

wget https://github.com/containerd/nerdctl/releases/download/v2.0.1/nerdctl-2.0.1-linux-amd64.tar.gz -O bin/nerdctl-2.0.1-linux-amd64.tar.gz

tar -zxf bin/nerdctl-2.0.1-linux-amd64.tar.gz -C bin

docker build -t $REGISTERY/nerdctl:$NERDCTL_VERSION -f deployments/nerdctl/Dockerfile.nerdctl .

# rm bin/nerdctl-2.0.1-linux-amd64.tar.gz
# rm bin/nerdctl