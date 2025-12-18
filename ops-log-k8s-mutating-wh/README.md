# Prysm Kubernetes Mutating Webhook for RADOSGW Sidecar Injection

This is a Kubernetes **Mutating Admission Webhook** designed to automatically
inject a **Prysm sidecar** into **RADOSGW deployments** managed by Rook-Ceph.
The sidecar container scans **RGW operation logs** and exposes **Prometheus
metrics**.

## Features

- **Automatic Sidecar Injection**: Detects `rook-ceph-rgw` deployments and
  injects a **Prysm sidecar**.
- **Prometheus Metrics**: Extracts metrics from `rgw-ops-logs` and serves them
  on port **9090**.
- **Dynamic Image Configuration**: Supports configuring the sidecar image via
  the `SIDECAR_IMAGE` environment variable.
- **Cert-Manager Integration**: Uses `cert-manager` to generate TLS
  certificates, with **automatic CA bundle injection**.
- **Secure Webhook**: Runs on port **8443** and validates incoming deployments.

---

## **Automatic Sidecar Injection**
The webhook **automatically detects** RADOSGW (`rook-ceph-rgw`) deployments and
injects a **Prysm sidecar** container. It ensures that only specific RADOSGW
instances are modified by checking **a predefined set of labels**.

### **Label Requirements**
To be **eligible for mutation**, a deployment **must have the following
labels**:

| Label | Description |
|-------|-------------|
| `app: rook-ceph-rgw` | Identifies the deployment as an RGW (RADOS Gateway) instance. |
| `app.kubernetes.io/component: cephobjectstores.ceph.rook.io` | Confirms it belongs to the Ceph Object Store component in Rook. |
| `app.kubernetes.io/created-by: rook-ceph-operator` | Ensures that the deployment was created by the Rook-Ceph Operator. |
| `app.kubernetes.io/managed-by: rook-ceph-operator` | Ensures the deployment is managed by Rook. |
| `prysm-sidecar: "yes"` | Enables Prysm sidecar injection _(must be set in `CephObjectStore.spec.gateway.labels`)_. |

>**Important:** The `prysm-sidecar: "yes"` label must be **defined in the Rook CephObjectStore configuration** under `spec.gateway.labels`. Example:

```yaml
apiVersion: ceph.rook.io/v1
kind: CephObjectStore
metadata:
  name: my-store
  namespace: rook-ceph
spec:
  gateway:
    labels:
      prysm-sidecar: "yes"
```

If this label is not set, the webhook will not modify the deployment.

#### **Sidecar Injection Process**
1.	The webhook listens for CREATE and UPDATE operations on Deployment
	resources.
2.	When a new or updated deployment matches the required labels, the
	webhook inspects its pod specification.
3.  If the **Prysm sidecar is missing**, it is **automatically injected** with
    the following configuration:
  - **Container Name**: `prysm-sidecar`
  - **Image**: Defined by `SIDECAR_IMAGE` environment variable.
  - **Args**:
    ```sh
    local-producer ops-log --log-file=/var/log/ceph/ops-log.log --max-log-file-size=10 --prometheus=true --prometheus-port=9090 -v=info
    ```
  - **Ports**:
    - `9090/TCP` (Prometheus metrics endpoint)
  - **Volume Mounts**:
    - `/etc/ceph` (Rook configuration)
    - `/run/ceph` (Ceph daemon sockets)
    - `/var/log/ceph` (RGW operation logs)
    - `/var/lib/ceph/crash` (Crash logs)
  - **Environment Variables**:
    - `POD_NAME`: Auto-populated with the pod’s name.
4.  If a **Prysm sidecar already exists**, the webhook **updates it** to ensure
    consistency with the latest configuration.
5.	The modified deployment is then approved and applied to the cluster.

This ensures consistent, automated sidecar injection into selected
rook-ceph-rgw instances, allowing **real-time monitoring of RGW operations**.

---

## Configure Sidecar via Secret or ConfigMap

The webhook supports injecting **environment variables** into the Prysm sidecar
using either a **Secret** or a **ConfigMap**.  This allows each RADOSGW
deployment to customize the sidecar's behavior independently.

### Option 1: Use a Secret

Add this annotation to your CephObjectStore or RADOSGW deployment:

```yaml
   annotations:
     prysm-sidecar/sidecar-env-secret: "prysm-sidecar-env"
```
The specified secret must exist in the same namespace and look like:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: prysm-sidecar-env
  namespace: rook-ceph
type: Opaque
stringData:
  LOG_FILE_PATH: "/var/log/ceph/ops-log.log"
  MAX_LOG_FILE_SIZE: "10"
  PROMETHEUS_PORT: "9090"
  IGNORE_ANONYMOUS_REQUESTS: "true"
  TRACK_REQUESTS_BY_METHOD: "true"
  TRACK_REQUESTS_BY_STATUS: "true"
  TRACK_ERRORS_BY_USER: "true"
  TRACK_REQUESTS_BY_USER: "true"
  TRACK_REQUESTS_BY_BUCKET: "true"
  TRACK_BYTES_SENT_BY_IP: "true"
  TRACK_BYTES_RECEIVED_BY_IP: "true"
