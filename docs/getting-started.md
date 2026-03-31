# Getting started

## Requirements

- Kubernetes 1.28+
- Rook-Ceph operator deployed
- Prometheus (scrapes metrics)
- cert-manager (ops-log webhook needs TLS)
- `kubectl` with cluster access
- RabbitMQ (only for ops-log audit trail)

## Container images

Published to GitHub Container Registry on every tagged release:

| Image | What it does |
|-------|-------------|
| `ghcr.io/cobaltcore-dev/prysm:<tag>` | Main binary -- runs all producers |
| `ghcr.io/cobaltcore-dev/prysm-webhook:<tag>` | Ops-log mutating webhook |

Tags follow semver (`0.0.36`), major.minor (`0.0`), and git SHA.

Latest version: [releases](https://github.com/cobaltcore-dev/prysm/releases).

## Producers

Prysm has three producers. Each runs as a separate Kubernetes workload:

| Producer | Command | K8s pattern | External deps |
|----------|---------|-------------|---------------|
| [RadosGW Usage](radosgw-usage.md) | `remote-producer radosgw-usage` | Deployment | RadosGW Admin API |
| [Disk Health](disk-health.md) | `local-producer disk-health-metrics` | DaemonSet | smartctl, nvme-cli (bundled) |
| [Ops Log](ops-log.md) | `local-producer ops-log` | Sidecar (via webhook) | RGW ops-log file; RabbitMQ (optional) |

## Quick start

### 1. RadosGW Usage producer

Pulls bucket/user usage metrics from the RadosGW Admin API.

```bash
kubectl apply -f radosgw-usage-deployment.yaml
```

Full walkthrough: [RadosGW Usage](radosgw-usage.md).

### 2. Disk Health producer

Reads SMART/NVMe attributes on every node running Ceph OSDs.

```bash
kubectl apply -f diskhealthmetrics-serviceaccount.yaml
kubectl apply -f diskhealthmetrics-daemon-set.yaml
```

Full walkthrough: [Disk Health](disk-health.md).

### 3. Ops Log producer

A mutating webhook injects a sidecar into RGW pods. The sidecar parses RGW operation logs and exposes Prometheus metrics. It can also publish CADF audit events to RabbitMQ.

```bash
kubectl apply -f webhook-namespace.yaml
kubectl apply -f webhook-cert.yaml
kubectl apply -f webhook-deployment.yaml
kubectl apply -f webhook-service.yaml
kubectl apply -f webhook-config.yaml
```

Full walkthrough: [Ops Log](ops-log.md).

## Configuration

All producers accept configuration three ways. In Kubernetes, environment variables are the simplest.

### Environment variables (preferred in K8s)

Set via `env`, `envFrom` with a ConfigMap or Secret in your manifests.

### CLI flags

```bash
prysm remote-producer radosgw-usage --admin-url "http://..." --access-key "..." --secret-key "..."
```

### Config file (local multi-producer mode)

```bash
prysm local-producer use-config --config=/path/to/config.yaml
```

See [examples/config/config.yaml](../examples/config/config.yaml) for the format.

## Logging

All producers log structured JSON via zerolog.

Set verbosity with the `-v` flag:

```
-v debug    # verbose
-v info     # standard
-v warn     # default
-v error    # errors only
```

## Prometheus integration

Every producer exposes metrics on an HTTP port (default `8080`; ops-log sidecar uses `9090`).

### ServiceMonitor example

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: prysm-radosgw-usage
  namespace: rook-ceph
  labels:
    prometheus: kube-prometheus
spec:
  selector:
    matchLabels:
      app: radosgw-usage-exporter
  endpoints:
    - port: metrics
      interval: 60s
      path: /metrics
```

### PodMonitor example (sidecar metrics)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: prysm-ops-log
  namespace: rook-ceph
  labels:
    prometheus: kube-prometheus
spec:
  selector:
    matchLabels:
      app: rook-ceph-rgw
  podMetricsEndpoints:
    - port: prysm-metrics
      interval: 60s
      path: /metrics
```

Match `labels`, `namespace`, and `interval` to your Prometheus operator setup.

## Next steps

- [RadosGW Usage producer](radosgw-usage.md) -- deployment walkthrough
- [Disk Health producer](disk-health.md) -- deployment walkthrough
- [Ops Log producer](ops-log.md) -- deployment walkthrough + RabbitMQ audit setup
