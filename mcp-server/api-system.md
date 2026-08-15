# System API

> See [API.md](API.md) for base URLs, authentication, and shared conventions.

## GET /api/stats

Get system statistics.

**Response**
```json
{
  "documents_count": 25,
  "rules_count": 15,
  "projects_count": 8,
  "failures_count": 42
}
```

## POST /api/ingest

Trigger document ingestion from filesystem.

**Response**
```json
{
  "status": "ingest started"
}
```
