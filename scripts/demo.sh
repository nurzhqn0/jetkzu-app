#!/usr/bin/env bash
# Walks through the full JetKZu demo flow against a running gateway.
# Usage: ./scripts/demo.sh
set -euo pipefail

API=${API:-http://localhost:8080}
PASS_EMAIL=${PASS_EMAIL:-passenger_$(date +%s)@example.kz}
DRIVER_EMAIL=${DRIVER_EMAIL:-driver_$(date +%s)@example.kz}
PASSWORD=${PASSWORD:-Password123}

say() { printf "\n\033[1;36m▶ %s\033[0m\n" "$*"; }

say "Health check"
curl -s "$API/health" | head -c 200; echo

say "Register passenger"
PASS_REG=$(curl -s -X POST "$API/api/auth/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$PASS_EMAIL\",\"password\":\"$PASSWORD\",\"full_name\":\"Demo Passenger\",\"role\":\"passenger\"}")
echo "$PASS_REG"
PASS_ID=$(echo "$PASS_REG" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)

say "Login passenger"
PASS_LOGIN=$(curl -s -X POST "$API/api/auth/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$PASS_EMAIL\",\"password\":\"$PASSWORD\"}")
PASS_TOKEN=$(echo "$PASS_LOGIN" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
echo "Passenger token acquired"

say "Register driver user"
DRV_REG=$(curl -s -X POST "$API/api/auth/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$DRIVER_EMAIL\",\"password\":\"$PASSWORD\",\"full_name\":\"Demo Driver\",\"role\":\"driver\"}")
DRV_USER_ID=$(echo "$DRV_REG" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)

say "Login driver"
DRV_LOGIN=$(curl -s -X POST "$API/api/auth/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$DRIVER_EMAIL\",\"password\":\"$PASSWORD\"}")
DRV_TOKEN=$(echo "$DRV_LOGIN" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')

say "Register driver profile + vehicle"
DRV_REG_RES=$(curl -s -X POST "$API/api/drivers/register" -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $DRV_TOKEN" \
  -d "{\"user_id\":\"$DRV_USER_ID\",\"license_number\":\"KZ-AA-12345\"}")
echo "$DRV_REG_RES"
DRIVER_ID=$(echo "$DRV_REG_RES" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)

curl -s -X POST "$API/api/drivers/vehicle" -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $DRV_TOKEN" \
  -d "{\"driver_id\":\"$DRIVER_ID\",\"plate_number\":\"123ABC01\",\"make\":\"Toyota\",\"model\":\"Camry\",\"year\":2022,\"color\":\"white\"}" | head -c 300; echo

say "Driver goes online + sets location near Astana"
curl -s -X PATCH "$API/api/drivers/status" -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $DRV_TOKEN" \
  -d "{\"driver_id\":\"$DRIVER_ID\",\"status\":\"online\"}" | head -c 200; echo

curl -s -X PATCH "$API/api/drivers/location" -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $DRV_TOKEN" \
  -d "{\"driver_id\":\"$DRIVER_ID\",\"latitude\":51.169392,\"longitude\":71.449074}" | head -c 200; echo

say "Estimate ride"
curl -s -X POST "$API/api/rides/estimate" -H 'Content-Type: application/json' \
  -d '{"pickup_lat":51.169392,"pickup_lng":71.449074,"dropoff_lat":51.180000,"dropoff_lng":71.460000}'
echo

say "Create ride (auto-publishes ride.requested → driver auto-assigned via NATS)"
RIDE_RES=$(curl -s -X POST "$API/api/rides" -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $PASS_TOKEN" \
  -d "{\"passenger_id\":\"$PASS_ID\",\"pickup_lat\":51.169392,\"pickup_lng\":71.449074,\"dropoff_lat\":51.180000,\"dropoff_lng\":71.460000}")
echo "$RIDE_RES"
RIDE_ID=$(echo "$RIDE_RES" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)

say "Sleep 2s, then fetch ride (should now be driver_assigned)"
sleep 2
curl -s "$API/api/rides/$RIDE_ID" -H "Authorization: Bearer $PASS_TOKEN"; echo

say "Driver arrived → in_progress → completed"
curl -s -X PATCH "$API/api/rides/$RIDE_ID/status" -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $DRV_TOKEN" -d '{"status":"driver_arrived"}' > /dev/null
curl -s -X PATCH "$API/api/rides/$RIDE_ID/status" -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $DRV_TOKEN" -d '{"status":"in_progress"}' > /dev/null
curl -s -X POST "$API/api/rides/$RIDE_ID/complete" -H "Authorization: Bearer $DRV_TOKEN"; echo

say "Sleep 2s, then check payment"
sleep 2
curl -s "$API/api/rides/$RIDE_ID/payment" -H "Authorization: Bearer $PASS_TOKEN"; echo

say "Check passenger notifications"
curl -s "$API/api/notifications/my" -H "Authorization: Bearer $PASS_TOKEN" | head -c 500; echo

say "Demo complete."
