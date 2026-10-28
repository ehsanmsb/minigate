# MiniGate

MiniGate exposes selected private S3-compatible buckets through public, read-only URLs. Each configured name has its own storage endpoint and credentials. The only object operation is `GetObject`.

```text
GET https://files.example.com/avatars/photos/profile.png
                              │       └─ object key
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
    path: public
  - name: documents
    bucket: project-documents
    endpoint: https://objects.other-provider.example
    access_key: OTHER_ACCESS_KEY
    secret_key: OTHER_SECRET_KEY
```

`name` is the first URL path segment; `bucket` is the actual storage bucket. If `bucket` is omitted, it defaults to `name`. `path` is an optional key prefix, so `/avatars/photos/profile.png` reads `public/photos/profile.png` in `private-images`. Omit `path` or set it to `""` to expose keys from the bucket root. If `region` is omitted, it defaults to `us-east-1`. Bucket names in the configuration must be unique.

Give each configured credential permission to read only its intended bucket's objects. Keep the configuration with your deployment secrets; do not commit real credentials. The public URL allows anyone to read every object reachable through that route.

## Run

```bash
go run ./cmd/minigate -config config.yaml -addr :8080
```

The service accepts `GET /{name}/{object-key}` and `GET /health`. Other methods return 405. A missing object returns 404; storage failures return 502. `Range` requests are forwarded to storage for partial downloads. S3 content metadata is forwarded in the response.

HTTP routing uses [chi](https://github.com/go-chi/chi); storage access uses the official AWS SDK for Go v2. The HTTP server itself uses Go's standard `net/http` package.

Responses include `Access-Control-Allow-Origin: *`, so public files can also be read by cross-origin JavaScript `fetch`. Requests that need a CORS preflight are outside this GET-only API.

## Replicas

MiniGate is stateless. Run multiple identical replicas behind a load balancer using the same configuration; no session affinity or shared volume is needed. `GET /health` can serve as a health probe. On SIGTERM, the server stops accepting new connections and waits up to 30 seconds for active responses to finish. Set your deployment's termination grace period longer than 30 seconds if it uses one.
