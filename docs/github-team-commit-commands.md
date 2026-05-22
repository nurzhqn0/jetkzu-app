# GitHub team commit commands

Run these commands from the real Git repository root. Replace each `*_GITHUB_EMAIL`
with the member's GitHub email or GitHub noreply email so the commits attach to
the correct GitHub profile.

The project has a few shared files, especially `gateway/internal/router/router.go`,
`gateway/internal/handlers/handlers.go`, `web/src/pages/DashboardPage.tsx`, and
top-level docs. For those files, use `git add -p` when you need to stage only the
member-owned hunks.

Commit order: Nurzhan, Ali, Dias, Nurassyl.

## 0. Verify before committing

```bash
go test ./...
docker compose config --quiet
docker compose up -d --build
curl -s http://127.0.0.1:3000/health
```

## 1. Nurzhan

Nurzhan owns DriverService, vehicle management, Redis GEO driver location, and
driver dashboard actions.

```bash
git add \
  proto/driver \
  gen/go/driver \
  services/driver \
  pkg/redis

git commit \
  --author="Nurzhan <NURZHAN_GITHUB_EMAIL>" \
  -m "feat(driver): expand driver and vehicle grpc endpoints"
```

```bash
git add -p \
  gateway/internal/router/router.go \
  gateway/internal/handlers/handlers.go

git commit \
  --author="Nurzhan <NURZHAN_GITHUB_EMAIL>" \
  -m "feat(gateway): expose driver management endpoints"
```

```bash
git add -p \
  web/src/pages/DashboardPage.tsx \
  web/src/styles.css

git commit \
  --author="Nurzhan <NURZHAN_GITHUB_EMAIL>" \
  -m "feat(web): add driver dashboard"
```

## 2. Ali

Ali owns UserService, auth/session flows, JWT, validation, and user-facing auth UI.

```bash
git add \
  proto/user \
  gen/go/user \
  services/user \
  pkg/jwt \
  pkg/validator

git add -p \
  gateway/internal/router/router.go \
  gateway/internal/handlers/handlers.go \
  gateway/internal/middleware \
  web/src/pages/AuthPage.tsx \
  web/src/components \
  web/src/lib

git commit \
  --author="Ali <ALI_GITHUB_EMAIL>" \
  -m "feat(user): expand auth and profile grpc endpoints"
```

```bash
git add -p \
  gateway/internal/router/router.go \
  gateway/internal/handlers/handlers.go \
  gateway/internal/middleware

git commit \
  --author="Ali <ALI_GITHUB_EMAIL>" \
  -m "feat(gateway): expose user endpoints and frontend cors"
```

```bash
git add \
  web/src/pages/AuthPage.tsx \
  web/src/components \
  web/src/lib

git add -p \
  web/src/App.tsx \
  web/src/main.tsx \
  web/src/styles.css

git commit \
  --author="Ali <ALI_GITHUB_EMAIL>" \
  -m "feat(web): add auth and profile screens"
```

## 3. Dias

Dias owns PaymentService, NotificationService, SMTP, receipts, Docker/frontend
service packaging, final demo checks, and grading documentation.

```bash
git add \
  proto/payment \
  proto/notification \
  gen/go/payment \
  gen/go/notification \
  services/payment \
  services/notification

git add -p \
  pkg/natsbus \
  gateway/internal/router/router.go \
  gateway/internal/handlers/handlers.go

git commit \
  --author="Dias <DIAS_GITHUB_EMAIL>" \
  -m "feat(payment-notification): expand billing and notification endpoints"
```

```bash
git add \
  Dockerfile \
  docker-compose.yml \
  .dockerignore \
  Makefile \
  web/Dockerfile \
  web/.dockerignore \
  web/nginx.conf

git commit \
  --author="Dias <DIAS_GITHUB_EMAIL>" \
  -m "feat(compose): add frontend service"
```

```bash
git add \
  tests \
  docs \
  README.md \
  scripts/demo.sh

git commit \
  --author="Dias <DIAS_GITHUB_EMAIL>" \
  -m "test: verify final demo flow"
```

## 4. Nurassyl

Nurassyl owns RideService, ride lifecycle, ride history, scheduling, rating, and
passenger ride dashboard actions.

```bash
git add \
  proto/ride \
  gen/go/ride \
  services/ride

git add -p \
  pkg/natsbus

git commit \
  --author="Nurassyl <NURASSYL_GITHUB_EMAIL>" \
  -m "feat(ride): expand ride lifecycle grpc endpoints"
```

```bash
git add -p \
  gateway/internal/router/router.go \
  gateway/internal/handlers/handlers.go

git commit \
  --author="Nurassyl <NURASSYL_GITHUB_EMAIL>" \
  -m "feat(gateway): expose ride history and lifecycle endpoints"
```

```bash
git add -p \
  web/src/pages/DashboardPage.tsx \
  web/src/styles.css

git commit \
  --author="Nurassyl <NURASSYL_GITHUB_EMAIL>" \
  -m "feat(web): add passenger ride flow"
```

## 5. Verify after committing

```bash
go test ./...
docker compose config --quiet
docker compose up -d --build
curl -s http://127.0.0.1:3000/health
git log --oneline --format="%h %an %s" -12
```
