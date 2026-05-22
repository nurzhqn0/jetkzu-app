CREATE TABLE IF NOT EXISTS rides (
    id            UUID PRIMARY KEY,
    passenger_id  UUID NOT NULL,
    driver_id     UUID,
    pickup_lat    DOUBLE PRECISION NOT NULL,
    pickup_lng    DOUBLE PRECISION NOT NULL,
    dropoff_lat   DOUBLE PRECISION NOT NULL,
    dropoff_lng   DOUBLE PRECISION NOT NULL,
    status        TEXT NOT NULL,
    price         NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rides_passenger ON rides(passenger_id);
CREATE INDEX IF NOT EXISTS idx_rides_driver ON rides(driver_id);
CREATE INDEX IF NOT EXISTS idx_rides_status ON rides(status);

CREATE TABLE IF NOT EXISTS ride_status_history (
    id          BIGSERIAL PRIMARY KEY,
    ride_id     UUID NOT NULL REFERENCES rides(id) ON DELETE CASCADE,
    status      TEXT NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ride_price_estimations (
    id           BIGSERIAL PRIMARY KEY,
    ride_id      UUID NOT NULL REFERENCES rides(id) ON DELETE CASCADE,
    price        NUMERIC(12,2) NOT NULL,
    distance_km  DOUBLE PRECISION NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