```

### Option 2: Use a ConfigMap

Alternatively, you can use a ConfigMap by setting this annotation:

```yaml
   annotations:
     prysm-sidecar/sidecar-env-configmap: "prysm-sidecar-config"
```
Example ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prysm-sidecar-config
  namespace: rook-ceph
data:
  LOG_FILE_PATH: "/var/log/ceph/ops-log.log"
  MAX_LOG_FILE_SIZE: "10"
  PROMETHEUS_PORT: "9090"
  IGNORE_ANONYMOUS_REQUESTS: "true"
  TRACK_REQUESTS_BY_METHOD: "true"
  TRACK_REQUESTS_BY_STATUS: "true"
  TRACK_ERRORS_BY_USER: "true"
  TRACK_REQUESTS_BY_USER: "true"
  TRACK_REQUESTS_BY_BUCKET: "true"
  TRACK_BYTES_SENT_BY_IP: "true"
  TRACK_BYTES_RECEIVED_BY_IP: "true"
```
### You Can Use Both

If both annotations are set, the sidecar will receive **both** sources via
envFrom, in the order:
1.	Secret (if specified)
2.	ConfigMap (if specified)

This allows sensitive data to be stored in Secrets, while general config can go
in a ConfigMap.

### Benefits

- Each RADOSGW instance can have its own metrics configuration
- Keeps configuration clean and modular
- Avoids hardcoding environment variables into the webhook

---
### Important Notes
> The referenced Secret or ConfigMap must exist before the deployment is
> created, or pod startup may fail.

---

## **Environment Variables**

| Variable         | Description                                      | Default |
|-----------------|--------------------------------------------------|---------|
| `WEBHOOK_PORT`  | Port for the webhook server                      | `8443`  |
| `SIDECAR_IMAGE` | The Prysm sidecar image (use a specific version tag) | _None_  |

### **Best Practice: Use Explicit Version Tags**
It is **strongly recommended** to use a **specific version tag** instead of
`latest`. This ensures:
- **Predictability**: Prevents unexpected changes due to automatic image
  updates.
- **Security**: Avoids potential vulnerabilities in newly pushed images.
- **Stability**: Ensures compatibility with the webhook’s configuration.

#### **Example: Setting a Fixed Version**
```yaml
env:
  - name: SIDECAR_IMAGE
    value: "ghcr.io/cobaltcore-dev/prysm:v1.2.3"
```

This ensures that **every deployment uses the same tested and verified
version** of the Prysm sidecar.

⸻

## **Deployment**

#### **Deploy cert-manager Resources**

The webhook **uses cert-manager** to **generate TLS certificates** and
**automatically inject the CA bundle** into the MutatingWebhookConfiguration.
```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned-issuer
  namespace: webhook
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: prysm-webhook-cert
  namespace: webhook
spec:
  secretName: prysm-webhook-cert
  dnsNames:
    - prysm-webhook-service.webhook.svc
  issuerRef:
    name: selfsigned-issuer
    kind: Issuer
```

---

#### **Deploy the Webhook Server**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prysm-webhook-service
  namespace: webhook
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prysm-webhook-service
  template:
    metadata:
      labels:
        app: prysm-webhook-service
    spec:
      containers:
      - name: prysmwebhook
        image: "ghcr.io/cobaltcore-dev/prysm-wh:v1.2.3"
        ports:
        - containerPort: 8443
        volumeMounts:
        - name: certs
          mountPath: "/certs"
          readOnly: true
        env:
        - name: SIDECAR_IMAGE
          value: "ghcr.io/cobaltcore-dev/prysm:v1.2.3"
        imagePullPolicy: Always
      volumes:
      - name: certs
        secret:
          secretName: prysm-webhook-cert
```

⸻

## **Deploy the Mutating Webhook Configuration**

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: prysm-webhook
  annotations:
    cert-manager.io/inject-ca-from: "webhook/prysm-webhook-cert"
webhooks:
  - name: prysm-webhook.injector.webhook
    clientConfig:
      service:
        name: prysm-webhook-service
        namespace: webhook
        path: "/mutate"
    admissionReviewVersions: ["v1"]
    sideEffects: None
    rules:
      - operations: ["CREATE","UPDATE"]
        apiGroups: ["apps"]
        apiVersions: ["v1"]
        resources: ["deployments"]
```
For more information, visit the [Prysm ops-log local-producer](https://github.com/cobaltcore-dev/prysm/blob/main/pkg/producers/opslog/README.md) documentation.
