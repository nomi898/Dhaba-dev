# Dhaba Orders

A local restaurant ordering system with a React customer/kitchen/statistics UI, FastAPI menu service, Go orders API, Spring Boot queue worker, Laravel statistics API, PostgreSQL and Redis. Run the services directly on your development machine. Container and deployment setup is intentionally deferred.

## Repository layout

- `web/` — React + Vite frontend. Customer ordering, order tracking, kitchen board and live stats.
- `menu-api/` — FastAPI menu endpoints with PostgreSQL reads and Redis caching.
- `orders-api/` — Go order validation, persistence, status endpoints and Redis jobs.
- `kitchen-worker/` — Spring Boot non-web worker, Redis queue consumer and heartbeat.
- `stats-api/` — Laravel API backed by Redis order metrics.
- `database/postgres/` — schema and sample-data SQL scripts.

## Requirements

Install PostgreSQL 14+ and Redis 6+ locally, plus Node.js 20+, Python 3.11+, Go 1.23+, Java 21+, PHP 8.2+ and Composer. No container runtime is required.

## Configure local services

Copy `.env.example` to `.env` and set `POSTGRES_PASSWORD`. In a shell at the repository root, load the values with `set -a; source .env; set +a` before starting backend services. Never commit `.env` files.

On macOS with Homebrew:

```sh
brew install go postgresql@16 redis golangci-lint
brew services start postgresql@16
brew services start redis
export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"

set -a; source .env; set +a
psql -d postgres -c "CREATE ROLE $POSTGRES_USER LOGIN PASSWORD '$POSTGRES_PASSWORD'"
createdb -O "$POSTGRES_USER" "$POSTGRES_DB"
export PGPASSWORD=$POSTGRES_PASSWORD
psql -h localhost -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f database/postgres/001_schema.sql
psql -h localhost -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f database/postgres/002_seed.sql
```

Both SQL scripts are idempotent and safe to re-run. If Redis has a password, set the same value in each service environment.

## Run services

Start each process in a separate terminal from the repository root:

```sh
# Menu API
cd menu-api && python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000
```

```sh
# Orders API
cd orders-api && go run ./cmd/server
```

```sh
# Kitchen worker
cd kitchen-worker && mvn spring-boot:run
```

```sh
# Statistics API
cd stats-api && composer install
cp .env.example .env  # first run only; set APP_KEY with php artisan key:generate
php artisan serve --host 127.0.0.1 --port 8082
```

```sh
# Frontend
cd web && npm install && npm run dev
```

Open the Vite URL shown in the terminal (normally `http://localhost:5173`). The Vite proxy forwards `/api/menu`, `/api/orders` and `/api/stats` to the local APIs. Set `MENU_API_URL`, `ORDERS_API_URL` and `STATS_API_URL` if you use different ports.

The APIs expose health checks at `/health`; the Go service also exposes `/info`. Kitchen status can be advanced from the kitchen board. Set `KITCHEN_AUTO_ADVANCE=false` to disable the worker's timed status progression.

## Tests and quality checks

```sh
cd web && npm test && npm run lint && npm run build
cd menu-api && pytest && ruff check .
cd orders-api && go test ./... && golangci-lint run
cd kitchen-worker && mvn test && mvn checkstyle:check
cd stats-api && composer test && composer analyse
```

## Order lifecycle integration check

1. Confirm all APIs report healthy and the frontend can load menu categories.
2. Place an order from the customer page and confirm a persisted order ID is returned.
3. Confirm the order appears on the kitchen board and moves through `NEW`, `COOKING`, and `READY`.
4. Open order tracking with the returned ID and confirm status updates.
5. Confirm the statistics dashboard reflects the order and item counts.

See `.env.example` for the supported shared configuration keys. PostgreSQL and Redis must be running before using the full order flow.
