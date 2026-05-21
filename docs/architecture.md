# Architecture

JetKZu — пять микросервисов на Go + API Gateway. Все сервисы пакуются в один Go-модуль и собираются одним мульти-стейдж Dockerfile с разными `TARGET`-аргументами.

## Слои в каждом сервисе (clean architecture)

```
delivery/grpc   ──▶ usecase ──▶ repository (interface)
                                    ▲
                                    │
                       infrastructure/postgres (или redis/smtp) — реализация
```

* `domain/` хранит только модели + бизнес-правила (например `CanTransition` для статусов).
* `usecase/` — координирует доменную логику; не знает про SQL/Redis/gRPC.
* `repository/` — пакет с интерфейсами; в нём *нет* импортов pgx/redis. Конкретные реализации лежат в `infrastructure/`.
* `delivery/grpc/` — gRPC handlers; вызывают usecase, конвертируют domain ↔ proto.

`domain → usecase → repository` зависимости идут только вниз. `infrastructure` зависит от `domain`/`repository`, но не наоборот. `cmd/main.go` собирает зависимости.

## Транзакции

| Где | Действие | Файл |
|-----|----------|------|
| user-service          | `CreateWithVerification`: вставка `users` + `verification_tokens` | [services/user/internal/infrastructure/postgres/user_repo.go](../services/user/internal/infrastructure/postgres/user_repo.go) |
| user-service          | `VerifyEmail`: проверка/удаление токена + апдейт `email_verified` | то же |
| driver-service        | `UpdateStatus`: апдейт `drivers` + запись в `driver_status_history` | [services/driver/internal/infrastructure/postgres/repo.go](../services/driver/internal/infrastructure/postgres/repo.go) |
| ride-service          | `CreateWithHistory`: ride + первая запись истории | [services/ride/internal/infrastructure/postgres/repo.go](../services/ride/internal/infrastructure/postgres/repo.go) |
| ride-service          | `UpdateStatusWithHistory`: FOR UPDATE + FSM check + status_history | то же |
| ride-service          | `AssignDriver`: блокирует ride, ставит driver_id и `driver_assigned` | то же |
| payment-service       | `CreateWithEvent`: payment + payment_events | [services/payment/internal/infrastructure/postgres/repo.go](../services/payment/internal/infrastructure/postgres/repo.go) |
| payment-service       | `UpdateStatusWithEvent`: FSM + event внутри tx | то же |

Все транзакции используют `BeginTx → defer Rollback → Commit` шаблон, поэтому если хотя бы один шаг падает, ничего не остаётся в БД.

## Redis (driver-service)

* `driver:geo` — Sorted Set по GEOADD, используется для `FindNearestDrivers` (Redis GEOSEARCH).
* `driver:status:<id>` — текущий status. Кеш фильтрует выдачу: в результате nearest остаются только `online`.

## NATS

`pkg/natsbus/Bus` — тонкая обёртка над `nats.Conn`:

* `Publish(ctx, subject, payload)` сериализует JSON и проставляет `X-Correlation-ID` заголовок.
* `Subscribe(subject, queue, handler)` использует queue-группы, что обеспечивает at-least-once доставку и горизонтальное масштабирование сервисов.

## Observability

* **Logs** — `pkg/logger` оборачивает zap в JSON-формат с полями `service` и `correlation_id`.
* **Metrics** — `pkg/metrics` регистрирует:
  - `jetkzu_grpc_requests_total{service, method, code}`
  - `jetkzu_grpc_latency_seconds_bucket`
  - `jetkzu_nats_events_total{service, subject, direction}`
  - `jetkzu_http_requests_total{method, path, status}`
  - `/health` и `/metrics` поднимаются в каждом сервисе.
* **Tracing-lite** — correlation-id присваивается на gateway, прокидывается в gRPC metadata и в NATS header.
* **Dashboards** — Grafana подхватывает `deploy/grafana/dashboards/jetkzu.json` через provisioning, datasource подцеплен автоматически.

## Graceful shutdown

Все `main.go` слушают `SIGINT/SIGTERM`, вызывают `grpcServer.GracefulStop()` и `srv.Shutdown(ctx)` с 5-секундным таймаутом, закрывают pgx-pool/redis/nats.
