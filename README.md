# MiniGate

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
    path: public
    allowed_keys:
      - public/photos/profile.png
      - public/photos/cover.png
  - name: documents
    bucket: project-documents
    endpoint: https://objects.other-provider.example
    access_key: OTHER_ACCESS_KEY
    secret_key: OTHER_SECRET_KEY
```

`name` is the first URL path segment; `bucket` is the actual storage bucket. If `bucket` is omitted, it defaults to `name`. `path` is an optional allowed prefix that remains visible in the URL: `/avatars/public/photos/profile.png` reads the same key `public/photos/profile.png` in `private-images`. With `path: public`, other prefixes return 404. Set `strip_path: true` on that bucket only if you want the old URL form `/avatars/photos/profile.png` to read `public/photos/profile.png`. `strip_path` requires `path`.

`allowed_keys` limits a route to exact S3 keys. In the example, only the two listed image files are public; other files under `public/` return 404. If `allowed_keys` is omitted, every object under `path` is public. If both `path` and `allowed_keys` are omitted, the whole bucket is public through this route. If `region` is omitted, it defaults to `us-east-1`. Public names must be unique.

**Migration:** Existing configurations that used `path` with a hidden prefix need `strip_path: true` to keep their old public URLs.

Give each configured credential `GetObject` permission only for the intended keys or prefix when the storage provider supports that policy. Keep the configuration with your deployment secrets; do not commit real credentials. The public URL allows anyone to read every object reachable through that route.

## Run

```bash
go run ./cmd/minigate -config config.yaml -addr :8080
```

The service accepts `GET /{name}/{object-key}` and `GET /health`. Other methods return 405. A missing object returns 404; storage failures return 502. `Range` requests are forwarded to storage for partial downloads. S3 content metadata is forwarded in the response.

HTTP routing uses [chi](https://github.com/go-chi/chi); storage access uses the official AWS SDK for Go v2. The HTTP server itself uses Go's standard `net/http` package.

Responses include `Access-Control-Allow-Origin: *`, so public files can also be read by cross-origin JavaScript `fetch`. Requests that need a CORS preflight are outside this GET-only API.

## Replicas

MiniGate is stateless. Run multiple identical replicas behind a load balancer using the same configuration; no session affinity or shared volume is needed. `GET /health` can serve as a health probe. On SIGTERM, the server stops accepting new connections and waits up to 30 seconds for active responses to finish. Set your deployment's termination grace period longer than 30 seconds if it uses one.
