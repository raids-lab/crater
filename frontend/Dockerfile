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

# Use the stable Alpine version of Nginx as the base image
FROM nginx:stable-alpine

# Add OpenContainers image metadata labels (https://github.com/opencontainers/image-spec)
LABEL org.opencontainers.image.source="https://github.com/raids-lab/crater-frontend"
LABEL org.opencontainers.image.description="Crater Web Frontend"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Copy custom nginx configuration
COPY ./hack/deploy/nginx.conf /etc/nginx/nginx.conf

# Copy built dist directory to Nginx web root
COPY ./dist /usr/share/nginx/html

# Expose port 80 for HTTP traffic
EXPOSE 80

# Start Nginx in foreground mode
CMD ["nginx", "-g", "daemon off;"]