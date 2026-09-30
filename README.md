# gw-wallet

`gw-wallet` is a Go HTTP service for user accounts, multi-currency wallets, and currency exchange. It stores users and wallet balances in PostgreSQL, publishes wallet events through NATS JetStream, and calls a separate gRPC exchange service for rates and conversions.

## Features

- User registration and JWT-based login
- USD, EUR, and RUB wallet balances
- Deposits and withdrawals
- Exchange-rate lookup and currency conversion
- PostgreSQL migrations applied automatically at startup
- Wallet events published to NATS

## Requirements

- Go 1.26 or later for local development
- Docker and Docker Compose for the full local stack
- A running exchange gRPC service for `/api/v1/exchange/rates` and `/api/v1/exchange`

## Run With Docker Compose

The Compose stack starts the wallet service, PostgreSQL, and NATS:

```sh
docker compose up --build
```

The API is available at `http://localhost:8080`. PostgreSQL data and NATS data are persisted in the `postgres_data` and `nats_data` volumes.

By default, the container connects to an exchange service at `host.docker.internal:8081`. Override it when starting the stack if the exchange service is elsewhere:

```sh
EXCHANGEADDR=host.docker.internal:8081 docker compose up --build
```

Useful Compose overrides include `SERVERPORT`, `POSTGRES_PORT`, `NATS_PORT`, and `NATS_MONITOR_PORT`.

Stop the services with:

```sh
docker compose down
```

Add `-v` to also remove the persisted PostgreSQL and NATS data.

## Run Locally

Start PostgreSQL and NATS, then create a configuration file from the example:

```sh
cp config.env.example config.env
```

Adjust `config.env` for the local services. The application reads these variables:

| Variable | Description | Default |
| --- | --- | --- |
| `NATSCONNSTR` | NATS connection string | `nats://localhost:4222` |
| `NATSSTREAM` | JetStream stream name | `wallet` |
| `STORAGECONNSTR` | PostgreSQL connection string | `localhost:5432` |
| `SERVERPORT` | HTTP listen address/port | `8080` |
| `EXCHANGEADDR` | Exchange service gRPC address | `localhost:8081` |
| `AUTHSECRET` | JWT signing secret | `development-secret` |

Run the service:

```sh
go run ./cmd/main.go
```

The service runs its embedded PostgreSQL migrations automatically before accepting requests.

## API

All endpoints are under `/api/v1`. Protected endpoints require:

```http
Authorization: Bearer <token>
```

### Register

```sh
curl -X POST http://localhost:8080/api/v1/register \
	-H 'Content-Type: application/json' \
	-d '{"username":"alice","password":"secret","email":"alice@example.com"}'
```

### Login

The response contains a JWT valid for five minutes.

```sh
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
	-H 'Content-Type: application/json' \
	-d '{"username":"alice","password":"secret"}' | jq -r .token)
```

### Check balance

```sh
curl -H "Authorization: Bearer $TOKEN" \
	http://localhost:8080/api/v1/balance
```

### Deposit or withdraw

Amounts are integer units in the selected currency. Supported currency columns are `USD`, `EUR`, and `RUB`.

```sh
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"currency":"USD","amount":100}'

curl -X POST http://localhost:8080/api/v1/wallet/withdraw \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"currency":"USD","amount":25}'
```

### Exchange currencies

Rates are provided by the exchange service:

```sh
curl http://localhost:8080/api/v1/exchange/rates

curl -X POST http://localhost:8080/api/v1/exchange \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"from_currency":"USD","to_currency":"EUR","amount":10}'
```

## Endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/register` | No | Create a user and wallet |
| `POST` | `/api/v1/login` | No | Return a JWT |
| `GET` | `/api/v1/balance` | Yes | Read the authenticated user's balance |
| `POST` | `/api/v1/wallet/deposit` | Yes | Deposit funds |
| `POST` | `/api/v1/wallet/withdraw` | Yes | Withdraw funds |
| `GET` | `/api/v1/exchange/rates` | No | Fetch exchange rates |
| `POST` | `/api/v1/exchange` | Yes | Convert wallet funds |

The generated API description is available in [`docs/swagger.yaml`](docs/swagger.yaml) and [`docs/swagger.json`](docs/swagger.json). The service does not currently expose a Swagger UI route.

## Development

Run the test suite with:

```sh
go test ./...
```

Some integration tests use Testcontainers and require Docker to be available.

## Project Layout

```text
cmd/                       Application entrypoint
internal/app/              Service wiring and startup
internal/broker/           NATS integration
internal/cache/            In-memory exchange-rate cache
internal/config/           Environment loading
internal/model/            Domain models
internal/server/           HTTP routes and handlers
internal/service/          Authentication, wallet, and exchange logic
internal/storage/postgres/ PostgreSQL storage and migrations
docs/                      Generated API descriptions
```
