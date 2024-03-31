package monitor

import (
	"fmt"
	"log"
)

func (p *PrometheusClient) QueryNodeCPUUsageRatio() map[string]float32 {
	apiURL := PrometheusAPI
	client := NewPrometheusClient(apiURL)
	query := `100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`
	data, err := client.Float32MapQuery(query, "instance")
	// print result
	// fmt.Println(data)
	if err != nil {
		log.Fatalf("queryMetric error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryNodeMemoryUsageRatio() map[string]float32 {
	apiURL := PrometheusAPI
	client := NewPrometheusClient(apiURL)
	query := "(1 - (avg by (instance) (node_memory_MemAvailable_bytes) / avg by (instance) (node_memory_MemTotal_bytes))) * 100"
	data, err := client.Float32MapQuery(query, "instance")
	// print result
	// fmt.Println(data)
	if err != nil {
		log.Fatalf("queryMetric error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryNodeAllocatedMemory() map[string]int {
	apiURL := PrometheusAPI
	client := NewPrometheusClient(apiURL)
	query := "node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes"
	data, err := client.IntMapQuery(query, "instance")
	// print result
	// fmt.Println(data)
	if err != nil {
		log.Fatalf("queryMetric error: %v", err)
		return nil
	}
	return data
}

func (p *PrometheusClient) QueryPodCPURatio(podName string) float32 {
	fmt.Printf("Querying CPU usage for pod %q\n", podName)
	apiURL := PrometheusAPI
	client := NewPrometheusClient(apiURL)
	query := fmt.Sprintf("sum(rate(container_cpu_usage_seconds_total{pod=%q}[5m]))", podName)
	data, err := client.Float32MapQuery(query, "")
	if err != nil {
		log.Fatalf("queryMetric error: %v", err)
		return 0.0
	}
	var sum float32
	for _, v := range data {
		sum += v
	}
	return sum
}

func (p *PrometheusClient) QueryPodMemory(podName string) int {
	fmt.Printf("Querying CPU usage for pod %q\n", podName)
	apiURL := PrometheusAPI
	client := NewPrometheusClient(apiURL)
	query := fmt.Sprintf("container_memory_usage_bytes{pod=%q}", podName)
	data, err := client.IntMapQuery(query, "")
	if err != nil {
		log.Fatalf("queryMetric error: %v", err)
		return 0
	}
	var sum int
	for _, v := range data {
		sum += v
	}
	return sum
}
