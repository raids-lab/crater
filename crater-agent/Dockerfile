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

FROM python:3.11-slim

LABEL org.opencontainers.image.source="https://github.com/raids-lab/crater-agent"
LABEL org.opencontainers.image.description="Crater AI operations agent service"
LABEL org.opencontainers.image.licenses="Apache-2.0"

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV CRATER_AGENT_HOST=0.0.0.0
ENV CRATER_AGENT_PORT=8000

WORKDIR /app

COPY pyproject.toml README.md ./
COPY config ./config
COPY crater_agent ./crater_agent

RUN python -m pip install --no-cache-dir --upgrade pip \
    && python -m pip install --no-cache-dir .

EXPOSE 8000

CMD ["sh", "-c", "python -m uvicorn crater_agent.app:app --host ${CRATER_AGENT_HOST:-0.0.0.0} --port ${CRATER_AGENT_PORT:-8000}"]
