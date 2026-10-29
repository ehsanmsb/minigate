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

The chart requests `100m` CPU and `64Mi` memory and limits each container to `500m` CPU and `256Mi` memory. Adjust `resources` in your values file after measuring production traffic.

The chart defaults to `logLevel: WARN`, so routine requests do not fill the logs. Use `--set logLevel=INFO` to include access denials and invalid ranges, or `--set logLevel=DEBUG` to record every object request. Health probes stay silent.

The application waits at most 5 seconds per attempt for S3 response headers and 15 seconds in total for each object response, including streaming. Set `s3Timeout` in chart values if needed. A timeout before response headers returns 504; a stalled stream after headers is terminated and logged.

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

## OpenShift

Log in with `oc`, select your project, and create the configuration Secret in that project. The Route is disabled by default; enable it to expose file URLs through the OpenShift router:

```sh
oc login <cluster-api-url>
oc project <project-name>
oc create secret generic minigate-config --from-file=config.yaml=./private-config.yaml
helm upgrade --install minigate ./charts/minigate \
  --set existingSecret=minigate-config \
  --set replicaCount=3 \
  --set route.enabled=true
oc get route minigate
```

OpenShift assigns a hostname when `route.host` is empty. The resulting URL is `https://<route-host>/<public-name>/<object-key>`. By default, the Route terminates TLS at the router and redirects HTTP to HTTPS. Its router certificate must cover the assigned host. To use a custom domain, set `route.host`, configure DNS for that host, and arrange a matching certificate on the router. If TLS is handled elsewhere, set `route.tls.enabled=false` and configure HTTPS at that edge. The Route targets only the HTTP Service port; metrics remain internal.

The container does not request a fixed user ID, so an OpenShift project can assign a UID under its security context constraints. The image and chart run without privilege escalation and do not write to the root filesystem. Confirm that the cluster can pull the GHCR image and reach your S3 endpoints. If your environment requires a pull Secret, set `imagePullSecrets`.

If your OpenShift cluster uses Contour instead of the built-in router, install Contour and its `projectcontour.io/v1` HTTPProxy CRD first, then enable HTTPProxy with a hostname:

```sh
helm upgrade --install minigate ./charts/minigate \
  --set existingSecret=minigate-config \
  --set httpProxy.enabled=true \
  --set httpProxy.fqdn=files.example.com \
  --set httpProxy.tlsSecretName=files-tls
```

The `files-tls` Secret must contain a certificate for `files.example.com`. Leave `httpProxy.tlsSecretName` empty only if HTTP is intentionally served behind another TLS edge. Enable only the exposure template used by your cluster; Ingress, Route, and HTTPProxy are all disabled by default.

When HTTPProxy is enabled, its route uses `timeoutPolicy.response: 20s`, slightly longer than the application's 15-second `s3Timeout`. This gives the app time to return 504 on a storage timeout. If you increase `s3Timeout`, increase the HTTPProxy response timeout as well.
