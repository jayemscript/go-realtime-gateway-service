# Realtime Gateway Service — MVP Implementation Plan

## 1. Goal

Build an application-agnostic realtime gateway in Go that:

- accepts authenticated native WebSocket clients;
- receives events from trusted services over HTTP;
- routes events by `appId` and user, or broadcasts them inside one app;
- uses Redis Pub/Sub to fan events out across gateway instances;
- runs locally or in Docker;
- can be tested end to end using only Postman—no backend or frontend required.

The gateway transports ephemeral events. It does not own notification records, unread counts, chat history, orders, or other business data. MVP delivery is best effort; an application's normal HTTP API remains its source of truth.

## 2. Current Repository Status

At the time of planning, the repository contains only:

```text
LICENSE
docs/plan.md
```

There is no Go module, source code, test setup, Docker configuration, or README. Implementation starts from a clean repository.

The installed toolchain is:

```text
go version go1.26.6 windows/amd64
```

`go.exe` is installed at `C:\Program Files\Go\bin\go.exe`, but the current Codex PowerShell process does not see it on `PATH`. Restart Codex/PowerShell after the Go installation or add `C:\Program Files\Go\bin` to `PATH`. Until then, the absolute path can be used.

Redis will run in Docker. Redis does not need to be installed directly on Windows.

## 3. MVP Technical Decisions

### HTTP and WebSocket

- Native RFC 6455 WebSockets, not Socket.IO.
- Go `net/http`; no web framework is needed for this small API.
- JSON request and message payloads.
- `log/slog` JSON logs.
- `github.com/coder/websocket` for upgrades, frames, heartbeat, and test clients.

### Client authentication

- Clients authenticate with JWTs.
- MVP verification uses HS256 with a configured secret.
- The verifier explicitly accepts only the configured algorithm and validates `exp`, `sub`, `appId`, issuer, and audience.
- Postman sends `Authorization: Bearer <token>` in the WebSocket handshake.
- A future browser client may use a short-lived token in `?access_token=` because native browser WebSockets cannot set an Authorization header. Query tokens must be short-lived and request URLs must not be logged.
- A development-only endpoint creates test JWTs so Postman does not depend on another backend. Production startup must fail if this endpoint is enabled.

### Producer authentication

- `POST /v1/events` uses `Authorization: Bearer <service-api-key>`.
- The configured credential has an allowlist of `appId` values.
- The gateway checks the requested `appId`; it never trusts the body alone.
- Multiple credentials and service JWT/JWKS support are later improvements.

### Broker

- Define a small `Broker` interface from the beginning.
- An in-memory adapter enables the first single-process Postman test.
- A Redis Pub/Sub adapter completes the scalable MVP.
- Use one Redis channel, `rt:events`, for the MVP. Every event contains `appId`, and each instance filters against local connections.
- Never store WebSocket connection objects in Redis.

### Routing

Core MVP audience modes are user targeting:

```json
{"userIds":["usr_123","usr_456"]}
```

or app broadcast:

```json
{"all":true}
```

Exactly one audience mode is accepted. Channels and roles are the next module after the core delivery path works.

### Dependencies

Use only:

| Package | Purpose |
|---|---|
| `github.com/coder/websocket` | WebSocket protocol |
| `github.com/golang-jwt/jwt/v5` | JWT creation and verification |
| `github.com/redis/go-redis/v9` | Redis Ping and Pub/Sub |
| `github.com/prometheus/client_golang` | Metrics |

Use standard-library `crypto/rand` for opaque IDs, `log/slog` for logging, `net/http` for routing, and `testing` for tests. Do not add a router, config package, ID package, or assertion library without a demonstrated need.

## 4. API and WebSocket Contract

### HTTP endpoints

