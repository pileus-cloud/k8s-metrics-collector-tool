/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type PrometheusParams struct {
	url      string
	username string
	password string
	headers  map[string]string
	queryCondition string
	kubeletJobName string
	kubeStateMetricsJobName string
}

type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Job  string `json:"job"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

const (
	KubeletDefaultJobName = "kubelet"
	KubeStateMetricsDefaultJobName = "kube-state-metrics"
)

func queryPrometheus(prometheusParams PrometheusParams, query string) (PrometheusResponse, error) {
	var response PrometheusResponse

	u, _ := url.Parse(prometheusParams.url+"/api/v1/query")
	q := u.Query()
	q.Add("query", query)

	u.RawQuery = q.Encode()

	// Create an HTTP client with Basic Authentication credentials
	client := &http.Client{}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return response, err
	}
	if prometheusParams.username != "" {
		req.SetBasicAuth(prometheusParams.username, prometheusParams.password)
	}
	for key, value := range prometheusParams.headers {
		req.Header.Add(key, value)
	}


	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			fmt.Printf("Make sure filtering condition is correct")
		} 
		return response, fmt.Errorf("unexpected HTTP status code: %v", resp.Status)
	}

	// Read and parse the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}
	
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("error parsing JSON response: %v", err)
	}

	return response, err
}

func askPrometheusParams() (PrometheusParams, error) {
	var prometheusParams PrometheusParams
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Prometheus url: ")

	prometheusParams.url, _ = reader.ReadString('\n')
	prometheusParams.url = strings.TrimSpace(prometheusParams.url)

	if prometheusParams.url == "" {
		return prometheusParams, fmt.Errorf("Prometheus url can't be empty")
	}

	fmt.Printf("Username (leave empty if no authentication is required): ")

	prometheusParams.username, _ = reader.ReadString('\n')
	prometheusParams.username = strings.TrimSpace(prometheusParams.username)

	if prometheusParams.username != "" {
		fmt.Printf("Password: ")

		prometheusParams.password, _ = reader.ReadString('\n')
	}

	fmt.Printf("Additional headers (format: header1:value1,header2:value2, leave empty if no headers required): ")
	headers, _ := reader.ReadString('\n')
	headers = strings.TrimSpace(headers)
	if headers != "" {
		headers_array := strings.Split(headers, ",")
		prometheusParams.headers = make(map[string]string)
		for _, header := range headers_array {
			parts := strings.Split(header, ":")
			if len(parts) != 2 {
				return prometheusParams, fmt.Errorf("Wrong headers format. Use this format: header1:value1,header2:value2")
			}
			prometheusParams.headers[parts[0]] = parts[1]
		}
	}
	
	fmt.Printf("Enter a filtering condition for queries (leave empty if no filtering is required): ")
	prometheusParams.queryCondition, _ = reader.ReadString('\n')
	prometheusParams.queryCondition = strings.TrimSpace(prometheusParams.queryCondition)

	fmt.Printf("Enter the kubelet job name (default `kubelet`): ")
	prometheusParams.kubeletJobName, _ = reader.ReadString('\n')
	prometheusParams.kubeletJobName = strings.TrimSpace(prometheusParams.kubeletJobName)
	if prometheusParams.kubeletJobName == "" {
		prometheusParams.kubeletJobName = KubeletDefaultJobName
	}

	fmt.Printf("Enter the kube-state-metrics job name (default `kube-state-metrics`): ")
	prometheusParams.kubeStateMetricsJobName, _ = reader.ReadString('\n')
	prometheusParams.kubeStateMetricsJobName = strings.TrimSpace(prometheusParams.kubeStateMetricsJobName)
	if prometheusParams.kubeStateMetricsJobName == "" {
		prometheusParams.kubeStateMetricsJobName = KubeStateMetricsDefaultJobName
	}

	return prometheusParams, nil
}

func preCheckPrometheus(prometheusParams PrometheusParams) error {
	expectedMetrics := map[string]string{
		"kube_node_labels": prometheusParams.kubeStateMetricsJobName,
		"kube_node_info": prometheusParams.kubeStateMetricsJobName,
		"kube_node_status_capacity": prometheusParams.kubeStateMetricsJobName,
		"kube_pod_container_resource_requests": prometheusParams.kubeStateMetricsJobName,
		"kube_pod_info": prometheusParams.kubeStateMetricsJobName,
		"kube_pod_container_info": prometheusParams.kubeStateMetricsJobName,
		"kube_pod_container_resource_limits": prometheusParams.kubeStateMetricsJobName,
		"container_cpu_usage_seconds_total": prometheusParams.kubeletJobName,
		"container_memory_usage_bytes": prometheusParams.kubeletJobName,
		"container_network_receive_bytes_total": prometheusParams.kubeletJobName,
		"container_network_transmit_bytes_total": prometheusParams.kubeletJobName,
		"kube_pod_labels": prometheusParams.kubeStateMetricsJobName,
		"kube_pod_created": prometheusParams.kubeStateMetricsJobName,
		"kube_pod_completion_time": prometheusParams.kubeStateMetricsJobName,
		"kube_replicaset_owner": prometheusParams.kubeStateMetricsJobName,
	}

	errors := 0
	missingMetrics := false
	differentJobNames := false
	for metricName, jobName := range expectedMetrics {

		// TODO: which time interval we want??
		response, err := queryPrometheus(prometheusParams, fmt.Sprintf("count by (job) (%s{%s})[2h]", metricName, prometheusParams.queryCondition))
		if err != nil {
			return err
		}

		if len(response.Data.Result) == 0 {
			fmt.Printf("Missing metric %s\n", metricName)
			missingMetrics = true
			errors += 1
			continue
		}
		
		differentJobNames = true
		for _, metric := range response.Data.Result {
			if metric.Metric.Job == jobName {
				differentJobNames = false
				fmt.Printf("Found metric %s, job name: %s\n", metricName, metric.Metric.Job)
				break
			}
		}
		if differentJobNames {
			errors += 1
			fmt.Printf("Can't find metric %s with the specified job name %s\n", metricName, jobName)
			for _, metric := range response.Data.Result {
				fmt.Printf("Found job name: %s\n", metric.Metric.Job)
			}
		}

	}

	fmt.Println("--------------------------------------------")
	if errors == 0 {
		fmt.Println("Validation passed")
	} else {
		fmt.Println("Validation did not pass")
		if missingMetrics {
			fmt.Println(" - Some metrics are missing, check if all required targets are enabled and healthy")
			fmt.Println("   CAdvisor and kube-state-metrics are required")
		}
		if differentJobNames {
			fmt.Println(" - Some metrics have different job names")
			fmt.Println("   Specify correct job names in prompt")
		}
	}
	fmt.Println("--------------------------------------------")

	return nil
}

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if all required metrics are available in prometheus",
	Long:  `Check if all required metrics are available in prometheus`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prometheusParams, err := askPrometheusParams()
		if err != nil {
			return err
		}

		return preCheckPrometheus(prometheusParams)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

}
