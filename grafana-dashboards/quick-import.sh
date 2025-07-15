# GRAFANA_API="http://localhost:3000/api"
# API_KEY="glsa_xxxxxxxx"

# check Grafana API health
health=$(curl -s -i -H "Authorization: Bearer $API_KEY" "$GRAFANA_API/api/health")
health_code=$(echo "$health" | grep HTTP | awk '{print $2}')
if [ "$health_code" -eq 200 ]; then
  echo "Grafana API is healthy."
else
  echo "Grafana API is not healthy. Status code: $health_code"
  exit 1
fi

for json_file in dashboards/*.json; do
  echo "Processing $json_file..."
  
  # Create a temporary file
  temp_file=$(mktemp)
  
  # core logic:
  # 1. Remove the original dashboard's id and uid (set to null)
  # 2. Wrap it in the format required by Grafana (top-level contains dashboard

  jq '
    .id = null | 
    .uid = null | 
    {
      "dashboard": ., 
      "folderId": 0,  # 导入到根目录
      "overwrite": false  # 不覆盖现有仪表盘（若需覆盖设为 true）
    }
  ' "$json_file" > "$temp_file"
  
  # send the dashboard to Grafana API
  response=$(curl -s -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $API_KEY" \
    --data-binary "@$temp_file" \
    "$GRAFANA_API/api/dashboards/db")
  
  status_code=${response: -3}
  body=${response%???}
  
  if [ "$status_code" -eq 200 ]; then
    echo "successfully imported: $json_file"
    echo "Response: $body"
  else
    echo "Error: $status_code"
    echo "Details: $body"
  fi
  
  rm "$temp_file"
done