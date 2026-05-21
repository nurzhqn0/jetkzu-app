# JetKZu

Локальный аналог Yandex Taxi для Казахстана. Учебный проект на Go: clean-architecture микросервисы, gRPC между сервисами, REST API Gateway, NATS, PostgreSQL, Redis, SMTP, Prometheus + Grafana.

> Один `docker compose up` поднимает всё: 5 микросервисов, gateway, очередь, кеш, базу, миграции и observability stack.

## Архитектура

```
                ┌─────────────┐
   browser ───▶ │ API Gateway │  REST :8080
                │   (Go)      │
                └──────┬──────┘
                       │ gRPC
   ┌───────────────────┼───────────────────────────┐
   ▼                   ▼               ▼           ▼
 user-svc          driver-svc       ride-svc   payment-svc   notification-svc
   :50051            :50052          :50053       :50054         :50055
   │                   │               │             │             │
   │                   │ Redis(geo)    │             │             │ SMTP/Mock
   └─── Postgres ──────┴─── Postgres ──┴─── Postgres ┴─── Postgres ┴─── Postgres
                                  ▲
                                  │
                                NATS
                            ride.requested
                            driver.assigned
                            ride.completed
                            payment.succeeded
                            user.registered  ... etc
```

Каждый сервис организован по clean architecture:

```
services/<svc>/
  cmd/main.go
  internal/
    domain/            # доменная модель + правила
    repository/        # interfaces (porte)
    usecase/           # бизнес-логика
    delivery/grpc/     # gRPC handlers
    infrastructure/    # реализация repo (postgres/redis/smtp)
    config/
  migrations/          # SQL up/down
  tests/               # unit-тесты
```

## Сервисы

| Сервис | gRPC порт | HTTP (health/metrics) | База |
|--------|-----------|------------------------|------|
| user-service         | 50051 | 8081 | jetkzu_users |
| driver-service       | 50052 | 8082 | jetkzu_drivers + Redis |
| ride-service         | 50053 | 8083 | jetkzu_rides |
| payment-service      | 50054 | 8084 | jetkzu_payments |
| notification-service | 50055 | 8085 | jetkzu_notifications |
| api-gateway          | —     | 8080 | — |

## gRPC endpoints (всего: 61)

**UserService (14)**: `RegisterUser`, `LoginUser`, `LogoutUser`, `ValidateSession`, `GetUserProfile`, `GetUserByEmail`, `ListUsers`, `UpdateUserProfile`, `ChangePassword`, `ResetPassword`, `VerifyUserEmail`, `ResendVerification`, `DeactivateUser`, `UpdateUserRole`

**DriverService (14)**: `RegisterDriver`, `AddVehicle`, `UpdateVehicle`, `DeleteVehicle`, `UpdateDriverStatus`, `UpdateDriverLocation`, `GetDriverLocation`, `FindNearestDrivers`, `ListDrivers`, `ListAvailableDrivers`, `AssignDriverToRide`, `GetDriver`, `GetDriverStatusHistory`, `SetDriverRating`

**RideService (14)**: `CreateRide`, `EstimateRidePrice`, `GetRide`, `ListUserRides`, `ListActiveRides`, `ListDriverRides`, `GetRideHistory`, `ScheduleRide`, `AcceptRide`, `RejectRide`, `UpdateRideStatus`, `CancelRide`, `CompleteRide`, `RateRide`

**PaymentService (10)**: `CreatePayment`, `GetPaymentByRide`, `GetPayment`, `ListUserPayments`, `ListFailedPayments`, `GetPaymentReceipt`, `ValidatePaymentMethod`, `ProcessPayment`, `CreateRefundRequest`, `RefundPayment`

**NotificationService (9)**: `SendEmailNotification`, `SendRideReceipt`, `GetNotificationHistory`, `ListUnreadNotifications`, `CountUnreadNotifications`, `MarkNotificationAsRead`, `MarkAllNotificationsAsRead`, `ResendNotification`, `DeleteNotification`

## Frontend

`web/` contains a Vite + React + TypeScript demo SPA with lucide-react icons and a minimal green, white, and blue control-panel design. It exercises auth/profile, driver, ride, payment, notification, and observability flows through the API Gateway.

Local development:

```bash
cd web
npm install
npm run dev
```

Docker Compose serves the built frontend at http://localhost:3000 and proxies `/api` to `api-gateway`.

## REST endpoints

