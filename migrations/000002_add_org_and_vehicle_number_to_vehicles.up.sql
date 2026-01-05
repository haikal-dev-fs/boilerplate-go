ALTER TABLE vehicles
ADD COLUMN IF NOT EXISTS organization_id BIGINT,
ADD COLUMN IF NOT EXISTS vehicle_number TEXT;

-- optional: kalau kamu mau semua vehicle wajib punya org:
ALTER TABLE vehicles ALTER COLUMN organization_id SET NOT NULL;