| Method | Path | Purpose | Authentication |
|---|---|---|---|
| `GET` | `/health` | Process is alive | None |
| `GET` | `/ready` | Required dependencies are ready | None |
| `GET` | `/metrics` | Prometheus/OpenMetrics | Deployment-controlled |
| `GET` | `/v1/ws` | WebSocket upgrade | Client JWT |
| `POST` | `/v1/events` | Publish an event | Service API key |
| `POST` | `/v1/dev/tokens` | Create a local client JWT | Development only |

### Publish request

```json
{
  "appId": "admin-dashboard",
  "type": "notification.created",
  "audience": {
    "userIds": ["usr_123"]
  },
  "data": {
    "notificationId": "notif_456",
    "title": "New application"
  }
}
```

The gateway supplies the trusted ID, source, and timestamp, publishes the normalized event to the broker, then returns:

```json
{
  "id": "evt_<opaque-id>",
  "status": "accepted"
}
```

Use HTTP `202 Accepted`. It means the configured broker accepted the event, not that every client received it.

### WebSocket messages

Ready message:

```json
{
  "type": "connection.ready",
  "connectionId": "conn_<opaque-id>"
}
```

Delivered event:

```json
{
  "type": "event",
  "id": "evt_<opaque-id>",
  "event": "notification.created",
  "timestamp": "2026-08-18T02:30:00Z",
  "data": {
    "notificationId": "notif_456",
    "title": "New application"
  }
}
```

Protocol error:

```json
{
  "type": "error",
  "code": "INVALID_MESSAGE",
  "message": "message is not valid"
}
```

There is no RPC, replay, or delivery acknowledgement protocol in the MVP.

## 5. Configuration

Use environment variables and validate them once during startup.

| Variable | Local default | Production requirement | Purpose |
|---|---:|---:|---|
| `APP_ENV` | `development` | Required | Runtime mode |
| `HTTP_ADDR` | `:8080` | Optional | Listen address |
| `SHUTDOWN_TIMEOUT` | `10s` | Optional | Graceful shutdown limit |
| `BROKER_DRIVER` | `memory` | Must be `redis` | `memory` or `redis` |
| `REDIS_ADDR` | `redis:6379` in Compose | Required for Redis | Redis address |
| `REDIS_PASSWORD` | empty | Deployment-specific | Redis password |
| `REDIS_DB` | `0` | Optional | Redis database |
| `JWT_SECRET` | no default | Required secret | HS256 secret |
| `JWT_ISSUER` | `realtime-local` | Required | Expected issuer |
| `JWT_AUDIENCE` | `realtime-gateway` | Required | Expected audience |
| `PUBLISH_API_KEY` | no default | Required secret | Publish credential |
| `PUBLISH_SOURCE` | `postman-test-service` | Required | Producer identity |
| `PUBLISH_ALLOWED_APP_IDS` | `admin-dashboard` | Required | Comma-separated app allowlist |
| `ENABLE_DEV_TOKEN_ENDPOINT` | `false` | Must be false | Enables test token creation |
| `ALLOWED_ORIGINS` | empty | Required for browsers | Comma-separated browser origins |
| `MAX_EVENT_BYTES` | `65536` | Optional | Publish body limit |
| `MAX_WS_MESSAGE_BYTES` | `4096` | Optional | Client message limit |
| `PING_INTERVAL` | `25s` | Optional | Heartbeat interval |
| `PONG_TIMEOUT` | `10s` | Optional | Heartbeat timeout |

Commit `.env.example` with placeholders only. Do not commit real secrets or require a `.env` parsing package; Docker Compose, PowerShell, or the deployment system supplies the environment.

## 6. Target Project Structure

