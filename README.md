# Task Scheduler Service

A RESTful Task Scheduler Service built with Go. Schedule HTTP requests as one-off or recurring tasks with persistent storage and execution tracking.

## Features

- **Flexible Scheduling**: One-off (specific datetime) and recurring (cron) tasks
- **Persistent Storage**: PostgreSQL with automatic migrations
- **Execution Tracking**: Every task execution is logged with detailed results
- **RESTful API**: Complete REST API with pagination and filtering
- **Metrics & Monitoring**: Prometheus metrics with Grafana dashboards
- **Interactive Docs**: Swagger UI for exploring and testing APIs
- **Cluster Safety**: PostgreSQL advisory locks prevent duplicate execution across instances

<br/>

<img width="906" height="535" alt="image" src="https://github.com/user-attachments/assets/f6d005a2-b27f-418b-93b0-8186bf4931e7" />

<br/>

<img width="903" height="539" alt="image" src="https://github.com/user-attachments/assets/3efab544-4b95-475e-915a-4ed6ba09b78f" />

<br/>


## Running Locally

### Prerequisites

- Docker and Docker Compose

### Bootstrapping

1. Clone and start the services:
```bash
git clone <repository-url>
cd task-scheduler
docker-compose up -d
```

The bootstrap process automatically:
- Creates the PostgreSQL database
- Runs database migrations
- Starts the API server
- Configures Prometheus and Grafana

2. Access the services:
- **API**: http://localhost:8080
- **Swagger UI**: http://localhost:8081 (Interactive API documentation)
- **Grafana**: http://localhost:3000 (Username: `admin`, Password: `admin`)
- **Prometheus**: http://localhost:9090 (Metrics)

3. Verify it's running:
```bash
curl http://localhost:8080/health
```

## Configuration

Configuration is managed through environment variables. Copy `.env.example` to `.env` to customize:

```bash
cp .env.example .env
```

Key configuration options:

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `postgres` |
| `DB_NAME` | Database name | `taskscheduler` |
| `PORT` | HTTP server port | `8080` |

## API Examples

### Create a one-off task
Schedule an HTTP request to execute at a specific time:
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Send notification",
    "trigger": {
      "type": "one-off",
      "datetime": "2025-07-01T10:00:00Z"
    },
    "action": {
      "method": "POST",
      "url": "https://httpbin.org/post",
      "payload": {"message": "Hello"}
    }
  }'
```

### Create a cron task
Schedule a recurring task using cron syntax (this example runs daily at 9 AM):
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily health check",
    "trigger": {
      "type": "cron",
      "cron": "0 9 * * *"
    },
    "action": {
      "method": "GET",
      "url": "https://httpbin.org/get"
    }
  }'
```

### List tasks
```bash
curl "http://localhost:8080/api/v1/tasks?status=scheduled&page=1&limit=20"
```

## API Endpoints

- `POST /api/v1/tasks` - Create a new task
- `GET /api/v1/tasks` - List all tasks (paginated)
- `GET /api/v1/tasks/{id}` - Get task details
- `PUT /api/v1/tasks/{id}` - Update a task
- `DELETE /api/v1/tasks/{id}` - Cancel a task
- `GET /api/v1/tasks/{id}/results` - Get execution results for a task
- `GET /api/v1/results` - List all task results (paginated)
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

## Monitoring

Access **Grafana** at http://localhost:3000 to view:
- Task execution rates
- Success/failure counts
- Execution duration percentiles (p50, p90, p95, p99)
- Success rate trends

## API Documentation

### Swagger UI
Interactive API documentation available at http://localhost:8081 where you can:
- Browse all available endpoints
- Test API calls directly from the browser
- View request/response schemas
- See example payloads

### OpenAPI Specification
Complete OpenAPI 3.0 specification available at `api/openapi.yaml`

### Postman Collection
Import the Postman collection for quick API testing:
```bash
api/postman_collection.json
```

### Response Format
All list endpoints return paginated responses:
```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "limit": 20,
  "total_pages": 5
}
```
