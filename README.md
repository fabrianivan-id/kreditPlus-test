# PT XYZ Multifinance Case Study (Go)

Service built from scratch with clean-architecture layout, MySQL ACID storage, and concurrency-safe limit usage.

## Highlights
- Clean Architecture separation: delivery → usecase → repository → entity
- ACID transactions with row locking (`SELECT ... FOR UPDATE`) for concurrent limit usage
- OWASP Top 10 mitigations: access control (API key), injection prevention (parameterized queries), security headers + CORS, rate limiting, and request body limits
- Unit tests for usecases
- Dockerized app + MySQL

## Project Structure
- `cmd/api` - application entrypoint
- `internal/entity` - domain entities
- `internal/usecase` - business rules
- `internal/repository` - interfaces + MySQL implementations
- `internal/delivery/http` - HTTP handlers + middleware
- `db/schema.sql` - database schema
- `docs/architecture.mmd` - architecture diagram (Mermaid)
- `docs/erd.mmd` - ERD (Mermaid)

## Configuration
Environment variables (defaults in `pkg/config/config.go`):
- `APP_ADDR` (default `:8080`)
- `DB_DSN` (default `root:password@tcp(localhost:3306)/kreditplus?parseTime=true`)
- `API_KEY` (required for non-health endpoints when set)
- `RATE_LIMIT_PER_MIN` (default `60`)
- `BODY_LIMIT_BYTES` (default `1048576`)
- `ALLOWED_ORIGINS` (comma-separated, e.g. `https://app.example.com` or `*`)

## Run Locally
1) Start MySQL and apply schema from `db/schema.sql`
2) Run:
```
go run ./cmd/api
```

## Run with Docker
```
docker compose up --build
```

## API (JSON)
All amounts are in integer Rupiah (IDR).

- `POST /customers`
```json
{
  "nik": "1234567890123456",
  "full_name": "Budi",
  "legal_name": "Budi Santoso",
  "birth_place": "Jakarta",
  "birth_date": "1990-01-01",
  "salary": 5000000,
  "ktp_photo_url": "https://...",
  "selfie_photo_url": "https://..."
}
```

- `POST /customers/{id}/limits`
```json
{
  "tenor_months": 3,
  "amount": 500000
}
```

- `GET /customers/{id}/limits`

- `POST /transactions`
```json
{
  "contract_number": "CN-001",
  "customer_id": 1,
  "tenor_months": 3,
  "asset_name": "Motor",
  "channel": "dealer",
  "otr": 200000,
  "admin_fee": 10000,
  "installment_amount": 20000,
  "interest_amount": 5000
}
```

- `GET /healthz` / `GET /readyz`

## Concurrency Handling
Transaction creation locks the limit row (`SELECT ... FOR UPDATE`) in a database transaction. This prevents race conditions when multiple concurrent transactions attempt to spend the same limit.

## OWASP Top 10 Mitigations Implemented
- Broken Access Control: API key middleware for all non-health endpoints
- Injection: parameterized SQL queries
- Security Misconfiguration: security headers + explicit CORS policy
- Rate limiting and request body size limits for DoS resilience

## Git Flow
Follow Git Flow branching:
- `main` for releases
- `develop` for integration
- `feature/*`, `release/*`, `hotfix/*` for work streams

## Diagrams
Mermaid sources:
- Architecture: `docs/architecture.mmd`
- ERD: `docs/erd.mmd`

# Contributing

## Git Flow
- `main`: production releases only
- `develop`: integration branch
- `feature/*`: new features (merge into `develop`)
- `release/*`: release prep (merge into `main` and back to `develop`)
- `hotfix/*`: urgent fixes (merge into `main` and back to `develop`)

## Commit Hygiene
- Keep commits small and focused
- Use clear messages (e.g., `feat: add limit transaction locking`)

## Tests
Run unit tests before opening a PR:
```
go test ./...
```