```text
realtime-gateway-service/
|-- cmd/
|   `-- gateway/
|       `-- main.go
|-- internal/
|   |-- api/
|   |   |-- server.go
|   |   |-- health.go
|   |   |-- events.go
|   |   `-- dev_tokens.go
|   |-- auth/
|   |   |-- client_jwt.go
|   |   `-- producer.go
|   |-- broker/
|   |   |-- broker.go
|   |   |-- memory.go
|   |   `-- redis.go
|   |-- config/
|   |   `-- config.go
|   |-- event/
|   |   `-- event.go
|   `-- gateway/
|       |-- client.go
|       |-- hub.go
|       |-- protocol.go
|       `-- registry.go
|-- test/
|   |-- integration/
|   `-- postman/
|       |-- realtime-gateway.postman_collection.json
|       `-- local.postman_environment.json
|-- .dockerignore
|-- .env.example
|-- .gitignore
|-- Dockerfile
|-- compose.yaml
|-- go.mod
|-- go.sum
|-- Makefile
|-- README.md
`-- LICENSE
```

README commands must include PowerShell equivalents. The Makefile is optional convenience and must not be required on Windows.

## 7. Implementation Phases

Each phase ends at a working gate. Do not start the next module while the current gate fails.

### Phase 0 — Toolchain and Docker preflight

Tasks:

1. Make `go` available in the repository PowerShell session.
2. Confirm Docker Desktop is running.
3. Verify:

   ```powershell
   go version
   docker version
   docker compose version
   git status --short
   ```

4. Confirm no `.pnpm-store`, dependency caches, logs, binaries, temporary files, or secrets are inside the repository.

Gate:

- Go reports `go1.26.6 windows/amd64`.
- Docker and Docker Compose respond successfully.
- Only intentional files are present.

### Phase 1 — Basic Go project setup

Tasks:

1. Initialize `go.mod` using the final repository module path.
2. Add only the essential directories.
3. Add `.gitignore`, `.dockerignore`, `.env.example`, and README.
4. Add application startup, structured logging, signal handling, and graceful HTTP shutdown.
5. Add standard-library environment configuration parsing and validation.
6. Add a multi-stage Dockerfile:
   - pinned Go builder image compatible with `go.mod`;
   - `CGO_ENABLED=0` release build;
   - small non-root runtime image;
   - only the compiled gateway copied into runtime;
   - no source, Go cache, or secrets in the final image.

Commands:

```powershell
go mod init <final-module-path>
go fmt ./...
go vet ./...
go test ./...
docker build -t realtime-gateway:dev .
```

Gate:

- The native service starts and exits cleanly on Ctrl+C.
- The gateway Docker image builds and starts.
- Invalid configuration gives one clear startup error.
- No third-party dependency has been added yet.

### Phase 2 — Install minimal required packages

Install dependencies deliberately:

```powershell
go get github.com/coder/websocket
go get github.com/golang-jwt/jwt/v5
go get github.com/redis/go-redis/v9
go get github.com/prometheus/client_golang
go mod tidy
go mod verify
```

Gate:

- `go.mod` and `go.sum` contain only required dependencies.
- `go mod verify`, `go vet ./...`, and `go test ./...` pass.
- No dependency cache, `vendor`, or `.pnpm-store` appears in the repository.

Go may remove packages that are installed but not imported yet. That is expected; add each dependency as its module is implemented, then run `go mod tidy`.

### Phase 3 — First module: HTTP foundation and lifecycle

Build:

- HTTP server with explicit timeouts;
- `GET /health` returning `200 {"status":"ok"}`;
- `GET /ready` returning broker readiness;
- JSON response and error helpers;
- request IDs;
- graceful shutdown that marks readiness false before draining requests;
- in-memory broker selected by `BROKER_DRIVER=memory`.

Gate:

- Unknown routes return JSON `404`.
- Unsupported methods return JSON `405`.
- `/health` does not depend on Redis.
- `/ready` reflects the selected broker.

### Phase 4 — Test the foundation locally and in Docker

Automated tests:

- config defaults and validation;
- health response status, headers, and body;
- readiness transitions;
- method and not-found behavior;
- graceful shutdown where practical.

Commands:

```powershell
go test ./...
go test -race ./...
go vet ./...
docker build -t realtime-gateway:dev .
docker run --rm -p 8080:8080 --env-file .env realtime-gateway:dev
```

If the Windows race detector requires a C compiler that is not installed, document it and run the race job in Linux CI. Do not silently skip it.

