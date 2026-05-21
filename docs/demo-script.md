# Demo script (5–7 минут защиты)

## 0. Подготовка (до начала)

```bash
cp .env.example .env
make docker-up        # билд может занять ~3-5 минут в первый раз
docker compose ps     # все 9 сервисов в "running"
docker compose logs migrate | tail   # все 5 БД мигрированы
```

## 1. Health (10 сек)

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

## 2. Регистрация пассажира

```bash
curl -X POST http://localhost:8080/api/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"aigerim@kz","password":"Password123","full_name":"Aigerim","role":"passenger"}'
```

Ответ содержит `user.id` и `verification_token`. NATS publish `user.registered` → notification-service пишет welcome email в `notifications` (или реально отправляет, если есть SMTP).

## 3. Логин пассажира → JWT

```bash
PASS_TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"aigerim@kz","password":"Password123"}' | jq -r .access_token)
```

## 4. Регистрация водителя

Тот же endpoint, `role=driver`. Получите `DRV_USER_ID` и `DRV_TOKEN`.

```bash
curl -X POST http://localhost:8080/api/drivers/register \
  -H "Authorization: Bearer $DRV_TOKEN" -H 'Content-Type: application/json' \
  -d '{"user_id":"'"$DRV_USER_ID"'","license_number":"KZ-A-001"}'
```

## 5. Машина + статус online + координаты

```bash
curl -X POST http://localhost:8080/api/drivers/vehicle -H "Authorization: Bearer $DRV_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"driver_id":"'"$DRIVER_ID"'","plate_number":"123ABC","make":"Toyota","model":"Camry","year":2022,"color":"white"}'

curl -X PATCH http://localhost:8080/api/drivers/status  -H "Authorization: Bearer $DRV_TOKEN" \
  -H 'Content-Type: application/json' -d '{"driver_id":"'"$DRIVER_ID"'","status":"online"}'

curl -X PATCH http://localhost:8080/api/drivers/location -H "Authorization: Bearer $DRV_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"driver_id":"'"$DRIVER_ID"'","latitude":51.169392,"longitude":71.449074}'
```

## 6. Оценка и создание поездки

```bash
curl -X POST http://localhost:8080/api/rides/estimate -H 'Content-Type: application/json' \
  -d '{"pickup_lat":51.169392,"pickup_lng":71.449074,"dropoff_lat":51.180,"dropoff_lng":71.460}'

curl -X POST http://localhost:8080/api/rides -H "Authorization: Bearer $PASS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"passenger_id":"'"$PASS_ID"'","pickup_lat":51.169392,"pickup_lng":71.449074,"dropoff_lat":51.180,"dropoff_lng":71.460}'
```

В этот момент:
- ride-service создаёт ride + ride_status_history в одной транзакции;
- публикует `ride.requested` в NATS;
- driver-service подбирает ближайшего online водителя в Redis GEO, помечает busy, публикует `driver.assigned`;
- ride-service подписан на `driver.assigned` → транзакционно обновляет ride.status = `driver_assigned`.

Через 1–2 секунды:

```bash
curl -s http://localhost:8080/api/rides/$RIDE_ID -H "Authorization: Bearer $PASS_TOKEN"
# → ride.status = driver_assigned, driver_id заполнен
```

## 7. Прохождение поездки

```bash
curl -X PATCH http://localhost:8080/api/rides/$RIDE_ID/status -H "Authorization: Bearer $DRV_TOKEN" \
  -H 'Content-Type: application/json' -d '{"status":"driver_arrived"}'
curl -X PATCH http://localhost:8080/api/rides/$RIDE_ID/status -H "Authorization: Bearer $DRV_TOKEN" \
  -H 'Content-Type: application/json' -d '{"status":"in_progress"}'
curl -X POST http://localhost:8080/api/rides/$RIDE_ID/complete -H "Authorization: Bearer $DRV_TOKEN"
```

`ride.completed` → payment-service слышит → создаёт payment (pending) и сразу processes (succeeded), две транзакции в payments + payment_events.

## 8. Платёж

```bash
curl http://localhost:8080/api/rides/$RIDE_ID/payment -H "Authorization: Bearer $PASS_TOKEN"
# → payment.status = succeeded
```

## 9. Notifications

```bash
curl http://localhost:8080/api/notifications/my -H "Authorization: Bearer $PASS_TOKEN"
# вы увидите welcome + ride requested + ride completed + payment receipt
```

## 10. Observability

* **Prometheus** http://localhost:9090 — query `rate(jetkzu_grpc_requests_total[1m])`
* **Grafana** http://localhost:3000 — дашборд "JetKZu Overview" (логин `admin/admin`)
* **NATS** http://localhost:8222 — топики, подписчики, in-flight messages
* **Logs**: `docker compose logs ride-service | grep correlation_id` — увидите один и тот же id у HTTP запроса в gateway и у gRPC handler'а

## 11. Тесты

```bash
make test                # unit
make test-integration    # E2E (gateway должен быть поднят)
```

## 12. Финальная сводка по критериям

Скажите комиссии:

* Clean architecture — *см. слои `domain/usecase/repository/infrastructure` в каждом из 5 сервисов*;
* gRPC — *61 методов в 5 сервисах, each team member owns at least 12 methods; см. `docs/grading-checklist.md`*;
* NATS — *показал реальную цепочку `ride.requested → driver.assigned → ride.completed → payment.succeeded`*;
* DB / Redis / migrations / transactions — *миграции прогнал контейнер `migrate`, транзакции в каждом из 5 сервисов*;
* SMTP — *Gmail если есть env, иначе MockSender; запись в `notifications`*;
* Tests — *unit + integration test, оба зелёные*;
* Observability — *Prometheus scrape всех 6 портов, Grafana дашборд, correlation_id end-to-end*.

Готово.
