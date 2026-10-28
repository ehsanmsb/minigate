# Configuration and routes

MiniGate serves `GET /{name}/{object-key}`. The URL keeps the complete S3 object key. Each `name` may point to a different bucket, endpoint and credential.

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
```

Here `/avatars/public/photos/profile.png` reads `public/photos/profile.png` from `private-images`. Keys outside `allowed_paths` return 404. Omit `allowed_paths` to expose the entire bucket through that name. Give the credential only `GetObject` permission for the intended objects. Anyone with a public URL can read a reachable object.

See the root [README](../README.md) for local execution and HTTP behavior.