Postman checks against the container:

1. `GET http://localhost:8080/health` returns `200`.
2. `GET http://localhost:8080/ready` returns `200` with the memory broker.
3. `POST http://localhost:8080/health` returns `405`.
4. `GET http://localhost:8080/not-found` returns JSON `404`.

Gate:

- Automated tests pass.
- The same application works natively and as a Docker container.
- Postman confirms the public HTTP contract.

### Phase 5 — Next module: WebSocket connection hub

Build:

- `GET /v1/ws` upgrade handler;
- generated connection IDs;
- concurrency-safe local registry;
- multiple connections per app/user;
- one bounded outbound queue per connection;
- one read loop and one write loop per connection;
- server ping/client pong heartbeat;
- maximum client message size;
- clean unregister on disconnect;
- `connection.ready` message.

Only during this phase, allow a clearly development-only identity input so transport can be tested before JWT work. Delete this bypass in Phase 7.

Postman gate:

- Two WebSocket tabs connect to `ws://localhost:8080/v1/ws`.
- Each receives a unique `connection.ready` message.
- Closing one removes only that connection.
- Heartbeat eventually closes an unresponsive connection.
- The checks pass against the Docker container.

### Phase 6 — Publishing and local routing module

Build:

- strict event and audience types;
- JSON decoding with unknown-field rejection;
- maximum body size;
- validation, normalization, generated event ID, and timestamp;
- `POST /v1/events`;
- service API-key middleware;
- publisher app allowlist;
- memory broker publish/subscribe;
- user-targeted delivery;
- app-wide broadcast;
- slow-client policy: disconnect a client whose outbound queue remains full instead of blocking the hub.

Validation rules:

- `appId` and `type` are required and length-limited;
- `data` is valid JSON and otherwise opaque;
- payload size is bounded;
- user IDs are trimmed, non-empty, deduplicated, and count-limited;
- exactly one audience mode is present;
- the producer cannot set trusted source, timestamp, or delivery status;
- the producer must be authorized for the requested app.

Postman gate:

1. Connect two users under `admin-dashboard` and one under `customer-portal`.
2. Publish to one user; only that user's tabs receive it.
3. Broadcast to `admin-dashboard`; the other app receives nothing.
4. A bad API key returns `401`.
5. A disallowed app returns `403`.
6. Invalid or oversized input returns `400` or `413` with a stable error code.

### Phase 7 — JWT authentication and app isolation

Build:

- typed claims: `sub`, `appId`, `roles`, `iss`, `aud`, `exp`, `iat`;
- HS256 verification with explicit algorithm enforcement;
- token extraction from the WebSocket Authorization header;
- optional short-lived `access_token` query support for future browser clients;
- development-only `POST /v1/dev/tokens`;
- production startup rejection when the development endpoint is enabled;
- browser origin allowlist while permitting clients with no `Origin`, such as Postman;
- removal of the Phase 5 identity bypass.

Example development token request:

```json
{
  "userId": "usr_123",
  "appId": "admin-dashboard",
  "roles": ["admin"],
  "expiresInSeconds": 900
}
```

Postman flow:

1. Call `POST /v1/dev/tokens` for each identity.
2. In a WebSocket request, set `Authorization: Bearer {{clientToken}}` before connecting.
3. Connect to `ws://localhost:8080/v1/ws`.
4. Publish with `Authorization: Bearer {{serviceApiKey}}`.
5. Observe the event in the intended tab.

Security gate:

- missing, malformed, expired, wrong-algorithm, wrong-issuer, or wrong-audience JWTs are rejected;
- missing `sub` or `appId` is rejected;
- registry identity comes only from verified claims;
- tokens and Authorization headers never appear in logs;
- equal user IDs in different apps never cause cross-app delivery.

### Phase 8 — Docker Compose, Redis, and multi-instance delivery

Build:

