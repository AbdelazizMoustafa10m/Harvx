# API Documentation

## Endpoints

### GET /health

Returns the service health status and version.

**Response:**

```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

### GET /users

Returns a list of all users.

**Response:**

```json
{
  "users": [],
  "total": 0
}
```

## Authentication

All API endpoints require a valid Bearer token in the Authorization header.

## Rate Limiting

Rate limited to 100 requests per minute per API key.