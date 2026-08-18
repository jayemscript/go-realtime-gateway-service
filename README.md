# Realtime Gateway Service

Go realtime gateway for authenticated WebSocket clients and server-published events.

## Current phase

Phase 1 establishes the Go module, process lifecycle, configuration loading, and Docker image. HTTP endpoints, WebSockets, authentication, Redis, and event routing are added in later phases.

## Prerequisites

- Go 1.26.6
- Bash (Git Bash on Windows or native Bash on Linux)
- Docker Desktop on Windows or Docker Engine/Compose on Linux

From the repository root, load the local Go environment helper:

```bash
source ./scripts/dev-env.sh
```

## Run locally

Copy the example configuration and start the process:

```bash
cp .env.example .env
go run ./cmd/gateway
```

The Phase 1 process logs its startup and shutdown lifecycle. Public endpoints are introduced in Phase 3.

## Run in Docker

Build and run the image:

```bash
docker build -t realtime-gateway:dev .
docker run --rm --env-file .env -p 8080:8080 realtime-gateway:dev
```

The runtime image is a small non-root image. Redis is not required until the Redis broker phase.

## Verify Phase 1

```bash
go fmt ./...
go vet ./...
go test ./...
docker build -t realtime-gateway:dev .
```

Run the Phase 0 environment check before committing:

```bash
bash ./scripts/preflight.sh
```

## Development workflow

Keep each implementation phase on its own branch and merge the phase PR into `main` before starting the next phase. Do not commit `.env` or other credentials.
