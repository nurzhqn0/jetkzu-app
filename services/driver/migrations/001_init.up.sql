CREATE TABLE IF NOT EXISTS drivers (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    license_number  TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'offline',
    latitude        DOUBLE PRECISION NOT NULL DEFAULT 0,
    longitude       DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_drivers_user ON drivers(user_id);
CREATE INDEX IF NOT EXISTS idx_drivers_status ON drivers(status);

CREATE TABLE IF NOT EXISTS vehicles (
    id            UUID PRIMARY KEY,
    driver_id     UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    plate_number  TEXT NOT NULL,
    make          TEXT NOT NULL,
    model         TEXT NOT NULL,
    year          INT NOT NULL,
    color         TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS driver_status_history (
    id          BIGSERIAL PRIMARY KEY,
    driver_id   UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    status      TEXT NOT NULL,
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS driver_assignments (
    id           BIGSERIAL PRIMARY KEY,
    driver_id    UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    ride_id      UUID NOT NULL,
    assigned_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
