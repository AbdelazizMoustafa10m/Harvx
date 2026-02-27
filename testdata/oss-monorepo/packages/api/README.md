# @acme/api

Express.js REST API server for the Acme Platform.

## Endpoints

- `GET /api/users` - List users (paginated)
- `POST /api/users` - Create a user
- `GET /api/jobs` - List background jobs
- `POST /api/jobs` - Enqueue a new job
- `GET /health` - Health check

## Development

```bash
npm run dev
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3001` | Server listen port |