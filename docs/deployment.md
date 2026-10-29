# Kubernetes deployment

The chart is in [`charts/minigate`](../charts/minigate); its [values and optional templates](../charts/minigate/README.md) are documented there. Create a Secret from a local configuration file; keep real credentials out of Git:

```sh
kubectl create secret generic minigate-config --from-file=config.yaml=./private-config.yaml
helm upgrade --install minigate ./charts/minigate \
  --set existingSecret=minigate-config \
  --set replicaCount=3
```

The Secret needs a `config.yaml` key. Every replica mounts the same Secret read-only and starts with the same routes. Change the Secret and restart the Deployment to load a new configuration. The application has no shared state or database.

Only Deployment and Service are enabled by default. For three fixed replicas and protection against one voluntary eviction, add `--set podDisruptionBudget.enabled=true` to the install command. HPA and ServiceAccount also default to disabled; enable them only when needed.

The chart defaults to `logLevel: WARN`, so routine requests do not fill the logs. Use `--set logLevel=INFO` to include access denials and invalid ranges, or `--set logLevel=DEBUG` to record every object request. Health probes stay silent.

For a public hostname, configure your ingress controller and TLS Secret, then enable the optional Ingress:

```sh
helm upgrade --install minigate ./charts/minigate \
  --set existingSecret=minigate-config \
  --set replicaCount=3 \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.host=files.example.com \
  --set ingress.tlsSecretName=files-tls
```

Use a released chart archive from a GitHub release for a matching image tag. If deploying directly from source before a release, set `image.tag` to an existing GHCR tag. The packaged chart defaults to its matching released image version. The service listens internally on port 8080; Kubernetes Service exposes port 80. `/health` is used for readiness and liveness. Ensure the cluster allows outbound HTTPS to each configured storage endpoint and can pull from GHCR.

Prometheus metrics are available on the internal Service port `9090` at `/metrics`. The Ingress routes only to the file-serving port, so it does not expose metrics publicly. If your cluster has Prometheus Operator and its ServiceMonitor CRD, enable `--set serviceMonitor.enabled=true`; set `serviceMonitor.additionalLabels` as needed by your Prometheus selector. Each replica exports its own metrics, including `minigate_configured_buckets`; scrape every pod through service discovery. Prometheus generates `up` per scrape target, while `/health` is the Kubernetes readiness and liveness check.
