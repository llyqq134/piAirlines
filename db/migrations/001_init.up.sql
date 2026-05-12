-- Core domain (from the provided ER diagram)

CREATE TABLE IF NOT EXISTS airlines (
  id   SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  code VARCHAR(16) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS tariffs (
  id    SERIAL PRIMARY KEY,
  name  TEXT NOT NULL,
  price NUMERIC(12,2) NOT NULL CHECK (price >= 0)
);

CREATE TABLE IF NOT EXISTS flights (
  id            SERIAL PRIMARY KEY,
  flight_number VARCHAR(32) NOT NULL UNIQUE,
  airline_id    INT NOT NULL REFERENCES airlines(id) ON DELETE RESTRICT,
  tariff_id     INT NOT NULL REFERENCES tariffs(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS passengers (
  id        SERIAL PRIMARY KEY,
  full_name TEXT NOT NULL,
  passport  VARCHAR(32) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS bookings (
  id           SERIAL PRIMARY KEY,
  passenger_id INT NOT NULL REFERENCES passengers(id) ON DELETE RESTRICT,
  flight_id    INT NOT NULL REFERENCES flights(id) ON DELETE RESTRICT,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS seats (
  id              SERIAL PRIMARY KEY,
  flight_id       INT NOT NULL UNIQUE REFERENCES flights(id) ON DELETE CASCADE,
  total_seats     INT NOT NULL CHECK (total_seats >= 0),
  available_seats INT NOT NULL CHECK (available_seats >= 0 AND available_seats <= total_seats)
);

CREATE TABLE IF NOT EXISTS sales (
  id         SERIAL PRIMARY KEY,
  booking_id INT NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE RESTRICT,
  amount     NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS refunds (
  id         SERIAL PRIMARY KEY,
  sale_id    INT NOT NULL REFERENCES sales(id) ON DELETE RESTRICT,
  amount     NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notifications (
  id           SERIAL PRIMARY KEY,
  passenger_id INT NOT NULL REFERENCES passengers(id) ON DELETE CASCADE,
  message      TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS reports (
  id          SERIAL PRIMARY KEY,
  report_type VARCHAR(64) NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Added for authentication (JWT + OAuth2)

CREATE TABLE IF NOT EXISTS users (
  id            BIGSERIAL PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_accounts (
  id               BIGSERIAL PRIMARY KEY,
  user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider         VARCHAR(32) NOT NULL,
  provider_user_id TEXT NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (provider, provider_user_id)
);

