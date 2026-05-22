# API examples (curl + ожидаемые ответы)

Подразумевается gateway по адресу `http://localhost:8080`.

## Регистрация пассажира

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@b.kz","password":"Password123","full_name":"Aigerim","phone":"+77011112233","role":"passenger"}'
```

```json
{
  "user": {
    "id": "9d92...","email":"a@b.kz","full_name":"Aigerim","phone":"+77011112233",
    "role":"passenger","email_verified":false,"created_at":"...","updated_at":"..."
  },
  "verification_token": "e3a..."
}
```

## Логин

```bash
curl -X POST http://localhost:8080/api/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"a@b.kz","password":"Password123"}'
```

```json
{
  "access_token": "eyJhbGciOi...",
  "expires_at": "2026-05-18T20:11:00Z",
  "user": { "id": "9d92...","email":"a@b.kz","role":"passenger","..." }
}
```

## Создание поездки

```bash
curl -X POST http://localhost:8080/api/rides \
  -H 'Authorization: Bearer <JWT>' -H 'Content-Type: application/json' \
  -d '{"passenger_id":"9d92...","pickup_lat":51.169,"pickup_lng":71.449,"dropoff_lat":51.180,"dropoff_lng":71.460}'
```

```json
{
  "ride": {
    "id":"7be4...","passenger_id":"9d92...","driver_id":"",
    "pickup_lat":51.169,"pickup_lng":71.449,"dropoff_lat":51.180,"dropoff_lng":71.460,
    "status":"requested","price":820.40,
    "created_at":"...","updated_at":"..."
  }
}
```

## Оценка цены

```bash
curl -X POST http://localhost:8080/api/rides/estimate -H 'Content-Type: application/json' \
  -d '{"pickup_lat":51.169,"pickup_lng":71.449,"dropoff_lat":51.180,"dropoff_lng":71.460}'
```

```json
{"price": 820.4, "distance_km": 2.67}
```

## Ближайшие водители

```bash
curl 'http://localhost:8080/api/drivers/nearest?lat=51.169&lng=71.449&radius_km=5&limit=5'
```

```json
{
  "drivers":[{"driver_id":"...","latitude":51.169,"longitude":71.449,"distance_km":0.0}]
}
```

## Платёж по поездке

```bash
curl http://localhost:8080/api/rides/7be4.../payment -H 'Authorization: Bearer <JWT>'
```

```json
{
  "payment": {
    "id":"...","ride_id":"7be4...","user_id":"9d92...",
    "amount":820.4,"currency":"KZT","status":"succeeded","method":"card",
    "created_at":"...","updated_at":"..."
  }
}
```

## Ошибки

```json
{"error":"invalid email or password","correlation_id":"6f2c..."}
```

`correlation_id` совпадает с `X-Correlation-ID` в response headers и можно по нему искать запрос в логах сервисов.
