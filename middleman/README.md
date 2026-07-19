# Middleman price cache

Small Go HTTP service that caches MetalpriceAPI quotes in SQLite and serves them to MetalTracker (and other clients).

## Behavior

- **Live:** on startup and shortly after each UTC hour (`:00`), fetches `/hourly` for that hour start (falls back to `/latest` if needed) and **appends** a row per metal stamped at minute 00.
- **Historical (past days):** at most one row per metal per calendar day, filled lazily from MetalpriceAPI historical/timeframe when requested.
- Rates are **currency per kilogram** of metal.

## Endpoints

| Path | Notes |
|------|--------|
| `GET /healthz` | Liveness |
| `GET /v1/latest?base=EUR&currencies=XAU,XAG,USD` | Newest snapshot |
| `GET /v1/{YYYY-MM-DD}?base=&currencies=` | Last snapshot that day (lazy-fills past days) |
| `GET /v1/timeframe?base=&currencies=&start_date=&end_date=` | One point per day in range |

Response shape matches MetalTracker’s `MiddlemanProvider` (`base`, `timestamp`, `date`, `rates`).

## Environment

| Variable | Default | Meaning |
|----------|---------|---------|
| `METALPRICE_API_KEY` | _(required)_ | Upstream key (paid plan with `unit=kilogram`) |
| `LISTEN_ADDR` | `:8080` | Bind address |
| `DB_PATH` | `/data/prices.db` in Docker, else `data/prices.db` | SQLite path |
| `POLL_INTERVAL` | `1h` | Hourly append interval |
| `MIDDLEMAN_API_KEY` | _(empty)_ | If set, require `X-API-Key` or `api_key` on `/v1/*` |
| `MIDDLEMAN_HOST_PORT` | `8080` | Host port in Compose |

## Docker deployment (recommended)

```bash
cd middleman
cp .env.example .env
# edit .env - set METALPRICE_API_KEY (and optionally MIDDLEMAN_API_KEY)

docker compose up -d --build
```

Check:

```bash
docker compose ps
curl http://127.0.0.1:8080/healthz
```

SQLite lives in the Docker volume `middleman-data` at `/data/prices.db`.

### Migrate the DB to another host

1. `docker compose stop`
2. Copy the volume file, e.g.  
   `docker run --rm -v middleman_middleman-data:/data -v ${PWD}:/backup alpine tar czf /backup/prices-backup.tar.gz -C /data .`
3. On the new host: restore into a volume mounted at `/data`, then `docker compose up -d`

Or copy `prices.db` out with `docker cp` after stop.

### HTTPS on a VPS

Put Caddy/nginx in front of port 8080 (or change `MIDDLEMAN_HOST_PORT`), terminate TLS there, and point MetalTracker at `https://prices.example.com`. Prefer setting `MIDDLEMAN_API_KEY` when the service is on the public internet.

## Run without Docker

```bash
cd middleman
go mod tidy
# Windows PowerShell:
$env:METALPRICE_API_KEY = "your_key"
go run ./cmd/server
```

## Point MetalTracker at it

In **Settings -> Price source**, choose Middleman and set base URL to e.g. `http://127.0.0.1:8080` or your public HTTPS URL (no trailing path). If `MIDDLEMAN_API_KEY` is set, the desktop app does not send it yet - leave that empty for LAN use, or we can add header support later.
