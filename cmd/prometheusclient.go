package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type PrometheusParams struct {
	Url                     string
	Username                string
	Password                string
	Headers                 map[string]string
	QueryCondition          string
	KubeletJobName          string
	KubeStateMetricsJobName string
}

type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Job string `json:"job"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type PrometheusLabelsResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

func makePrometheusRequestGet(prometheusParams PrometheusParams, url string) (*http.Response, error) {
	var resp *http.Response

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return resp, err
	}
	if prometheusParams.Username != "" {
		req.SetBasicAuth(prometheusParams.Username, prometheusParams.Password)
	}
	for key, value := range prometheusParams.Headers {
		req.Header.Add(key, value)
	}

	resp, err = client.Do(req)
	if err != nil {
		return resp, err
	}

	return resp, err
}

func makePrometheusQuery(prometheusParams PrometheusParams, query string) (PrometheusQueryResponse, error) {
	var response PrometheusQueryResponse

	u, _ := url.Parse(prometheusParams.Url + "/api/v1/query")
	q := u.Query()
	q.Add("query", query)

	u.RawQuery = q.Encode()

	resp, err := makePrometheusRequestGet(prometheusParams, u.String())
	if err != nil {
		return response, err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			fmt.Println("Make sure filtering condition is correct")
		}
		return response, fmt.Errorf("Unexpected HTTP status code: %v", resp.Status)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("error parsing JSON response: %v", err)
	}

	return response, err
}

func prometheusGetLabels(prometheusParams PrometheusParams, metric string) (PrometheusLabelsResponse, error) {
	var response PrometheusLabelsResponse

	u, _ := url.Parse(prometheusParams.Url + "/api/v1/labels")
	q := u.Query()
	q.Add("match[]", metric)

	u.RawQuery = q.Encode()

	resp, err := makePrometheusRequestGet(prometheusParams, u.String())
	if err != nil {
		return response, err
	}

	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("unexpected HTTP status code: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("error parsing JSON response: %v", err)
	}

	return response, err
}
