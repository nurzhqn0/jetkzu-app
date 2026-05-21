# Grading checklist (полное соответствие критериям)

| # | Критерий | Вес | Реализация | Где смотреть |
|---|----------|-----|------------|--------------|
| 1 | Clean architecture | 20% | Каждый сервис разделён на `domain → usecase → repository ← infrastructure`, делiver/grpc вызывает только usecase. | [services/user/internal/](../services/user/internal/), [services/ride/internal/](../services/ride/internal/) — структура одинаковая во всех 5 сервисах |
| 2 | ≥ 12 gRPC endpoints | 20% | **61 gRPC методов** в 5 сервисах; each member owns at least 12 methods | [proto/](../proto/), сводка ниже |
| 3 | Message Queue (NATS) | 20% | 12 subject'ов, queue-группы, publish/subscribe цепочки | [pkg/natsbus/](../pkg/natsbus/) + подписки в `services/*/cmd/main.go` |
| 4 | DB + Cache + migrations + transactions | 20% | PostgreSQL (5 БД), Redis GEO, `migrate/migrate` контейнер, ≥ 7 транзакций | см. [architecture.md](architecture.md#транзакции) |
| 5 | SMTP email | 10% | Реальный SMTP (Gmail/Outlook) если есть `SMTP_*` env, иначе MockSender | [services/notification/internal/infrastructure/smtp/sender.go](../services/notification/internal/infrastructure/smtp/sender.go) |
| 6 | Unit + Integration tests | 10% | 6 unit-наборов + E2E flow через gateway | [pkg/jwt/jwt_test.go](../pkg/jwt/jwt_test.go), [tests/integration/flow_test.go](../tests/integration/flow_test.go) |
| 7 | Frontend (бонус) | +10% | Vite + React + TypeScript SPA with lucide-react icons, green/white/blue minimal dashboard | [web/](../web/) |
| 8 | Grafana + tracing + metrics + logs (бонус) | +10% | Prometheus scrape конфиг, Grafana provisioning + дашборд, structured zap logs, correlation_id ↔ gRPC metadata ↔ NATS headers | [deploy/](../deploy/), [pkg/metrics/metrics.go](../pkg/metrics/metrics.go), [pkg/logger/logger.go](../pkg/logger/logger.go) |

## gRPC endpoints (61 шт)

| Member | Service area | Count | Methods |
|--------|--------------|-------|---------|
| Nurzhan | Driver | 14 | RegisterDriver, AddVehicle, UpdateVehicle, DeleteVehicle, UpdateDriverStatus, UpdateDriverLocation, GetDriverLocation, FindNearestDrivers, ListDrivers, ListAvailableDrivers, AssignDriverToRide, GetDriver, GetDriverStatusHistory, SetDriverRating |
| Ali | User + Gateway/Auth | 14 | RegisterUser, LoginUser, LogoutUser, ValidateSession, GetUserProfile, GetUserByEmail, ListUsers, UpdateUserProfile, ChangePassword, ResetPassword, VerifyUserEmail, ResendVerification, DeactivateUser, UpdateUserRole |
| Dias | Payment + Notification | 19 | CreatePayment, GetPaymentByRide, GetPayment, ListUserPayments, ListFailedPayments, GetPaymentReceipt, ValidatePaymentMethod, ProcessPayment, CreateRefundRequest, RefundPayment, SendEmailNotification, SendRideReceipt, GetNotificationHistory, ListUnreadNotifications, CountUnreadNotifications, MarkNotificationAsRead, MarkAllNotificationsAsRead, ResendNotification, DeleteNotification |
| Nurassyl | Ride | 14 | CreateRide, EstimateRidePrice, GetRide, ListUserRides, ListActiveRides, ListDriverRides, GetRideHistory, ScheduleRide, AcceptRide, RejectRide, UpdateRideStatus, CancelRide, CompleteRide, RateRide |

## NATS pub/sub цепочки

1. `user-service` ▶ `user.registered` ─▶ `notification-service` (welcome email)
2. `ride-service` ▶ `ride.requested` ─▶ `driver-service` подбирает водителя ▶ `driver.assigned` ─▶ `ride-service` ставит `driver_assigned` (через FOR UPDATE транзакцию)
3. `ride-service` ▶ `ride.completed` ─▶ `payment-service` создаёт + processes payment ▶ `payment.succeeded` ─▶ `notification-service` (receipt)
4. `ride-service` ▶ `ride.cancelled` ─▶ `payment-service` делает refund если платёж был

## Транзакции

| Сервис | Метод | Что атомарно |
|--------|-------|-------------|
| user | `CreateWithVerification` | INSERT users + INSERT verification_tokens |
| user | `VerifyEmail` | UPDATE users + DELETE verification_tokens |
| driver | `UpdateStatus` | UPDATE drivers + INSERT driver_status_history |
| ride | `CreateWithHistory` | INSERT rides + INSERT ride_status_history |
| ride | `UpdateStatusWithHistory` | SELECT FOR UPDATE + FSM check + UPDATE rides + INSERT ride_status_history |
| ride | `AssignDriver` | SELECT FOR UPDATE + UPDATE rides + INSERT ride_status_history |
| payment | `CreateWithEvent` | INSERT payments + INSERT payment_events |
| payment | `UpdateStatusWithEvent` | SELECT FOR UPDATE + FSM check + UPDATE payments + INSERT payment_events |

## Команды для защиты

```bash
make docker-up          # запуск всего стека
make migrate-up         # повторно прогнать миграции (миграции запускаются автоматически)
make demo               # пройти полный flow одной командой
make test               # запустить unit-тесты
make web-test           # собрать и проверить frontend
make test-integration   # запустить интеграционный тест (стек должен быть поднят)
make docker-logs        # смотреть логи
make docker-down        # остановить
```
