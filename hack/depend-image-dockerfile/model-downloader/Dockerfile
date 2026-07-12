# Copyright 2026 The Crater Project Team, RAIDS-Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM python:3.11-slim

ARG HUGGINGFACE_HUB_VERSION=1.23.0
ARG MODELSCOPE_VERSION=1.38.1
ARG MODELSCOPE_HUB_VERSION=0.1.7

LABEL org.opencontainers.image.source="https://github.com/raids-lab/crater"
LABEL org.opencontainers.image.description="Reproducible model and dataset downloader for Crater"
LABEL org.opencontainers.image.licenses="Apache-2.0"

RUN pip install --no-cache-dir \
      "huggingface_hub==${HUGGINGFACE_HUB_VERSION}" \
      "modelscope==${MODELSCOPE_VERSION}" \
      "modelscope-hub==${MODELSCOPE_HUB_VERSION}" \
    && python -c "import huggingface_hub, modelscope, modelscope_hub; print(huggingface_hub.__version__, modelscope.__version__, modelscope_hub.__version__)"

WORKDIR /data

ENTRYPOINT ["/bin/bash", "-c"]
