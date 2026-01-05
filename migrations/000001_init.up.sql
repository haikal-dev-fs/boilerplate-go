-- 000001_init.up.sql

CREATE TABLE organizations (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    code            TEXT UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vehicles (
    id                  BIGSERIAL PRIMARY KEY,
    organization_id     BIGINT REFERENCES organizations(id),
    plate_number        TEXT NOT NULL,
    name                TEXT,
    vin                 TEXT,
    vehicle_type        TEXT,
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    metadata            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ux_vehicles_org_plate
    ON vehicles (organization_id, plate_number);

CREATE TABLE data_sources (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    code        TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE devices (
    id                  BIGSERIAL PRIMARY KEY,
    data_source_id      BIGINT REFERENCES data_sources(id),
    external_id         TEXT NOT NULL,
    sim_number          TEXT,
    model               TEXT,
    protocol            TEXT,
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    metadata            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (data_source_id, external_id)
);

CREATE TABLE vehicle_devices (
    id              BIGSERIAL PRIMARY KEY,
    vehicle_id      BIGINT NOT NULL REFERENCES vehicles(id),
    device_id       BIGINT NOT NULL REFERENCES devices(id),
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unassigned_at   TIMESTAMPTZ
);

CREATE INDEX idx_vehicle_devices_active
    ON vehicle_devices (vehicle_id)
    WHERE active = TRUE;

CREATE TABLE position_log (
    id              BIGSERIAL PRIMARY KEY,
    vehicle_id      BIGINT NOT NULL REFERENCES vehicles(id),
    device_id       BIGINT NOT NULL REFERENCES devices(id),
    ts              TIMESTAMPTZ NOT NULL,
    lat             DOUBLE PRECISION NOT NULL,
    lon             DOUBLE PRECISION NOT NULL,
    speed_kph       NUMERIC(6,2),
    heading_deg     NUMERIC(6,2),
    altitude_m      NUMERIC(8,2),
    ignition_on     BOOLEAN,
    odometer_km     NUMERIC(10,2),
    raw_payload     JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_device_ts UNIQUE (device_id, ts)
);

CREATE INDEX idx_position_vehicle_ts_desc
    ON position_log (vehicle_id, ts DESC);

CREATE INDEX idx_position_vehicle_ts
    ON position_log (vehicle_id, ts);

CREATE INDEX idx_position_ts
    ON position_log (ts);

CREATE TABLE vehicle_current_position (
    vehicle_id      BIGINT PRIMARY KEY REFERENCES vehicles(id),
    device_id       BIGINT REFERENCES devices(id),
    ts              TIMESTAMPTZ NOT NULL,
    lat             DOUBLE PRECISION NOT NULL,
    lon             DOUBLE PRECISION NOT NULL,
    speed_kph       NUMERIC(6,2),
    heading_deg     NUMERIC(6,2),
    ignition_on     BOOLEAN,
    odometer_km     NUMERIC(10,2),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE alert_types (
    id              BIGSERIAL PRIMARY KEY,
    code            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    default_severity TEXT NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE alerts (
    id              BIGSERIAL PRIMARY KEY,
    vehicle_id      BIGINT NOT NULL REFERENCES vehicles(id),
    device_id       BIGINT REFERENCES devices(id),
    alert_type_id   BIGINT NOT NULL REFERENCES alert_types(id),
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    status          TEXT NOT NULL,
    message         TEXT,
    payload         JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ
);

CREATE INDEX idx_alerts_vehicle_started
    ON alerts (vehicle_id, started_at DESC);

CREATE INDEX idx_alerts_status
    ON alerts (status);

CREATE TABLE trips (
    id                  BIGSERIAL PRIMARY KEY,
    vehicle_id          BIGINT NOT NULL REFERENCES vehicles(id),
    start_ts            TIMESTAMPTZ NOT NULL,
    end_ts              TIMESTAMPTZ NOT NULL,
    start_lat           DOUBLE PRECISION,
    start_lon           DOUBLE PRECISION,
    end_lat             DOUBLE PRECISION,
    end_lon             DOUBLE PRECISION,
    distance_km         NUMERIC(10,2),
    duration_seconds    INTEGER,
    max_speed_kph       NUMERIC(6,2),
    avg_speed_kph       NUMERIC(6,2),
    metadata            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trips_vehicle_start
    ON trips (vehicle_id, start_ts DESC);
