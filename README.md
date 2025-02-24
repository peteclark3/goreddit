# GoReddit Firehose

A real-time Reddit post collector and analyzer using Go, Kafka, and PostgreSQL.

## Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Node.js & npm
- Reddit API credentials

## Setup

1. Get Reddit API credentials from https://www.reddit.com/prefs/apps
   - Create a new script app
   - Note down client ID and secret

2. Configure the application:
   ```bash
   cp config/config.example.yaml config/config.yaml
   ```
   Update `config.yaml` with your Reddit credentials

3. Install Go dependencies:
   ```bash
   go mod download
   go mod tidy
   ```

4. Install frontend dependencies:
   ```bash
   cd frontend
   npm install
   cd ..
   ```

## Running

1. Start infrastructure:
   ```bash
   docker-compose up -d
   ```

2. Start the services (in separate terminals):
   ```bash
   # Start producer
   go run cmd/producer/main.go

   # Start consumer
   go run cmd/consumer/main.go

   # Start API
   go run cmd/api/main.go

   # Start frontend (in frontend directory)
   cd frontend
   npm run dev
   ```

3. Visit http://localhost:5173 in your browser

## Shutdown

```bash
docker-compose down
```

To completely reset (including volumes):
```bash
docker-compose down -v
```