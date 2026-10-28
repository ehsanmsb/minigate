# Kubernetes deployment

The chart is in [`charts/minigate`](../charts/minigate). Create a Secret from a local configuration file; keep real credentials out of Git:

```sh
kubectl create secret generic minigate-config --from-file=config.yaml=./private-config.yaml
helm upgrade --install minigate ./charts/minigate \
  --set existingSecret=minigate-config \
  --set replicaCount=3
```

The Secret needs a `config.yaml` key. Every replica mounts the same Secret read-only and starts with the same routes. Change the Secret and restart the Deployment to load a new configuration. The application has no shared state or database.

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
