# Deploy

This folder contains Docker assets for local development and deployment.

## Local development

From the repo root:

```
make dev
```

This starts:
- Postgres on `localhost:5432`
- Backend WS/HTTP server on `localhost:8080`
- Next.js dev server on `localhost:3000`

### Environment

Compose uses:
- `deploy/env/backend.dev.env` for backend configuration
- `deploy/env/web.dev.env` for frontend configuration

Update these files if you need different ports or DSNs.

## Production builds

The Dockerfiles are multi-stage and include a `prod` stage you can target in
your own deployment pipelines.
