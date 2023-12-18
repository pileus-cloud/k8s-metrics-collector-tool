package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"net/url"

	"github.com/spf13/cobra"
)

const (
	KubeletDefaultJobName          = "kubelet"
	KubeStateMetricsDefaultJobName = "kube-state-metrics"
)

func askPrometheusParams() (PrometheusParams, error) {
	var prometheusParams PrometheusParams
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Prometheus url: ")

	prometheusParams.Url, _ = reader.ReadString('\n')
	prometheusParams.Url = strings.TrimSpace(prometheusParams.Url)

	if prometheusParams.Url == "" {
		return prometheusParams, fmt.Errorf("Prometheus url can't be empty")
	}
	_, err := url.ParseRequestURI(prometheusParams.Url)
	if err != nil {
		return prometheusParams, err
	}


	fmt.Printf("Username (leave empty if no authentication is required): ")

	prometheusParams.Username, _ = reader.ReadString('\n')
	prometheusParams.Username = strings.TrimSpace(prometheusParams.Username)

	if prometheusParams.Username != "" {
		fmt.Printf("Password: ")

		prometheusParams.Password, _ = reader.ReadString('\n')
	}

	fmt.Printf("Additional headers (format: header1:value1,header2:value2, leave empty if no headers required): ")
	headers, _ := reader.ReadString('\n')
	headers = strings.TrimSpace(headers)
	if headers != "" {
		headers_array := strings.Split(headers, ",")
		prometheusParams.Headers = make(map[string]string)
		for _, header := range headers_array {
			parts := strings.Split(header, ":")
			if len(parts) != 2 {
				return prometheusParams, fmt.Errorf("Wrong headers format. Use this format: header1:value1,header2:value2")
			}
			prometheusParams.Headers[parts[0]] = parts[1]
		}
	}

	fmt.Printf("Enter a filtering condition for queries (leave empty if no filtering is required): ")
	prometheusParams.QueryCondition, _ = reader.ReadString('\n')
	prometheusParams.QueryCondition = strings.TrimSpace(prometheusParams.QueryCondition)

	fmt.Printf("Enter the kubelet job name (default `kubelet`): ")
	prometheusParams.KubeletJobName, _ = reader.ReadString('\n')
	prometheusParams.KubeletJobName = strings.TrimSpace(prometheusParams.KubeletJobName)
	if prometheusParams.KubeletJobName == "" {
		prometheusParams.KubeletJobName = KubeletDefaultJobName
	}

	fmt.Printf("Enter the kube-state-metrics job name (default `kube-state-metrics`): ")
	prometheusParams.KubeStateMetricsJobName, _ = reader.ReadString('\n')
	prometheusParams.KubeStateMetricsJobName = strings.TrimSpace(prometheusParams.KubeStateMetricsJobName)
	if prometheusParams.KubeStateMetricsJobName == "" {
		prometheusParams.KubeStateMetricsJobName = KubeStateMetricsDefaultJobName
	}

	return prometheusParams, nil
}

func checkKubeNodeLabels(prometheusParams PrometheusParams) error {
	response, err := prometheusGetLabels(prometheusParams, "kube_node_labels")
	if err != nil {
		return err
	}
	if len(response.Data) == 0 {
		return fmt.Errorf(" - kube_node_labels metric is missing\n   Make sure you have enabled the labels collection")
	}
	labelsCount := 0
	for _, label := range response.Data {
		if strings.HasPrefix(label, "label_") {
			labelsCount += 1
		}
	}
	if labelsCount == 0 {
		return fmt.Errorf(" - kube_node_labels labels must have a `label_` prefix")
	}
	fmt.Printf("Found %d labels in kube_node_labels metric\n", labelsCount)
	return nil
}

func getExpectedMetricsList(prometheusParams PrometheusParams) map[string]string {
	return map[string]string{
		"kube_node_labels":                       prometheusParams.KubeStateMetricsJobName,
		"kube_node_info":                         prometheusParams.KubeStateMetricsJobName,
		"kube_node_status_capacity":              prometheusParams.KubeStateMetricsJobName,
		"kube_pod_container_resource_requests":   prometheusParams.KubeStateMetricsJobName,
		"kube_pod_info":                          prometheusParams.KubeStateMetricsJobName,
		"kube_pod_container_info":                prometheusParams.KubeStateMetricsJobName,
		"kube_pod_container_resource_limits":     prometheusParams.KubeStateMetricsJobName,
		"container_cpu_usage_seconds_total":      prometheusParams.KubeletJobName,
		"container_memory_usage_bytes":           prometheusParams.KubeletJobName,
		"container_network_receive_bytes_total":  prometheusParams.KubeletJobName,
		"container_network_transmit_bytes_total": prometheusParams.KubeletJobName,
		"kube_pod_labels":                        prometheusParams.KubeStateMetricsJobName,
		"kube_pod_created":                       prometheusParams.KubeStateMetricsJobName,
		"kube_pod_completion_time":               prometheusParams.KubeStateMetricsJobName,
		"kube_replicaset_owner":                  prometheusParams.KubeStateMetricsJobName,
	}
}

func checkJobNameError(response PrometheusQueryResponse, jobName string, metricName string) bool {
	for _, metric := range response.Data.Result {
		if metric.Metric.Job == jobName {
			fmt.Printf("Found metric %s, job name: %s\n", metricName, metric.Metric.Job)
			return false
		}
	}
	fmt.Printf("Can't find metric %s with the specified job name %s\n", metricName, jobName)
	for _, metric := range response.Data.Result {
		fmt.Printf("Found job name: %s\n", metric.Metric.Job)
	}
	return true
}

func preCheckPrometheus(prometheusParams PrometheusParams) error {
	expectedMetrics := getExpectedMetricsList(prometheusParams)

	var errors []error
	missingMetricsError := false
	differentJobNamesError := false
	for metricName, jobName := range expectedMetrics {

		response, err := makePrometheusQuery(prometheusParams, fmt.Sprintf("count by (job) (%s{%s})", metricName, prometheusParams.QueryCondition))
		if err != nil {
			return err
		}

		if len(response.Data.Result) == 0 {
			fmt.Printf("Missing metric %s\n", metricName)
			missingMetricsError = true
			continue
		}
		if checkJobNameError(response, jobName, metricName) {
			differentJobNamesError = true
		}

	}
	if missingMetricsError {
		errors = append(errors, fmt.Errorf(" - Some metrics are missing\n   Check if all required targets are enabled and healthy"))
	}
	if differentJobNamesError {
		errors = append(errors, fmt.Errorf(" - Some metrics have different job names\n   Specify correct job names in prompt"))
	}

	kubeNodeLabelsError := checkKubeNodeLabels(prometheusParams)
	if kubeNodeLabelsError != nil {
		errors = append(errors, kubeNodeLabelsError)
	}

	formatResult(errors, prometheusParams)

	return nil
}

func formatResult(errors []error, prometheusParams PrometheusParams) {
	fmt.Println("--------------------------------------------")
	if len(errors) == 0 {
		fmt.Println("Validation passed")
		fmt.Printf("Do you want to generate values file? (Y/n): ")
		var answer string
		fmt.Scanln(&answer)
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "n" || answer == "no" {
			return
		}
		fmt.Printf("Enter a path to save the file to [./values.yaml]: ")
		var path string
		fmt.Scanln(&path)
		if strings.TrimSpace(path) == "" {
			path = "./values.yaml"
		}
		generateValuesFile(prometheusParams, path)
	} else {
		fmt.Println("Validation did not pass")
		for _, err := range errors {
			fmt.Println(err)
		}
	}
	fmt.Println("--------------------------------------------")
}

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
