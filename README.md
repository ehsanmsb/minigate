# MiniGate

MiniGate is a lightweight HTTP gateway that serves objects from S3-compatible storage using a clean hexagonal architecture. It maps incoming URL paths to configured buckets and prefixes, then streams the object directly to the client.

## Features
- Hexagonal wiring (ports/adapters) for easy testing and extension
- S3-compatible backends
- Path-based bucket routing with optional prefix mapping
- Streaming responses with metadata headers (Content-Type, Length, ETag, etc.)

## Configuration
`config.yaml` example:
```yaml
buckets:
  - name: avatars
    bucket: avatars
    endpoint: "https://s3.example.com"
    access_key: "ACCESS_KEY"
    secret_key: "SECRET_KEY"
    path: ""
```

### Mapping rules
- URL bucket name comes from `name`
- Actual S3 bucket comes from `bucket` (falls back to `name` if empty)
- `path` is an optional prefix added to the object key

Example request:
```
GET /avatars/theme/images/illustration.png
```

## Run
```bash
go run ./cmd/minigate -config config.yaml -addr :8080
```

## Health
```
GET /health
```

## Status
This project is under active development.
