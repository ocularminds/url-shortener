# URL Shortener

A small Go and MySQL service that creates cryptographically random, eight-character short links and redirects them to validated HTTP or HTTPS destinations. The browser client is dependency-free.

## Architecture

The backend is split by responsibility:

- `Application` owns process lifecycle, HTTP server limits, and graceful shutdown.
- `Handler` owns HTTP parsing, response formatting, security headers, and static files.
- `URLShortener` owns link creation, reuse, expiry, collision retries, and resolution.
- `LinkRepository` separates persistence from business logic; `MySQLRepository` reuses one bounded connection pool.
- `URLValidator`, `CryptoSlugGenerator`, and `TokenBucketLimiter` each own one security policy.

Dependencies are passed through constructors, so tests use deterministic in-memory collaborators and do not need MySQL.

## Requirements

- Go 1.26.6 or newer. This minimum includes standard-library security fixes detected during the review.
- MySQL 8 or newer.

## Setup

Create a database and least-privilege application user, then apply the schema:

```sql
CREATE DATABASE blogs CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
CREATE USER 'url_shortener'@'localhost' IDENTIFIED BY 'replace-with-a-secret';
GRANT SELECT, INSERT, UPDATE ON blogs.* TO 'url_shortener'@'localhost';
```

```sh
mysql -u url_shortener -p blogs < shortner.sql
make setup
```

For an existing database created by the original schema, take a backup and apply `migrations/001_harden_shortlink.sql` once instead of recreating the table.

Do not commit the database password. Supply it at runtime:

```sh
export URL_SHORTENER_DB_PASSWORD='replace-with-a-secret'
make server
```

The checked-in `config.json` contains non-secret local defaults. Supported environment overrides are:

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
| `URL_SHORTENER_VIEWS_DIR` | Static client directory |

Terminate HTTPS at a trusted reverse proxy and set `public_base_url` to the external HTTPS origin. The limiter deliberately ignores spoofable forwarded-IP headers, so it keys the immediate TCP peer; behind a reverse proxy, it limits that proxy as one peer unless trusted-proxy support is added explicitly.

## API

Create or retrieve a short link:

```sh
curl --fail-with-body \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/a/long/path"}' \
  http://localhost:8080/
```

The first request returns `201 Created`; requesting the same active URL returns `200 OK`. `GET /{slug}` returns a `302` redirect and increments the hit count.

## Verification

```sh
make test       # race detector and coverage
make vet
make build
make vuln       # official Go vulnerability database
make benchmark
```

All owned source and configuration files are kept below 500 lines. The previous 16 MB vendored browser theme was removed, reducing transfer and dependency exposure.
