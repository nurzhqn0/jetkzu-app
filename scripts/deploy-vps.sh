#!/usr/bin/env bash
set -euo pipefail

HOST="${1:?usage: scripts/deploy-vps.sh user@server [/opt/jetkzu]}"
APP_DIR="${2:-/opt/jetkzu}"

rsync -az --delete \
  --exclude ".git" \
  --exclude ".env" \
  --exclude ".idea" \
  --exclude "node_modules" \
  --exclude "web/node_modules" \
  --exclude "web/dist" \
  ./ "$HOST:$APP_DIR/"

ssh "$HOST" "cd '$APP_DIR' && test -f deploy/.env.production || cp deploy/production.env.example deploy/.env.production"
ssh "$HOST" "cd '$APP_DIR/deploy' && docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build"
ssh "$HOST" "cd '$APP_DIR/deploy' && docker compose --env-file .env.production -f docker-compose.prod.yml ps"