- Redis implementation of `Broker`;
- startup Ping and readiness state;
- publish normalized events to `rt:events`;
- subscription loop with cancellation and bounded retry/backoff;
- clean subscription close during shutdown;
- `compose.yaml` with:
  - Redis using an explicit image version;
  - Redis healthcheck;
  - gateway image built from this repository;
  - gateway dependency on healthy Redis;
  - named development network;
  - no unnecessary persistent Redis volume for ephemeral Pub/Sub;
  - one-service profile on port `8080`;
  - two-instance test profile on ports `8081` and `8082`;
- distinct gateway instance IDs in logs.

The gateway must publish through the broker and consume through the same subscription path as every other instance. It must not separately deliver locally and create duplicate events.

Commands:

```powershell
docker compose up --build redis gateway
docker compose --profile scale up --build
docker compose down
```

Cross-instance Postman gate:

1. Start Redis plus gateway instances on ports `8081` and `8082`.
2. Connect a WebSocket to gateway 2.
3. Send `POST /v1/events` to gateway 1.
4. Confirm the gateway 2 client receives exactly one event.
5. Connect the same user to both instances; each connection receives one event.
6. Stop Redis; `/health` remains `200` and `/ready` becomes `503`.
7. Restore Redis; subscription and readiness recover without restarting gateways.

Gate:

- Redis is supplied entirely by Docker.
- The gateway is also running in Docker.
- Postman proves cross-container, cross-instance delivery.

### Phase 9 — Observability and production hardening

Build:

- `/metrics`;
- active connections gauge;
- connection, accepted-event, delivered-event, dropped-event, WebSocket-error, and Redis-error counters;
- publish latency histogram;
- rate and connection limits;
- trusted proxy policy before accepting forwarded IP headers;
- consistent WebSocket close codes and HTTP error codes;
- production TLS/reverse-proxy documentation;
- final non-root container verification;
- graceful shutdown that stops new upgrades and publishes before closing clients.

Never use `userId`, `eventId`, or `connectionId` as metric labels. They create unbounded cardinality. They may be structured log fields.

Gate:

- metrics reflect connection, publish, delivery, drop, and error tests;
- logs have instance, connection, app, user, and event context without tokens or full payloads;
- container shutdown completes within the grace period;
- production configuration cannot use the memory broker or dev token endpoint.

### Phase 10 — Next routing module: channels and roles

This phase follows the notification-focused core and does not block the first release unless a real consumer requires it immediately.

Build:

- client `subscribe` and `unsubscribe` messages;
- normalized, length-limited channel names;
- per-connection membership;
- `audience.channel` routing;
- `audience.roles` routing using only verified JWT roles;
- exactly one of `userIds`, `all`, `channel`, or `roles` per event.

Gate:

- only subscribed connections receive channel events;
- only matching verified roles receive role events;
- neither mode crosses an `appId` boundary;
- reconnect creates a new connection; future client SDKs restore subscriptions.

### Phase 11 — Final automated and Postman test assets

Commit a Postman collection and local environment with secret placeholders only.

Collection folders:

```text
01 Health and readiness
02 Development tokens
03 WebSocket connection instructions
04 Publish to user
05 Broadcast to app
06 Authentication failures
07 Validation failures
08 App isolation
09 Redis cross-instance checklist
```

Final verification:

```powershell
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go mod tidy
go mod verify
docker build -t realtime-gateway:dev .
docker compose config
docker compose up --build -d
docker compose ps
```

Integration tests cover:

- one user with multiple tabs;
- user routing, broadcast, and app isolation;
- disconnect cleanup and heartbeat;
- slow-client handling;
- invalid JWT and API key cases;
- Redis cross-instance propagation;
- Redis failure and recovery;
- graceful shutdown.

Final gate:

- A new developer can clone, copy `.env.example`, run Docker Compose, generate a JWT, connect a Postman WebSocket, publish an HTTP event, and receive it using only README instructions.
- The Postman collection works against memory and Redis modes except for the explicitly Redis-only scale checks.
- Tests pass and the repository contains no caches, logs, binaries, coverage output, `.pnpm-store`, or real secrets.

