# URL Shortener API

A learning project built with Go, Fiber v3, and Redis. Create short links with optional custom codes and expiration times, without user accounts.

## Repositories

- **Backend (Go / Fiber / Redis):** [url-shortener-api](https://github.com/Izenberk/url-shortener-api)
- **Frontend (React / Vite):** [url-shortener-web](https://github.com/Izenberk/url-shortener-web)

## Features and architecture

```text
React frontend → POST /api/v1 → Fiber → Redis
Browser → GET /:url → Fiber → Redis lookup → 301 redirect
```

- Validate HTTP/HTTPS URLs, custom codes, and expiry values.
- Generate six-character codes when no custom code is supplied.
- Save a URL and TTL atomically with `SetNX`; reject duplicate custom codes with `409`.
- Retry generated-code collisions up to five attempts.
- Return `404` for unknown or expired codes.
- Apply a basic per-IP creation quota with a 30-minute window.
- Use Fiber logging and CORS middleware.

Redis DB 0 stores URL mappings with TTLs. DB 1 stores quotas and a shared redirect counter. Redis is the database; this project does not use SQL or an ORM.

## Requirements

- Docker Desktop with Linux containers and Docker Compose.
- Go 1.26.4 or a compatible newer toolchain for local execution and tests, as specified in `go.mod`. Host Go is not required to run the Dockerized API.
- Redis 7+ for integration tests using `PEXPIRETIME`.

Commands below use Git Bash from the `url-shortener-api` directory.

## Configuration

Create `.env` in this directory using these local development example values:

```dotenv
API_PORT=:3000
DB_ADDR=redis:6379
DB_PASS=
DOMAIN=http://localhost:3000
API_QUOTA=10
```

| Variable | Meaning |
| --- | --- |
| `API_PORT` | Listen address, including `:`. Use `:3000` with the supplied Compose mapping. |
| `DB_ADDR` | `redis:6379` inside Compose; `127.0.0.1:6379` when running Go on the host. |
| `DB_PASS` | Empty for the supplied local Redis. Setting this does not configure a password on the Redis server. |
| `DOMAIN` | Base URL of the API used in generated links; include the scheme and omit a trailing slash. |
| `API_QUOTA` | Positive integer creation quota per IP, for example `10`. |

Use `API_PORT`, not `APP_PORT`. These are examples, not production credentials. Do not commit real secrets.

## Run the application

```bash
docker compose up -d --build
docker compose ps
docker compose logs --tail=50 api
```

The API runs at `http://localhost:3000`. There is no root `/` health endpoint.

After Go source changes, repeat `docker compose up -d --build`. After `.env` changes:

```bash
docker compose up -d --force-recreate api
```

Stop the application with `docker compose down`. The named `redis_data` volume is retained; adding `-v` deletes it. Redis persistence policy determines which recent writes survive a restart, not the volume alone.

### Run Go on the host instead

Set `DB_ADDR=127.0.0.1:6379` in `.env`, then:

```bash
docker compose up -d --wait redis
go run .
```

Stop any existing Compose API with `docker compose stop api` before binding port 3000. Restore `DB_ADDR=redis:6379` before returning to the Dockerized API.

## OpenAPI documentation

The machine-readable API contract is [openapi.yaml](openapi.yaml), using OpenAPI 3.0.3. It covers both endpoints, validation rules, defaults, examples, current error messages, and the redirect Location header. No authentication is required.

After starting the backend, open **http://localhost:3000/docs** for Swagger UI. The specification is also served at **http://localhost:3000/docs/openapi.yaml**. Expand an endpoint, choose **Try it out**, edit the request, and select **Execute**. The spec defaults to `http://localhost:3000`; update its server URL if your API uses another address.

The UI assets (Swagger UI 5.33.0), initializer, and OpenAPI file are embedded in the Go binary. No CDN or Node.js runtime is needed to view the docs, and the existing Dockerfile works unchanged. Rebuild after editing the spec or documentation assets:

```bash
docker compose up -d --build api
```

Documentation routes are registered before `/:url`. The custom code `docs` is reserved case-insensitively and returns `400` on creation; `docs-link` remains valid. If an older database already contains a code named `docs`, its URL is now occupied by the documentation page; existing data is not automatically changed. There is no `/swagger` alias.

Vendor versions and license files are in `docs/vendor/`. Documentation tests verify HTML, static assets, and the served embedded spec without initializing Redis. You can still import `openapi.yaml` into an external OpenAPI viewer.

Creating links from a viewer writes real data and consumes quota. To inspect a redirect response, use `curl -i` without `-L`; interactive clients may follow the redirect automatically.

Keep the spec updated whenever handlers, validation, or response shapes change. `info.version` is the documentation contract version, not the Fiber or Go version. This file describes current behavior, including `503` quota errors and the existing `404` message.

## API reference

### POST /api/v1

| JSON field | Type | Rules |
| --- | --- | --- |
| `url` | string | Required HTTP/HTTPS URL with hostname; also checked by `govalidator.IsURL`. |
| `short` | string | Optional; empty means generated. Custom codes are case-sensitive, 3–32 ASCII letters, digits, `-`, or `_`; `docs` is reserved in all letter cases. |
| `expiry` | integer | Hours. Omitted or `0` means 24 hours; otherwise 1–720. |

```bash
curl -i http://localhost:3000/api/v1 \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/article","short":"my-link","expiry":24}'
```

Example `200` response; quota values vary:

```json
{
  "url": "https://example.com/article",
  "short": "http://localhost:3000/my-link",
  "expiry": 24,
  "rate_limit": 9,
  "rate_limit_reset": 29
}
```

`expiry` is the configured lifetime in hours, `rate_limit` is remaining quota, and `rate_limit_reset` is remaining quota-window time in whole minutes.

Repeating the same custom code while it exists returns `409`:

```json
{"error":"URL short already in use"}
```

### GET /:url

```bash
curl -i http://localhost:3000/my-link
```

An existing code returns `301` with the destination in `Location`. Unknown or expired codes return `404` with the current message `{"error":"short no found"}`.

| Status | Current behavior |
| --- | --- |
| `200` | Link created. |
| `301` | Redirect. |
| `400` | Invalid JSON or validation failure. |
| `404` | Unknown or expired code. |
| `409` | Custom code already exists. |
| `500` | Database or quota-data error. |
| `503` | Quota exceeded, generated-code attempts exhausted, or current self-domain guard rejection. |

Error responses contain an `error` string. Quota errors also contain `rate_limit_reset`. The current quota status is `503`, not `429`.

## Testing

### Unit and handler tests

Requires Go; Docker and Redis are not required.

```bash
go test ./... -v
```

Covers URL validation, custom code characters and lengths, expiry boundaries, and invalid-request HTTP responses.

### Integration tests

Uses a separate Redis on port 6380; application Redis remains on 6379.

```bash
docker compose --profile test up -d --wait redis-test &&
go test -tags=integration ./... -v -count=1
```

This runs both regular and integration tests without cached results. Files tagged `//go:build integration` are excluded from ordinary `go test ./...`.

Covered behavior:

- Create through the handler, verify stored URL, positive TTL up to one hour, and redirect.
- Reject duplicate codes without changing their URL or expiration timestamp.
- Concurrent requests produce one success and one conflict; storage matches the winner.
- Unknown codes return `404` without redirecting.
- Expired codes return `404`, using a short-lived Redis fixture.

Tests replace global clients and environment values temporarily and clean up their keys. Do not add `t.Parallel()` around this shared setup or run multiple suites against the same test Redis simultaneously.

Clean up:

```bash
docker compose --profile test stop redis-test
docker compose --profile test rm -f redis-test
```

## Code organization

| Path | Responsibility |
| --- | --- |
| `main.go` | App setup and route registration. |
| `docs.go`, `docs/` | Embedded Swagger UI, OpenAPI delivery, and vendor assets. |
| `docs_test.go` | Documentation routing and embedded asset tests. |
| `api/helpers/` | Validation and helper unit tests. |
| `api/routes/shorten.go` | Creation, quota checks, and atomic storage. |
| `api/routes/resolve.go` | Lookup and redirect. |
| `api/routes/*_test.go` | Handler and integration tests. |
| `internal/database/` | Redis clients and startup connection checks. |
| `Dockerfile` | Multi-stage API image build. |
| `docker-compose.yml` | API, application Redis, and optional test Redis. |

## Troubleshooting

| Symptom | Check |
| --- | --- |
| POST `/` returns `404` | Use `/api/v1`. |
| Cannot parse JSON | Send JSON matching the Content-Type header, not form-encoded data. |
| Redis connection fails | Use the correct Compose hostname or host loopback address. |
| Test Redis connection refused | Start `redis-test` and wait for health before running tests. |
| Port 6380 already allocated | Inspect `docker ps`; stop a confirmed old standalone test container if it owns this port. |
| Editor excludes integration files | Set `gopls.buildFlags` to `["-tags=integration"]` in the active VS Code workspace settings. |

## Current limitations

- No accounts, link ownership, management dashboard, or deployment setup.
- Localhost links only work on the device running the API.
- Redis mappings expire, but browsers may cache `301` redirects and bypass later API lookups. Strict browser-visible expiry needs a reviewed redirect/cache policy.
- Link storage is atomic; quota read/check/decrement operations are not yet atomic under concurrency.
- The redirect counter is shared, not per-link analytics.
- CORS uses Fiber defaults for development. Configuration validation, self-domain normalization, persistence policy, and production deployment remain follow-ups.
- Tests do not claim full coverage of quota races, Redis outages, retry exhaustion, or performance.

## Details to complete before submission

- Author/student ID and course: [fill in]
- Screenshots, demo recording, and deployment URL if any: [fill in]
- Design decisions and lessons learned: [fill in]
- Learning sources and assistance used, including AI assistance where applicable: [describe accurately according to course rules]
