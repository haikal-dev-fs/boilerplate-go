-- Tabel organisasi (customer)
CREATE TABLE IF NOT EXISTS organizations (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    code        TEXT UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    active      BOOLEAN NOT NULL DEFAULT TRUE
);

-- Tipe user:
-- - SUPER_ADMIN : user internal kamu
-- - ORG_USER    : user milik suatu organization (admin/user)
CREATE TABLE IF NOT EXISTS users (
    id               BIGSERIAL PRIMARY KEY,
    email            TEXT NOT NULL UNIQUE,
    password_hash    TEXT NOT NULL,
    full_name        TEXT,
    user_type        TEXT NOT NULL,  -- 'SUPER_ADMIN' atau 'ORG_USER'
    organization_id  BIGINT REFERENCES organizations(id),
    org_role         TEXT,           -- 'ADMIN' atau 'USER' (kalau ORG_USER)
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_user_type_check CHECK (user_type IN ('SUPER_ADMIN', 'ORG_USER')),
    CONSTRAINT users_org_role_check CHECK (
        (user_type = 'SUPER_ADMIN' AND org_role IS NULL AND organization_id IS NULL)
        OR
        (user_type = 'ORG_USER' AND org_role IN ('ADMIN', 'USER') AND organization_id IS NOT NULL)
    )
);
