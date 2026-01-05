-- 000001_init.down.sql

DROP INDEX IF EXISTS idx_trips_vehicle_start;
DROP TABLE IF EXISTS trips;

DROP INDEX IF EXISTS idx_alerts_status;
DROP INDEX IF EXISTS idx_alerts_vehicle_started;
DROP TABLE IF EXISTS alerts;

DROP TABLE IF EXISTS alert_types;

DROP TABLE IF EXISTS vehicle_current_position;

DROP INDEX IF EXISTS idx_position_ts;
DROP INDEX IF EXISTS idx_position_vehicle_ts;
DROP INDEX IF EXISTS idx_position_vehicle_ts_desc;
DROP TABLE IF EXISTS position_log;

DROP INDEX IF EXISTS idx_vehicle_devices_active;
DROP TABLE IF EXISTS vehicle_devices;

DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS data_sources;

DROP INDEX IF EXISTS ux_vehicles_org_plate;
DROP TABLE IF EXISTS vehicles;

DROP TABLE IF EXISTS organizations;
