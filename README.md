# URL Shortener

A small Go and MySQL service that creates cryptographically random, eight-character short links and redirects them to validated HTTP or HTTPS destinations. The browser client is dependency-free and embedded in the server binary.

## Project layout

```text
cmd/urlshortener/          executable entrypoint and process signals
config/                    configuration loading, defaults, and validation
configs/                   non-secret runtime configuration files
core/models/               dependency-free domain entities
core/repository/           persistence contract
core/repository/mysql/     MySQL adapter and database migrations
core/service/              URL-shortening business rules
web/                       HTTP handlers, middleware, lifecycle, embedded UI
web/static/                dependency-free browser client
tests/                     external black-box tests and benchmarks
```

Dependencies point inward: `models` has no project dependencies, `repository` depends on models, `service` depends on the repository contract and models, and `web` composes those layers. Interfaces live with their consumers where practical. Constructors inject dependencies, and functional options provide deterministic clocks for tests without exposing internal state.

## Responsibilities

- `web.Application` owns server limits and graceful shutdown.
- `web.Handler` owns HTTP parsing, response formatting, headers, and embedded assets.
- `service.URLShortener` owns creation, reuse, expiry, collision retries, and resolution.
- `repository.LinkRepository` isolates persistence; `mysql.Repository` reuses one bounded pool.
- `service.URLValidator`, `service.CryptoSlugGenerator`, and `web.TokenBucketLimiter` each own one policy.

## Requirements

- Go 1.26.6 or newer.
- MySQL 8 or newer.

## Setup

Create a database and least-privilege application user:

```sql
CREATE DATABASE blogs CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
CREATE USER 'url_shortener'@'localhost' IDENTIFIED BY 'replace-with-a-secret';
GRANT SELECT, INSERT, UPDATE ON blogs.* TO 'url_shortener'@'localhost';
```

Apply the schema and download Go modules:

```sh
mysql -u url_shortener -p blogs < core/repository/mysql/migrations/000_create_shortlink.sql
make setup
```

For a database created by the original schema, take a backup and apply `core/repository/mysql/migrations/001_harden_shortlink.sql` once instead of recreating the table.

Do not commit the database password. Supply it at runtime:

```sh
export URL_SHORTENER_DB_PASSWORD='replace-with-a-secret'
make server
```

The checked-in `configs/config.json` contains non-secret local defaults. Supported environment overrides are:

| Variable | Purpose |
| --- | --- |
| `URL_SHORTENER_CONFIG` | Alternate JSON configuration path |
| `URL_SHORTENER_PORT` | HTTP listen port |
| `URL_SHORTENER_PUBLIC_BASE_URL` | Trusted public origin used to construct short URLs |
| `URL_SHORTENER_DB_NAME` | Database name |
| `URL_SHORTENER_DB_USERNAME` | Database user |
| `URL_SHORTENER_DB_PASSWORD` | Database password; environment-only |
| `URL_SHORTENER_DB_HOST` / `URL_SHORTENER_DB_PORT` | Database endpoint |
| `URL_SHORTENER_DB_TLS_MODE` | MySQL driver TLS mode; use `true` outside localhost |
| `URL_SHORTENER_DB_MAX_OPEN_CONNECTIONS` | Pool concurrency limit |
| `URL_SHORTENER_DB_MAX_IDLE_CONNECTIONS` | Warm idle connection limit |
| `URL_SHORTENER_RATE_PER_MINUTE` | Per-peer create refill rate |
| `URL_SHORTENER_RATE_BURST` | Per-peer create burst capacity |
| `URL_SHORTENER_RATE_MAX_CLIENTS` | Maximum in-memory limiter entries |

Terminate HTTPS at a trusted reverse proxy and set `public_base_url` to the external HTTPS origin. The limiter ignores spoofable forwarded-IP headers and keys the immediate TCP peer.

## API

```sh
curl --fail-with-body \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/a/long/path"}' \
  http://localhost:8080/
```

The first request returns `201 Created`; requesting the same active URL returns `200 OK`. `GET /{slug}` returns a `302` redirect and increments the hit count.

## Verification

```sh
make test       # external tests, race detector, and coverage
make vet
make build
make vuln       # official Go vulnerability database
make benchmark
```

All source and configuration files remain below 500 lines.
