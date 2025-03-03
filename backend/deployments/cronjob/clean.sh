#!/bin/bash

# 检查必须的环境变量是否设置
if [ -z "$HOST" ] || [ -z "$USERNAME" ] || [ -z "$PASSWORD" ]; then
  echo "必须设置 HOST、USERNAME 和 PASSWORD 环境变量"
  exit 1
fi

# 登录，获取返回 JSON 数据
response=$(curl -s -X POST \
  "https://${HOST}/api/auth/login" \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d "{
  \"auth\": \"normal\",
  \"password\": \"${PASSWORD}\",
  \"username\": \"${USERNAME}\"
}")

# 利用 jq 提取 accessToken
accessToken=$(echo "$response" | jq -r '.data.accessToken')

# 清理长时间任务
echo "清理长时间任务..."
curl -X DELETE \
  "https://${HOST}/api/v1/admin/operations/cleanup?batchDays=${BATCH_DAYS}&interactiveDays=${INTERACTIVE_DAYS}" \
  -H 'accept: application/json' \
  -H "Authorization: Bearer ${accessToken}"

# 清理GPU低利用率任务
echo "清理GPU低利用率任务..."
curl -X DELETE \
  "https://${HOST}/api/v1/admin/operations/auto?timeRange=${TIME_RANGE}&waitTime=${WAIT_TIME}&util=${UTIL}" \
  -H 'accept: application/json' \
  -H "Authorization: Bearer ${accessToken}"