| Метод | Путь | Описание | Auth |
|-------|------|----------|------|
| POST  | `/api/auth/register` | регистрация | — |
| POST  | `/api/auth/login`    | логин | — |
| POST  | `/api/users/verify-email` | подтверждение email | — |
| GET   | `/api/users/me` | профиль | JWT |
| PUT   | `/api/users/me` | обновить профиль | JWT |
| POST  | `/api/drivers/register` | стать водителем | JWT |
| POST  | `/api/drivers/vehicle`  | добавить машину | JWT |
| PATCH | `/api/drivers/status`   | online/offline/busy | JWT |
| PATCH | `/api/drivers/location` | обновить координаты | JWT |
| GET   | `/api/drivers/nearest`  | поиск ближайших | — |
| POST  | `/api/drivers/assign`   | ручное назначение | — |
| POST  | `/api/rides`            | создать поездку | JWT |
| POST  | `/api/rides/estimate`   | расчет цены | — |
| GET   | `/api/rides/{id}`       | по id | JWT |
| GET   | `/api/rides/my`         | мои поездки | JWT |
| PATCH | `/api/rides/{id}/status`| смена статуса | JWT |
| POST  | `/api/rides/{id}/cancel`| отмена | JWT |
| POST  | `/api/rides/{id}/complete`| завершить | JWT |
| POST  | `/api/payments`         | создать платеж | JWT |
| GET   | `/api/rides/{ride_id}/payment` | по ride | JWT |
| POST  | `/api/payments/process` | обработать | JWT |
| POST  | `/api/payments/refund`  | возврат | JWT |
| POST  | `/api/notifications/email` | отправить email | JWT |
| GET   | `/api/notifications/my` | свои уведомления | JWT |
| PATCH | `/api/notifications/{id}/read` | прочитано | JWT |

## NATS события

| Subject | Publisher | Subscribers |
|---------|-----------|-------------|
| `user.registered`      | user-service          | notification-service |
| `user.email_verified`  | user-service          | (reserved)           |
| `ride.requested`       | ride-service          | driver-service, notification-service |
| `ride.status_changed`  | ride-service          | (reserved)           |
| `ride.completed`       | ride-service          | payment-service, notification-service |
| `ride.cancelled`       | ride-service          | payment-service      |
| `driver.location_updated` | driver-service     | (reserved)           |
| `driver.assigned`      | driver-service        | ride-service, notification-service |
| `payment.created`      | payment-service       | (reserved)           |
| `payment.succeeded`    | payment-service       | notification-service |
| `payment.refunded`     | payment-service       | (reserved)           |
| `notification.sent`    | notification-service  | (reserved)           |

Каждое сообщение несёт `X-Correlation-ID`, который Gateway генерирует на входе.

## Быстрый старт

```bash
git clone <repo> && cd JetKZu
cp .env.example .env             # при желании пропишите SMTP

# 1. сгенерировать proto-стабы (не нужно если уже есть в gen/go/)
make proto

# 2. поднять всё
make docker-up
# или: docker compose up -d --build

# 3. посмотреть, что миграции отработали
docker compose logs migrate

# 4. прогнать demo flow одной командой
make demo
```

Через 1-2 минуты после старта будут доступны:

* http://localhost:8080/health — API Gateway
* http://localhost:9090         — Prometheus
* http://localhost:3000         — Grafana (`admin` / `admin`, дашборд "JetKZu Overview")
* http://localhost:8222         — NATS monitoring UI

## Запуск тестов

```bash
make test                                            # unit
make docker-up && make test-integration              # integration (нужен поднятый стек)
```

Unit-тесты бегут в чистом Go-контейнере без зависимости от docker-compose. Integration тест поднимает реальный flow через REST (`tests/integration/flow_test.go`).

## SMTP

По умолчанию используется встроенный `MockEmailSender` — он записывает каждое письмо в лог и в таблицу `notifications`. Чтобы посылать реальные письма, в `.env` укажите:

```
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your@gmail.com
SMTP_PASSWORD=app-password         # Gmail App Password, не основной
SMTP_FROM=your@gmail.com
```

## Тестовый demo flow

`./scripts/demo.sh` делает полный путь:

1. Регистрация пассажира → JWT
2. Регистрация водителя → профиль + автомобиль
3. Водитель идёт online, шлёт координаты в Astana (51.169, 71.449)
4. Пассажир оценивает поездку (`/api/rides/estimate`) и создаёт её
5. `ride.requested` → driver-service подбирает водителя через Redis GEO → `driver.assigned` → ride-service переводит ride в `driver_assigned`
6. Driver переключает ride: `driver_arrived → in_progress → completed`
7. `ride.completed` → payment-service создаёт и обрабатывает платёж
8. `payment.succeeded` → notification-service шлёт receipt

## Документация

* [`docs/architecture.md`](docs/architecture.md)
* [`docs/api-examples.md`](docs/api-examples.md)
* [`docs/grading-checklist.md`](docs/grading-checklist.md)
* [`docs/demo-script.md`](docs/demo-script.md)
* [`docs/github-commit-plan.md`](docs/github-commit-plan.md)
* [`docs/github-team-commit-commands.md`](docs/github-team-commit-commands.md)