## 8. HTTP Error Contract

All HTTP errors use:

```json
{
  "error": {
    "code": "INVALID_EVENT",
    "message": "event is not valid",
    "requestId": "req_<opaque-id>"
  }
}
```

Initial codes:

```text
INVALID_JSON
INVALID_EVENT
PAYLOAD_TOO_LARGE
UNAUTHORIZED
FORBIDDEN_APP
METHOD_NOT_ALLOWED
NOT_FOUND
BROKER_UNAVAILABLE
RATE_LIMITED
INTERNAL_ERROR
```

Never expose Redis addresses, stack traces, JWT validation details, or secrets to callers.

## 9. Testing Strategy

### Unit tests

Use table-driven tests for configuration, event validation, JWT verification, registry matching, protocol parsing, and middleware.

### HTTP and WebSocket integration tests

Use `httptest.Server` and a WebSocket client to exercise real handlers and the memory broker without fixed ports.

### Redis integration tests

Run tests against the Redis Compose service. Keep Redis-dependent tests behind an explicit integration command or build tag so `go test ./...` stays deterministic when Docker is unavailable.

### Docker tests

- Build the image from a clean context.
- Run the same health and Postman flow against the container.
- Use Compose for Redis and scale tests.
- Do not mount source code or Go caches into the production-like container test.

### Postman

Postman is the manual acceptance client for both sides:

- HTTP requests generate development JWTs and publish events.
- WebSocket requests connect with JWTs and display delivered events.
- Variables hold `baseHttpUrl`, `baseWsUrl`, `clientToken`, and `serviceApiKey`.

Postman desktop supports WebSocket handshake headers. Configure headers before connecting.

## 10. MVP Definition of Done

The core MVP is complete when:

1. The service builds with Go 1.26.6.
2. The service runs natively and as a non-root Docker container.
3. Docker Compose initializes Redis and the gateway without a host Redis installation.
4. Health and readiness are correct and independently meaningful.
5. Postman can generate a local JWT without another backend.
6. Postman can establish an authenticated WebSocket connection.
7. One user can have multiple simultaneous connections.
8. An authenticated service can publish to users or broadcast within one app.
9. Invalid credentials and disallowed apps are rejected.
10. Matching user IDs in different apps never cause cross-app delivery.
11. Redis carries an event between two gateway containers without duplicate delivery.
12. Disconnect, heartbeat, Redis recovery, slow-client, and shutdown behavior are tested.
13. Metrics and logs provide useful operations data without leaking secrets.
14. README and Postman assets reproduce the complete flow.

Channels and roles are the next module and do not block the notification-focused first release.

## 11. Explicitly Out of Scope

- notification persistence and unread counts;
- durable queues, Redis Streams, Kafka, or NATS;
- replay and offline WebSocket delivery;
- exactly-once or guaranteed delivery;
- delivery acknowledgements;
- Socket.IO compatibility;
- RPC over WebSocket;
- complex presence;
- mobile push through FCM/APNs;
- frontend/NestJS SDKs before the protocol is stable;
- an admin UI;
- multi-region replication;
- business-specific permission lookups on each event.

## 12. Implementation Order Summary

```text
Phase 0   Go + Docker preflight
Phase 1   Basic setup + Dockerfile
Phase 2   Minimal dependencies
Phase 3   HTTP/lifecycle module
Phase 4   Native + Docker + Postman foundation test
Phase 5   WebSocket hub module
Phase 6   Publish API + local routing module
Phase 7   JWT auth + app isolation module
Phase 8   Compose + Redis + two gateway containers
Phase 9   Observability + hardening
Phase 10  Channels + roles (next module)
Phase 11  Final automated + Postman release gate
```

Phases 0–7 produce the first useful Postman-tested vertical slice using the memory broker. Phases 8–9 complete the Dockerized, horizontally scalable MVP. Phase 10 extends routing only after the core notification path has been proven.
