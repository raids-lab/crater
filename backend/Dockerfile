# Copyright 2025 RAIDS Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM alpine:3.22

ARG TARGETPLATFORM
ARG BIN_DIR

# Add OpenContainers image metadata labels (https://github.com/opencontainers/image-spec)
LABEL org.opencontainers.image.source="https://github.com/raids-lab/crater-backend"
LABEL org.opencontainers.image.description="Crater Web Backend"
LABEL org.opencontainers.image.licenses="Apache-2.0"

RUN apk add tzdata && ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /

ENV GIN_MODE=release
COPY $BIN_DIR/bin-${TARGETPLATFORM//\//_}/controller .
COPY $BIN_DIR/bin-${TARGETPLATFORM//\//_}/migrate .
RUN chmod +x controller migrate

EXPOSE 8088:8088

# entrypoint will be replaced by the command in k8s deployment, so it's just a placeholder
ENTRYPOINT ["sh", "-c", "echo 'Use k8s deployment to start the service'"]
