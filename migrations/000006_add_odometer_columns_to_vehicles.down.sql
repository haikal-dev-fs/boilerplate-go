-- 000006_add_odometer_columns_to_vehicles.down.sql

-- Remove the index first
DROP INDEX IF EXISTS idx_vehicles_odometer_source;

-- Remove the columns
ALTER TABLE vehicles
DROP COLUMN IF EXISTS odometer_source,
DROP COLUMN IF EXISTS current_odometer_km,
DROP COLUMN IF EXISTS device_distance_base_km,
DROP COLUMN IF EXISTS odometer_base_km;