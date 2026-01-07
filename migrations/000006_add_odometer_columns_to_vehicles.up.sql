-- 000006_add_odometer_columns_to_vehicles.up.sql

ALTER TABLE vehicles
ADD COLUMN IF NOT EXISTS odometer_base_km DECIMAL(12,3) DEFAULT 0.0,
ADD COLUMN IF NOT EXISTS device_distance_base_km DECIMAL(12,3) DEFAULT 0.0,
ADD COLUMN IF NOT EXISTS current_odometer_km DECIMAL(12,3) DEFAULT 0.0,
ADD COLUMN IF NOT EXISTS odometer_source TEXT DEFAULT 'DEVICE_GPS';

-- Add comments for clarity
COMMENT ON COLUMN vehicles.odometer_base_km IS 'Base odometer value in kilometers';
COMMENT ON COLUMN vehicles.device_distance_base_km IS 'Device distance base value in kilometers for calculation';
COMMENT ON COLUMN vehicles.current_odometer_km IS 'Current calculated odometer value in kilometers';
COMMENT ON COLUMN vehicles.odometer_source IS 'Source of odometer calculation (DEVICE_GPS, MANUAL, etc.)';

-- Create index for odometer source for efficient querying
CREATE INDEX IF NOT EXISTS idx_vehicles_odometer_source ON vehicles(odometer_source);