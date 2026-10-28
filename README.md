# MiniGate

[Documentation](docs/README.md) · [Kubernetes deployment](docs/deployment.md) · [Releases](docs/releases.md)

MiniGate exposes selected private S3-compatible buckets through public, read-only URLs. Each configured name has its own storage endpoint and credentials. The only object operation is `GetObject`.

```text
GET https://files.example.com/avatars/public/photos/profile.png
                              │       └─ full object key
                              └─ public name configured below
```

## Configuration

```yaml
buckets:
  - name: avatars
    bucket: private-images
    endpoint: https://s3.example.com
    region: us-east-1
    access_key: ACCESS_KEY
    secret_key: SECRET_KEY
    allowed_paths:
      - public/photos
      - public/icons
  - name: documents
    bucket: project-documents
    endpoint: https://objects.other-provider.example
    access_key: OTHER_ACCESS_KEY
    secret_key: OTHER_SECRET_KEY
```

`name` is the first URL path segment; `bucket` is the actual storage bucket. If `bucket` is omitted, it defaults to `name`. `allowed_paths` lists the directories to expose within one bucket. In the example, `/avatars/public/photos/profile.png` reads the same key `public/photos/profile.png` in `private-images`. Every object under `public/photos/` and `public/icons/` is public; keys under other prefixes return 404. The directory names remain in the public URL.

For a single directory, give `allowed_paths` one entry. The object key in the public URL is always the exact key requested from S3; no path is added or removed. If `allowed_paths` is omitted, the whole bucket is public through this route. If `region` is omitted, it defaults to `us-east-1`. Public names must be unique.

Give each configured credential `GetObject` permission only for the intended prefixes when the storage provider supports that policy. Keep the configuration with your deployment secrets; do not commit real credentials. The public URL allows anyone to read every object reachable through that route.

## Run

```bash
go run ./cmd/minigate -config config.yaml -addr :8080
```

The service accepts `GET /{name}/{object-key}` and `GET /health`. Other methods return 405. A missing or inaccessible object returns 404; storage failures return 502. S3 can return `AccessDenied` for a missing object when credentials lack `ListBucket`, so MiniGate maps that error to 404 and logs it. `Range` requests are forwarded to storage for partial downloads. S3 content metadata is forwarded in the response.

HTTP routing uses [chi](https://github.com/go-chi/chi); storage access uses the official AWS SDK for Go v2. The HTTP server itself uses Go's standard `net/http` package.

Responses include `Access-Control-Allow-Origin: *`, so public files can also be read by cross-origin JavaScript `fetch`. Requests that need a CORS preflight are outside this GET-only API.

## Replicas

MiniGate is stateless. Run multiple identical replicas behind a load balancer using the same configuration; no session affinity or shared volume is needed. `GET /health` can serve as a health probe. On SIGTERM, the server stops accepting new connections and waits up to 30 seconds for active responses to finish. Set your deployment's termination grace period longer than 30 seconds if it uses one.
