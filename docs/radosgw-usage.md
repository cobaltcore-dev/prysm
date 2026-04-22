# RadosGW Usage producer

Collects bucket, user, and quota metrics from the RadosGW Admin API and exposes them for Prometheus. Internally, it runs an embedded NATS JetStream instance for state management. No external NATS server required.

## Prerequisites

- RadosGW with admin API enabled
- A CephObjectStoreUser with admin capabilities: `usage=read`, `buckets=read`, `users=read`
- Network path from the producer pod to the RadosGW admin endpoint

## Deployment

### Step 1: Create a CephObjectStoreUser

The producer needs RadosGW admin credentials. When you create a CephObjectStoreUser, Rook generates a Secret with `AccessKey` and `SecretKey` automatically.

```bash
kubectl apply -f - <<'EOF'
apiVersion: ceph.rook.io/v1
kind: CephObjectStoreUser
metadata:
  name: rgw-admin-user
  namespace: rook-ceph
spec:
  store: my-store
  displayName: "RGW Admin for Prysm"
  capabilities:
    user: "read"
    bucket: "read"
    usage: "read"
EOF
```

The secret name follows this pattern:

```
rook-ceph-object-user-<store>-<user>
```

Example: `rook-ceph-object-user-my-store-rgw-admin-user`

Verify it exists:

```bash
kubectl -n rook-ceph get secret rook-ceph-object-user-my-store-rgw-admin-user -o jsonpath='{.data.AccessKey}' | base64 -d
```

### Step 2: Deploy the producer

The deployment pulls credentials directly from the Rook-created secret:

```bash
kubectl apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-store-ext-rgw-exporter
  namespace: rook-ceph
  labels:
    app: my-store-ext-rgw-exporter
  annotations:
    secret.reloader.stakater.com/reload: "rook-ceph-object-user-my-store-rgw-admin-user"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-store-ext-rgw-exporter
  template:
    metadata:
      labels:
        app: my-store-ext-rgw-exporter
    spec:
      containers:
        - name: prysm
          image: ghcr.io/cobaltcore-dev/prysm:0.0.36
          args:
            - remote-producer
            - radosgw-usage
            - --rgw-cluster-id=my-store
            - --prometheus=true
            - -v=info
          env:
            - name: ADMIN_URL
              value: "http://rook-ceph-rgw-my-store.rook-ceph.svc:8080"
            - name: INTERVAL
              value: "120"
            - name: ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: rook-ceph-object-user-my-store-rgw-admin-user
                  key: AccessKey
            - name: SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: rook-ceph-object-user-my-store-rgw-admin-user
                  key: SecretKey
          ports:
            - containerPort: 8080
              name: metrics
EOF
```

Replace these values with your own:

| Value | Description |
|-------|-------------|
| `my-store` | Your CephObjectStore name |
| `rgw-admin-user` | Your CephObjectStoreUser name |
| `ADMIN_URL` | RadosGW service URL: `http://rook-ceph-rgw-<store>.<namespace>.svc:<port>` |
| `INTERVAL` | Seconds between collection cycles (default: 120) |

