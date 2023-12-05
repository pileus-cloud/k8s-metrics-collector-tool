/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"net/http"
	"io"
	"encoding/json"

	"github.com/spf13/cobra"
)

type PrometheusCreds struct {
	url string
	username string
	password string
}

func queryPrometheusGet(prometheusCreds PrometheusCreds, endpoint string) ([]byte, error) {
	var body []byte

	// Create an HTTP client with Basic Authentication credentials
    client := &http.Client{}
    req, err := http.NewRequest("GET", prometheusCreds.url + endpoint, nil)
    if err != nil {
        return body, err
    }
	if prometheusCreds.username != "" {
		req.SetBasicAuth(prometheusCreds.username, prometheusCreds.password)
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


func askPrometheusCredentials() (PrometheusCreds, error) {
	var prometheusCreds PrometheusCreds
	reader := bufio.NewReader(os.Stdin)

		fmt.Printf("Prometheus url: ")

		prometheusCreds.url, _ = reader.ReadString('\n')
		prometheusCreds.url = strings.TrimSpace(prometheusCreds.url)

		if prometheusCreds.url == "" {
			return prometheusCreds, fmt.Errorf("Prometheus url can't be empty")
		}

		fmt.Printf("Username (leave empty if no authentication is required): ")

		prometheusCreds.username, _ = reader.ReadString('\n')
		prometheusCreds.username = strings.TrimSpace(prometheusCreds.username)

		if prometheusCreds.username != "" {
			fmt.Printf("Password: ")

			prometheusCreds.password, _ = reader.ReadString('\n')
		}
	return prometheusCreds, nil
}

func preCheckPrometheus(prometheusCreds PrometheusCreds) error {
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

	body, err := queryPrometheusGet(prometheusCreds, "/api/v1/label/__name__/values")
	if err != nil {
        return err
    }

	var result struct {
        Data []string `json:"data"`
    }

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("error parsing JSON response: %v", err)
	}

    // Check if expected metrics are present in the response
	var missingMetrics []string
    for _, metricName := range expectedMetrics {
        found := false
        for _, name := range result.Data {
            if metricName == name {
                found = true
				fmt.Printf("Found metric %s\n", metricName)
                break
            }
        }
        if !found {
			fmt.Printf("Missing metric %s\n", metricName)
            missingMetrics = append(missingMetrics, metricName)
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
	Long: `Check if all required metrics are available in prometheus`,
	RunE: func(cmd *cobra.Command, args []string) error {
		prometheusCreds, err := askPrometheusCredentials()
		if err != nil {
			return err
		}

		return preCheckPrometheus(prometheusCreds)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

}
