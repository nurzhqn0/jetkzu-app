# VPS Deployment

This deploys JetKZu with Docker Compose on a single VPS. The public entrypoint is the frontend Nginx container; it proxies `/api` and `/health` to the internal API Gateway.

## Server Prerequisites

- Ubuntu 22.04/24.04 or another Linux VPS with Docker Engine and Docker Compose v2.
- Open inbound TCP `80` on the VPS firewall.
- Open inbound TCP `443` only if you add a host-level TLS reverse proxy.
- Keep Postgres, Redis, NATS, Prometheus, Grafana, and service gRPC ports closed to the public internet.

## First Deploy

```bash
git clone <repo-url> /opt/jetkzu
cd /opt/jetkzu
cp .env.vps.example .env.vps
nano .env.vps
docker compose --env-file .env.vps -f docker-compose.vps.yml up -d --build
docker compose --env-file .env.vps -f docker-compose.vps.yml ps
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1/health
```

Generate strong secrets before editing `.env.vps`:

```bash
openssl rand -base64 48
```

Use URL-safe characters for `POSTGRES_PASSWORD` because it is embedded in internal Postgres connection URLs.

## Updating

```bash
cd /opt/jetkzu
git pull
docker compose --env-file .env.vps -f docker-compose.vps.yml up -d --build
docker compose --env-file .env.vps -f docker-compose.vps.yml logs -f --tail=200
```

The `migrate` service runs automatically before application services start.

## Public HTTP Mode

The default `.env.vps.example` exposes the frontend directly on `0.0.0.0:80`:

```dotenv
FRONTEND_BIND_ADDR=0.0.0.0
FRONTEND_HOST_PORT=80
```

Use this for quick VPS deployment by IP address or when TLS is terminated by an external load balancer.

## Host Nginx + HTTPS Mode

If you want TLS on the VPS itself, bind the frontend only to localhost:

```dotenv
FRONTEND_BIND_ADDR=127.0.0.1
FRONTEND_HOST_PORT=3000
```

Then configure host Nginx or Caddy to proxy your domain to `http://127.0.0.1:3000`. With Nginx, the minimal server block is:

```nginx
server {
  server_name example.com;

  location / {
    proxy_pass http://127.0.0.1:3000;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}
```

After that, issue certificates with Certbot or your preferred TLS tooling.

## Operations

Grafana and Prometheus are bound to localhost by default:

```bash
ssh -L 3001:127.0.0.1:3001 -L 9090:127.0.0.1:9090 user@your-vps
```

Then open:

- Grafana: `http://127.0.0.1:3001`
- Prometheus: `http://127.0.0.1:9090`

Useful commands:

```bash
docker compose --env-file .env.vps -f docker-compose.vps.yml logs -f --tail=200
docker compose --env-file .env.vps -f docker-compose.vps.yml restart api-gateway
docker compose --env-file .env.vps -f docker-compose.vps.yml down
```
