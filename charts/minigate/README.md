# MiniGate Helm chart

This chart runs MiniGate as a stateless Kubernetes Deployment. Its default manifests are only a Deployment and a ClusterIP Service. Ingress, HPA, PDB, ServiceAccount, and ServiceMonitor are opt-in.

## Install

Create a Secret with a `config.yaml` key before installing. Keep S3 credentials out of chart values and Git:

```sh
kubectl create secret generic minigate-config --from-file=config.yaml=./private-config.yaml
helm upgrade --install minigate ./charts/minigate \
  --set existingSecret=minigate-config \
  --set replicaCount=3
```

For three fixed replicas that tolerate one voluntary eviction:

```sh
helm upgrade --install minigate ./charts/minigate \
  --set existingSecret=minigate-config \
  --set replicaCount=3 \
  --set podDisruptionBudget.enabled=true
```

To let Kubernetes adjust replica count, enable `autoscaling.enabled` and set `minReplicas`/`maxReplicas`; `replicaCount` is ignored while HPA is enabled. CPU autoscaling needs a working resource metrics API and a CPU request, which this chart sets by default.

## Main values

| Value | Default | Purpose |
| --- | --- | --- |
| `existingSecret` | `""` | Required Secret containing `config.yaml` |
| `replicaCount` | `1` | Fixed replicas when HPA is off |
| `image.repository` | `ghcr.io/ehsanmsb/minigate` | Container image |
| `image.tag` | `""` | Uses the chart's `appVersion` when empty |
| `imagePullSecrets` | `[]` | Credentials for a private registry |
| `logLevel` | `WARN` | Application log level |
| `service.port` | `80` | Internal HTTP Service port |
| `service.metricsPort` | `9090` | Internal Prometheus Service port |
| `ingress.enabled` | `false` | Expose file URLs through an Ingress |
| `serviceMonitor.enabled` | `false` | Create a Prometheus Operator ServiceMonitor |
| `autoscaling.enabled` | `false` | Create a CPU-based HPA |
| `podDisruptionBudget.enabled` | `false` | Create a PDB with `maxUnavailable: 1` |
| `serviceAccount.create` | `false` | Create a dedicated ServiceAccount |
| `resources.requests` | `100m` CPU, `64Mi` memory | Pod scheduling and HPA CPU baseline |

The chart also supports `nameOverride`, `fullnameOverride`, `podAnnotations`, `podLabels`, `podSecurityContext`, `securityContext`, `nodeSelector`, `tolerations`, `affinity`, and configurable probes. See [`values.yaml`](values.yaml) for all defaults.

Ingress sends traffic only to the file-serving port. Metrics stay on the internal Service. Enable ServiceMonitor only when its CRD is installed. The chart does not create or store S3 credentials.
