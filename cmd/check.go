
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	KubeletDefaultJobName = "kubelet"
	KubeStateMetricsDefaultJobName = "kube-state-metrics"
)

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

func checkKubeNodeLabels(prometheusParams PrometheusParams) error {
	response, err := prometheusGetLabels(prometheusParams, "kube_node_labels")
	if err != nil {
		return err
	}
	if len(response.Data) == 0 {
		return fmt.Errorf(" - kube_node_labels metric is missing\n   make sure you have enabled the labels collection")
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

		// TODO: which time interval we want??
		response, err := makePrometheusQuery(prometheusParams, fmt.Sprintf("count by (job) (%s{%s})", metricName, prometheusParams.queryCondition))
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
		errors = append(errors, fmt.Errorf(` - Some metrics are missing, check if all required targets are enabled and healthy
		   CAdvisor and kube-state-metrics are required`))
	}
	if differentJobNamesError {
		errors = append(errors, fmt.Errorf(" - Some metrics have different job names\n   Specify correct job names in prompt"))
	}

	kubeNodeLabelsError := checkKubeNodeLabels(prometheusParams)
	if kubeNodeLabelsError != nil {
		errors = append(errors, kubeNodeLabelsError)
	}

	formatResult(errors)

	return nil
}

func formatResult(errors []error) {
	fmt.Println("--------------------------------------------")
	if len(errors) == 0 {
		fmt.Println("Validation passed")
	} else {
		fmt.Println("Validation did not pass")
		for _, err := range errors {
			fmt.Println(err)
		}
	}
	fmt.Println("--------------------------------------------")
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
