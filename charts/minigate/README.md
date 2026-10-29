# MiniGate Helm chart

This chart runs MiniGate as a stateless Kubernetes Deployment. Its default manifests are only a Deployment and a ClusterIP Service. Ingress, OpenShift Route, Contour HTTPProxy, HPA, PDB, ServiceAccount, and ServiceMonitor are opt-in.

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
| `s3Timeout` | `15s` | Maximum time for one S3 response, including body streaming |
| `service.port` | `80` | Internal HTTP Service port |
| `service.metricsPort` | `9090` | Internal Prometheus Service port |
| `ingress.enabled` | `false` | Expose file URLs through an Ingress |
| `route.enabled` | `false` | Create an OpenShift Route |
| `route.host` | `""` | Let OpenShift assign the host when empty |
| `route.tls.enabled` | `true` | Edge TLS and HTTP-to-HTTPS redirect when Route is enabled |
| `httpProxy.enabled` | `false` | Create a Contour HTTPProxy |
| `httpProxy.fqdn` | `""` | Required hostname when HTTPProxy is enabled |
| `httpProxy.tlsSecretName` | `""` | Optional TLS Secret for HTTPProxy |
| `httpProxy.timeoutPolicy.response` | `20s` | Total response limit, slightly longer than the app's default |
| `serviceMonitor.enabled` | `false` | Create a Prometheus Operator ServiceMonitor |
| `autoscaling.enabled` | `false` | Create a CPU-based HPA |
| `podDisruptionBudget.enabled` | `false` | Create a PDB with `maxUnavailable: 1` |
| `serviceAccount.create` | `false` | Create a dedicated ServiceAccount |
| `resources.requests` | `100m` CPU, `64Mi` memory | Pod scheduling and HPA CPU baseline |
| `resources.limits` | `500m` CPU, `256Mi` memory | Default container limits; tune for traffic and object sizes |

The chart also supports `nameOverride`, `fullnameOverride`, `podAnnotations`, `podLabels`, `podSecurityContext`, `securityContext`, `nodeSelector`, `tolerations`, `affinity`, and configurable probes. See [`values.yaml`](values.yaml) for all defaults.

Ingress sends traffic only to the file-serving port. Metrics stay on the internal Service. Enable ServiceMonitor only when its CRD is installed. The chart does not create or store S3 credentials.

For OpenShift, see the [deployment guide](../../docs/deployment.md#openshift). HTTPProxy requires Contour and its `projectcontour.io/v1` CRD; it is not an OpenShift built-in. Enable only the exposure template your cluster uses. Route and HTTPProxy also send traffic only to the file-serving Service port.

The app limits each S3 response to `s3Timeout`. For HTTPProxy, the 20-second response timeout gives the app time to return its own 504 after 15 seconds. If you increase `s3Timeout`, increase `httpProxy.timeoutPolicy.response` too.
