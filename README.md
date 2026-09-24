## Testing

Run these commands from the `url-shortener-api` directory.
Examples use Git Bash.

### Unit and handler tests

Requires Go. Docker and Redis are not required.
Covers URL validation and invalid API requests.

```bash
go test ./... -v
```

### Integration tests

Requires Go and Docker Desktop.
Uses a separate Redis instance on port 6380.
Covers URL creation, Redis storage, TTL configuration, and redirects.

```bash
docker compose --profile test up -d --wait redis-test &&
go test -tags=integration ./... -v -count=1
```

This command runs both the regular tests and integration tests.

### Clean up test Redis

```bash
docker compose --profile test stop redis-test
docker compose --profile test rm -f redis-test
```