The `secret.reloader.stakater.com/reload` annotation restarts the pod when the Rook secret rotates. Requires [Stakater Reloader](https://github.com/stakater/Reloader).

Run only one replica. The embedded NATS KV store is in-process and not shared between instances.

### Step 3: Service for Prometheus

```bash
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Service
metadata:
  name: radosgw-usage-metrics
  namespace: rook-ceph
  labels:
    app: my-store-ext-rgw-exporter
spec:
  selector:
    app: my-store-ext-rgw-exporter
  ports:
    - name: metrics
      port: 8080
      targetPort: 8080
EOF
```

### Step 4: ServiceMonitor (Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: radosgw-usage-metrics
  namespace: rook-ceph
  labels:
    prometheus: kube-prometheus
spec:
  selector:
    matchLabels:
      app: my-store-ext-rgw-exporter
  endpoints:
    - port: metrics
      interval: 120s
      path: /metrics
```

Set `interval` equal to or greater than your `INTERVAL` value.

### Step 5: Verify

```bash
# Check deployment
kubectl -n rook-ceph get deploy my-store-ext-rgw-exporter

# Check logs
kubectl -n rook-ceph logs -l app=my-store-ext-rgw-exporter --tail=30

# Test metrics
kubectl -n rook-ceph port-forward svc/radosgw-usage-metrics 8080:8080 &
curl -s http://localhost:8080/metrics | grep radosgw_
```

## Helm deployment

If you manage Rook-Ceph with Helm, the producer can be templated:

```yaml
# values.yaml
objectstore:
  name: my-store
  prysm:
    repository:
      image: ghcr.io/cobaltcore-dev/prysm
      tag: "0.0.36"
      pullPolicy: IfNotPresent
    rgwAdminUrl: "http://rook-ceph-rgw-my-store.rook-ceph.svc:8080"
    rgwMetrics:
      enabled: true
      interval: "120"
      user:
        name: rgw-admin-user
        store: my-store    # defaults to objectstore.name if omitted
```

The Helm template pulls `AccessKey` and `SecretKey` from the Rook-created secret `rook-ceph-object-user-<store>-<user>` via `secretKeyRef`.

## Environment variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ADMIN_URL` | RadosGW admin API endpoint | | Yes |
| `ACCESS_KEY` | Admin access key (from Rook secret) | | Yes |
| `SECRET_KEY` | Admin secret key (from Rook secret) | | Yes |
| `RGW_CLUSTER_ID` | Cluster ID label for metrics (or use `--rgw-cluster-id`) | | Yes |
| `NODE_NAME` | Node identifier | | No |
| `INSTANCE_ID` | Instance identifier | | No |
| `PROMETHEUS_ENABLED` | Enable metrics endpoint (or use `--prometheus`) | `false` | No |
| `PROMETHEUS_PORT` | HTTP port for metrics | `8080` | No |
| `COOLDOWN_INTERVAL` / `INTERVAL` | Seconds between collection cycles | `120` | No |
| `SYNC_CONTROL_NATS` | Use embedded NATS KV (must be true) | `true` | No |
| `SYNC_EXTERNAL_NATS` | Use external NATS instead | `false` | No |
| `SYNC_CONTROL_URL` | External NATS URL (when `SYNC_EXTERNAL_NATS=true`) | | No |
| `SYNC_CONTROL_BUCKET_PREFIX` | NATS KV bucket name prefix | `sync` | No |

## Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `radosgw_user_buckets_total` | Gauge | user, cluster | Buckets per user |
| `radosgw_user_objects_total` | Gauge | user, cluster | Objects per user |
| `radosgw_user_data_size_bytes` | Gauge | user, cluster | Data size per user |
| `radosgw_usage_bucket_quota_enabled` | Gauge | bucket, user, cluster | Bucket quota enabled (0/1) |
| `radosgw_usage_bucket_quota_size` | Gauge | bucket, user, cluster | Bucket quota max size |
| `radosgw_usage_bucket_quota_size_objects` | Gauge | bucket, user, cluster | Bucket quota max objects |
| `radosgw_usage_user_quota_enabled` | Gauge | user, cluster | User quota enabled (0/1) |
| `radosgw_usage_user_quota_size` | Gauge | user, cluster | User quota max size |
| `radosgw_usage_user_quota_size_objects` | Gauge | user, cluster | User quota max objects |
| `radosgw_usage_bucket_shards` | Gauge | bucket, user, cluster | Shard count per bucket |
| `radosgw_user_metadata` | Gauge | user, display_name, email, cluster | User metadata |

Full list: [metrics reference](../pkg/producers/radosgwusage/README.md).

## Architecture note

The producer starts an embedded NATS server with JetStream. It stores intermediate sync state (users, buckets, usage data) in NATS Key-Value buckets, then computes Prometheus metrics from that state each cycle. No external NATS needed.
