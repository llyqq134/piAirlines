-- Roles (admin/agent) + refresh tokens
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS role VARCHAR(16) NOT NULL DEFAULT 'agent';

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id           BIGSERIAL PRIMARY KEY,
  user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash   BYTEA NOT NULL UNIQUE,
  expires_at   TIMESTAMPTZ NOT NULL,
  revoked_at   TIMESTAMPTZ NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_expires_idx ON refresh_tokens(expires_at);

-- Reports payload storage
ALTER TABLE reports
  ADD COLUMN IF NOT EXISTS payload JSONB NULL;

-- Seat units (concrete seat per booking)
CREATE TABLE IF NOT EXISTS seat_units (
  id        BIGSERIAL PRIMARY KEY,
  flight_id INT NOT NULL REFERENCES flights(id) ON DELETE CASCADE,
  seat_no   VARCHAR(8) NOT NULL,
  status    VARCHAR(16) NOT NULL DEFAULT 'available', -- available|held|sold
  booking_id INT NULL UNIQUE REFERENCES bookings(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (flight_id, seat_no)
);

CREATE INDEX IF NOT EXISTS seat_units_flight_status_idx ON seat_units(flight_id, status);

