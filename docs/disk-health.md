# Disk Health producer

Reads SMART attributes and NVMe data from physical disks on Ceph OSD nodes. Runs as a DaemonSet in the `rook-ceph` namespace.

## Prerequisites

- Rook-Ceph cluster running
- Nodes with physical disks at `/dev/`
- Ceph OSD base path at `/var/lib/rook/rook-ceph/` (Rook default)

The container image ships with `smartmontools` and `nvme-cli` built in.

## Deployment

### Step 1: ServiceAccount and RBAC

```bash
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ceph-disk-health-exporter
  namespace: rook-ceph
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: rook-ceph
  name: ceph-disk-health-exporter-role
rules:
  - apiGroups: [""]
    resources: ["pods", "nodes"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: rook-ceph
  name: ceph-disk-health-exporter-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ceph-disk-health-exporter-role
subjects:
  - kind: ServiceAccount
    name: ceph-disk-health-exporter
    namespace: rook-ceph
EOF
```

### Step 2: ConfigMap

```bash
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: disk-health-config
  namespace: rook-ceph
data:
  PROMETHEUS_ENABLED: "true"
  PROMETHEUS_PORT: "8080"
  DISKS: "/dev/sda,/dev/sdb"
  INTERVAL: "60"
  CEPH_OSD_BASE_PATH: "/var/lib/rook/rook-ceph/"
  GROWN_DEFECTS_THRESHOLD: "10"
  PENDING_SECTORS_THRESHOLD: "3"
  REALLOCATED_SECTORS_THRESHOLD: "10"
  LIFETIME_USED_THRESHOLD: "80"
EOF
```

Set `DISKS` to match your node layout. Use `*` to monitor all disks.

### Step 3: DaemonSet

```bash
kubectl apply -f - <<'EOF'
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: ceph-disk-health-exporter
  namespace: rook-ceph
  labels:
    app: ceph-disk-health-exporter
spec:
  selector:
    matchLabels:
      app: ceph-disk-health-exporter
  template:
    metadata:
      labels:
        app: ceph-disk-health-exporter
    spec:
      serviceAccountName: ceph-disk-health-exporter
      containers:
        - name: disk-health-exporter
          image: ghcr.io/cobaltcore-dev/prysm:0.0.36
          args:
            - local-producer
            - disk-health-metrics
          envFrom:
            - configMapRef:
                name: disk-health-config
          env:
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: INSTANCE_ID
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
          securityContext:
            privileged: true
          volumeMounts:
            - name: host-dev
              mountPath: /dev
            - name: host-proc
              mountPath: /host/proc
              readOnly: true
            - name: host-rook-ceph
              mountPath: /var/lib/rook/rook-ceph
              readOnly: true
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
          ports:
            - containerPort: 8080
              name: metrics
      volumes:
        - name: host-dev
          hostPath:
            path: /dev
            type: Directory
        - name: host-proc
          hostPath:
            path: /proc
            type: Directory
        - name: host-rook-ceph
          hostPath:
            path: /var/lib/rook/rook-ceph
            type: Directory
      tolerations:
        - key: "node-role.kubernetes.io/control-plane"
          effect: "NoSchedule"
        - key: "node-role.kubernetes.io/worker"
          operator: "Exists"
      nodeSelector:
        kubernetes.io/os: linux
EOF
```

`privileged: true` is required -- smartctl needs direct access to host `/dev/` devices.

### Step 4: Service for Prometheus

```bash
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Service
metadata:
  name: disk-health-metrics
  namespace: rook-ceph
  labels:
    app: ceph-disk-health-exporter
spec:
  clusterIP: None
  selector:
    app: ceph-disk-health-exporter
  ports:
    - name: metrics
      port: 8080
      targetPort: 8080
EOF
```

### Step 5: ServiceMonitor (Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: disk-health-metrics
  namespace: rook-ceph
  labels:
    prometheus: kube-prometheus
spec:
  selector:
    matchLabels:
      app: ceph-disk-health-exporter
  endpoints:
    - port: metrics
      interval: 60s
      path: /metrics
```

### Step 6: Verify

```bash
# Check DaemonSet status
kubectl -n rook-ceph get ds ceph-disk-health-exporter

# Check logs
kubectl -n rook-ceph logs -l app=ceph-disk-health-exporter --tail=20

# Test the metrics endpoint
kubectl -n rook-ceph exec -it $(kubectl -n rook-ceph get pod -l app=ceph-disk-health-exporter -o name | head -1) -- wget -qO- http://localhost:8080/metrics | head -30
```

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DISKS` | Comma-separated device list, or `*` for all | `/dev/sda,/dev/sdb` |
| `INTERVAL` | Collection interval in seconds | `10` |
| `PROMETHEUS_ENABLED` | Enable metrics endpoint | `false` |
| `PROMETHEUS_PORT` | HTTP port for metrics | `8080` |
| `NODE_NAME` | Node identifier (use fieldRef) | |
| `INSTANCE_ID` | Instance identifier (use fieldRef) | |
| `CEPH_OSD_BASE_PATH` | Rook-Ceph OSD directory | `/var/lib/rook/rook-ceph/` |
| `GROWN_DEFECTS_THRESHOLD` | Alert threshold: grown defects | `10` |
| `PENDING_SECTORS_THRESHOLD` | Alert threshold: pending sectors | `3` |
| `REALLOCATED_SECTORS_THRESHOLD` | Alert threshold: reallocated sectors | `10` |
| `LIFETIME_USED_THRESHOLD` | Alert threshold: SSD lifetime used (%) | `80` |
| `ALL_ATTR` | Export all SMART attributes | `false` |
| `NATS_URL` | NATS server URL (optional) | |
| `NATS_SUBJECT` | NATS publish subject | `osd.disk.health` |

## OSD mapping

When `CEPH_OSD_BASE_PATH` is set, the producer maps physical devices to Ceph OSD IDs automatically. Every Prometheus metric gets an `osd_id` label.

This works with both direct block devices and LVM logical volumes.

## Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `smart_attributes` | Gauge | SMART attributes (labeled by `attribute`) |
| `disk_temperature_celsius` | Gauge | Disk temperature |
| `disk_reallocated_sectors` | Gauge | Reallocated sector count |
| `disk_pending_sectors` | Gauge | Pending sector count |
| `disk_power_on_hours_total` | Gauge | Cumulative power-on hours |
| `ssd_life_used_percentage` | Gauge | SSD wear level |
| `disk_error_counts_total` | Gauge | Error counts (labeled by `error_type`) |
| `disk_capacity_gb` | Gauge | Disk capacity in GB |
| `disk_info` | Gauge | Device metadata: vendor, model, serial, firmware, media_type |

For NVMe devices, `smart_attributes` includes `critical_warning`, `available_spare`, `available_spare_threshold`, and vendor IDs in hex.

Full list: [metrics reference](../pkg/producers/diskhealthmetrics/README.md).
