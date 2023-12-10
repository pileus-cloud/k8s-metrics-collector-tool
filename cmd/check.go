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
}

func queryPrometheus(prometheusParams PrometheusParams, query string) ([]byte, error) {
	var body []byte

	u, _ := url.Parse(prometheusParams.url+"/api/v1/query")
	q := u.Query()
	q.Add("query", query)

	u.RawQuery = q.Encode()

	// Create an HTTP client with Basic Authentication credentials
	client := &http.Client{}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return body, err
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
		return body, err
	}
	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("unexpected HTTP status code: %v", resp.Status)
	}

	// Read and parse the response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}
	return body, err
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
	
	prometheusParams.queryCondition = ""
	return prometheusParams, nil
}

func preCheckPrometheus(prometheusParams PrometheusParams) error {
	expectedMetrics := []string{
		"kube_node_labels",
		"kube_node_info",
		"kube_node_status_capacity",
		"kube_pod_container_resource_requests",
		"kube_pod_info",
		"kube_pod_container_info",
		"kube_pod_container_resource_limits",
		"container_cpu_usage_seconds_total",
		"container_memory_usage_bytes",
		"container_network_receive_bytes_total",
		"container_network_transmit_bytes_total",
		"kube_pod_labels",
		"kube_pod_created",
		"kube_pod_completion_time",
		"kube_replicaset_owner",
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

	// Check if expected metrics are present in the response
	var missingMetrics []string
	for _, metricName := range expectedMetrics {

		body, err := queryPrometheus(prometheusParams, fmt.Sprintf("count by (job) (%s{%s})", metricName, prometheusParams.queryCondition))
		if err != nil {
			return err
		}
		var response PrometheusResponse
	
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("error parsing JSON response: %v", err)
		}
		if len(response.Data.Result) == 0 {
			fmt.Printf("Missing metric %s\n", metricName)
			missingMetrics = append(missingMetrics, metricName)
			continue
		}
		if len(response.Data.Result) == 1 {
			fmt.Printf("Found metric %s, job name: %s\n", metricName, response.Data.Result[0].Metric.Job)
			continue
		}
		fmt.Printf("Found metric %s with multiple job names\n", metricName)
		for _, metric := range response.Data.Result {
			fmt.Printf("Job name: %s\n", metric.Metric.Job)
		}

	}

	if len(missingMetrics) == 0 {
		fmt.Println("Validation passed")
	} else {
		fmt.Println("--------------------------------------------")
		fmt.Println("Some metrics are missing, check if all required targets are enabled and healthy")
		fmt.Println("CAdvisor and kube-state-metrics are required")
		fmt.Println("--------------------------------------------")
	}

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
