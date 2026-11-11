#!/usr/bin/env bash

# Copyright 2026 The Crater Project Team, RAIDS-Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

set -euo pipefail

export PATH="${HOME}/.local/bin:${PATH}"

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly CLUSTER_NAME="crater-agent-dev"
readonly CONTEXT="kind-${CLUSTER_NAME}"
readonly NAMESPACE="crater-system"
readonly MODEL_STUB="crater-agent-smoke-model"

if [[ -z "${CRATER_AGENT_SMOKE_ADMIN_PASSWORD:-}" ]]; then
  printf 'CRATER_AGENT_SMOKE_ADMIN_PASSWORD is required\n' >&2
  exit 1
fi

kubectl --context "$CONTEXT" rollout status deployment/crater-agent \
  --namespace "$NAMESPACE" --timeout=180s
kubectl --context "$CONTEXT" rollout status deployment/crater-backend \
  --namespace "$NAMESPACE" --timeout=180s
kubectl --context "$CONTEXT" rollout status deployment/crater-frontend \
  --namespace "$NAMESPACE" --timeout=180s

cleanup_model_stub() {
  kubectl --context "$CONTEXT" delete pod,service,configmap "$MODEL_STUB" \
    --namespace "$NAMESPACE" --ignore-not-found --wait=false
}
trap cleanup_model_stub EXIT

# Reuse the already loaded Node.js image instead of requiring a real model or API key.
agent_image="$(kubectl --context "$CONTEXT" get deployment crater-agent \
  --namespace "$NAMESPACE" --output=jsonpath='{.spec.template.spec.containers[0].image}')"
kubectl --context "$CONTEXT" create configmap "$MODEL_STUB" \
  --namespace "$NAMESPACE" \
  --from-file="server.mjs=$ROOT_DIR/hack/kind/crater-agent-model-stub.mjs" \
  --dry-run=client --output=yaml | kubectl --context "$CONTEXT" apply -f -
kubectl --context "$CONTEXT" apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $MODEL_STUB
  namespace: $NAMESPACE
  labels:
    app: $MODEL_STUB
spec:
  containers:
    - name: model
      image: $agent_image
      imagePullPolicy: IfNotPresent
      command: [node, /stub/server.mjs]
      ports:
        - containerPort: 8000
      readinessProbe:
        tcpSocket:
          port: 8000
      volumeMounts:
        - name: stub
          mountPath: /stub
          readOnly: true
  volumes:
    - name: stub
      configMap:
        name: $MODEL_STUB
---
apiVersion: v1
kind: Service
metadata:
  name: $MODEL_STUB
  namespace: $NAMESPACE
spec:
  selector:
    app: $MODEL_STUB
  ports:
    - port: 8000
EOF
kubectl --context "$CONTEXT" wait --for=condition=Ready "pod/$MODEL_STUB" \
  --namespace "$NAMESPACE" --timeout=180s

kubectl --context "$CONTEXT" run crater-agent-smoke \
  --namespace "$NAMESPACE" \
  --image=curlimages/curl:8.12.1 \
  --restart=Never \
  --rm --attach \
  --env="ADMIN_PASSWORD=$CRATER_AGENT_SMOKE_ADMIN_PASSWORD" \
  --command -- sh -ec '
    curl --fail --silent --show-error http://crater-agent:8000/healthz
    curl --fail --silent --show-error http://crater-backend-svc/healthz
    status="$(curl --silent --output /dev/null --write-out "%{http_code}" \
      --request POST http://crater-agent:8000/chat)"
    test "$status" = "401"

    login_response="$(curl --fail --silent --show-error \
      --header "Content-Type: application/json" \
      --data "{\"auth\":\"normal\",\"username\":\"crater-admin\",\"password\":\"$ADMIN_PASSWORD\"}" \
      http://crater-backend-svc/api/auth/login)"
    access_token="$(printf "%s" "$login_response" | \
      sed -n "s/.*\"accessToken\":\"\([^\"]*\)\".*/\1/p")"
    test -n "$access_token"

    conversation_id="$(cat /proc/sys/kernel/random/uuid)"
    missing_model_message_id="smoke-no-model-$conversation_id"
    # An empty Kind database must report configuration failure, not a successful chat.
    status="$(curl --silent --show-error --max-time 30 \
      --output /tmp/missing-model.json --write-out "%{http_code}" \
      --header "Authorization: Bearer $access_token" \
      --header "Content-Type: application/json" \
      --data "{\"conversationId\":\"$conversation_id\",\"parentMessageId\":null,\"mode\":\"append\",\"message\":{\"id\":\"$missing_model_message_id\",\"role\":\"user\",\"parts\":[{\"type\":\"text\",\"text\":\"Hello\"}]}}" \
      http://crater-backend-svc/api/v1/agent/chat)"
    test "$status" = "503"
    grep -q "LLM is not configured" /tmp/missing-model.json

    cleanup_model_config() {
      curl --fail --silent --show-error --max-time 10 --request DELETE \
        --header "Authorization: Bearer $access_token" \
        http://crater-backend-svc/api/v1/admin/system-config/llm
    }
    trap cleanup_model_config EXIT
    curl --fail --silent --show-error --max-time 10 --request PUT \
      --header "Authorization: Bearer $access_token" \
      --header "Content-Type: application/json" \
      --data "{\"baseUrl\":\"http://crater-agent-smoke-model:8000/v1\",\"modelName\":\"smoke-model\",\"apiKey\":\"$conversation_id\",\"validate\":false}" \
      http://crater-backend-svc/api/v1/admin/system-config/llm

    # Reuse the failed conversation to verify configuration preflight closed its run.
    message_id="smoke-user-$conversation_id"
    chat_response="$(curl --fail --silent --show-error --max-time 30 \
      --header "Authorization: Bearer $access_token" \
      --header "Content-Type: application/json" \
      --data "{\"conversationId\":\"$conversation_id\",\"parentMessageId\":\"$missing_model_message_id\",\"mode\":\"append\",\"message\":{\"id\":\"$message_id\",\"role\":\"user\",\"parts\":[{\"type\":\"text\",\"text\":\"Introduce Crater\"}]}}" \
      http://crater-backend-svc/api/v1/agent/chat)"
    printf "%s" "$chat_response" | grep -q "Crater-Agent deterministic smoke response"

    conversation_response="$(curl --fail --silent --show-error \
      --header "Authorization: Bearer $access_token" \
      http://crater-backend-svc/api/v1/agent/conversations)"
    printf "%s" "$conversation_response" | grep -q "$conversation_id"

    message_response="$(curl --fail --silent --show-error \
      --header "Authorization: Bearer $access_token" \
      "http://crater-backend-svc/api/v1/agent/conversations/$conversation_id/messages")"
    printf "%s" "$message_response" | grep -q "$message_id"
    printf "%s" "$message_response" | grep -q "assistant"
    printf "%s" "$message_response" | grep -q "Crater-Agent deterministic smoke response"
  '

kubectl --context "$CONTEXT" get pods --namespace "$NAMESPACE"
