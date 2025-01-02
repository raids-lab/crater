package monitor

import (
	"encoding/json"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/raids-lab/crater/pkg/logutils"
)

const (
	queryTimeout = 10 * time.Second
)

//nolint:gocritic // TODO: remove no linter
func PodUtilToJSON(podUtil PodUtil) string {
	jsonBytes, err := json.Marshal(podUtil)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

func JSONToPodUtil(str string) (PodUtil, error) {
	ret := PodUtil{}
	if str == "" {
		return ret, nil
	}
	err := json.Unmarshal([]byte(str), &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

type PrometheusClient struct {
	client api.Client
	v1api  v1.API
}

func NewPrometheusClient(apiURL string) PrometheusInterface {
	client, err := api.NewClient(api.Config{
		Address: apiURL,
	})
	v1api := v1.NewAPI(client)
	if err != nil {
		logutils.Log.Errorf("failed to create Prometheus client: %v", err)
		panic(err)
	}
	return &PrometheusClient{
		client: client,
		v1api:  v1api,
	}
}
