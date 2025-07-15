# Grafana Dashboards

## Usage

### 1. Export Environment Variables

Set the required environment variables for your Grafana instance:

```shell
export GRAFANA_API="http://your-grafana-url:port"
export API_KEY="your-grafana-api-key"
```

### 2. Run the Import Script

Execute the `quick-import.sh` script to import all JSON files in the `dashboards/` directory:

```shell
chmod +x quick-import.sh

./quick-import.sh
```

