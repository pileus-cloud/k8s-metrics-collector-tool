# k8s-metrics-collector-tool

## Install

Download an executable binary from the latest release (https://github.com/pileus-cloud/k8s-metrics-collector-tool/releases)[https://github.com/pileus-cloud/k8s-metrics-collector-tool/releases]

## Usage

### Checking prometheus 

```
./k8s-metrics-collector-tool check
```

### Install kube-prometheus-stack

Installs kube-prometheus-stack helm chart with default configuration to collect all required metrics. The chart includes prometheus, CAdvisor and kube-state-metrics. For more detiled info see (https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)[https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack]


This command requires `helm3` installed. (https://helm.sh/docs/intro/install/)[https://helm.sh/docs/intro/install/]

```
./k8s-metrics-collector-tool installprom --namespace <k8s-namespace>
```