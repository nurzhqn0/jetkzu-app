#!/bin/sh
set -e

for db in jetkzu_users jetkzu_drivers jetkzu_rides jetkzu_payments jetkzu_notifications; do
  echo "Creating database $db";
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d postgres <<-EOSQL
    CREATE DATABASE $db;
EOSQL
done
