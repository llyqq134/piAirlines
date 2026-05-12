-- Demo seed data (safe to re-run).
-- Assumes migrations 001 + 002 + 003 already applied.

BEGIN;

-- Airlines
INSERT INTO airlines (name, code) VALUES
  ('Aeroflot', 'SU'),
  ('S7 Airlines', 'S7'),
  ('Ural Airlines', 'U6')
ON CONFLICT (code) DO NOTHING;

-- Tariffs
INSERT INTO tariffs (name, price) VALUES
  ('Economy', 9500.00),
  ('Comfort', 14500.00),
  ('Business', 32500.00)
ON CONFLICT DO NOTHING;

-- Flights
-- Keep deterministic IDs via lookup.
WITH a AS (
  SELECT id, code FROM airlines
), t AS (
  SELECT id, name FROM tariffs
)
INSERT INTO flights (flight_number, airline_id, tariff_id)
SELECT x.flight_number, a.id, t.id
FROM (VALUES
  ('SU100', 'SU', 'Economy'),
  ('SU200', 'SU', 'Business'),
  ('S710',  'S7', 'Comfort'),
  ('U650',  'U6', 'Economy')
) AS x(flight_number, airline_code, tariff_name)
JOIN a ON a.code = x.airline_code
JOIN t ON t.name = x.tariff_name
ON CONFLICT (flight_number) DO NOTHING;

UPDATE flights SET
  origin = CASE flight_number
    WHEN 'SU100' THEN 'Москва (SVO)'
    WHEN 'SU200' THEN 'Москва (SVO)'
    WHEN 'S710' THEN 'Санкт-Петербург (LED)'
    WHEN 'U650' THEN 'Екатеринбург (SVX)'
    ELSE origin
  END,
  destination = CASE flight_number
    WHEN 'SU100' THEN 'Казань (KZN)'
    WHEN 'SU200' THEN 'Сочи (AER)'
    WHEN 'S710' THEN 'Калининград (KGD)'
    WHEN 'U650' THEN 'Новосибирск (OVB)'
    ELSE destination
  END,
  departure_time = CASE flight_number
    WHEN 'SU100' THEN now() + interval '1 day 2 hours'
    WHEN 'SU200' THEN now() + interval '1 day 6 hours'
    WHEN 'S710' THEN now() + interval '2 days 1 hour'
    WHEN 'U650' THEN now() + interval '2 days 4 hours'
    ELSE departure_time
  END,
  arrival_time = CASE flight_number
    WHEN 'SU100' THEN now() + interval '1 day 4 hours'
    WHEN 'SU200' THEN now() + interval '1 day 9 hours'
    WHEN 'S710' THEN now() + interval '2 days 3 hours 30 minutes'
    WHEN 'U650' THEN now() + interval '2 days 8 hours'
    ELSE arrival_time
  END
WHERE flight_number IN ('SU100', 'SU200', 'S710', 'U650');

-- Seats + seat units (if missing)
WITH f AS (
  SELECT id, flight_number FROM flights
),
ins_seats AS (
  INSERT INTO seats (flight_id, total_seats, available_seats)
  SELECT f.id,
         CASE WHEN f.flight_number IN ('SU200') THEN 6 ELSE 10 END AS total,
         CASE WHEN f.flight_number IN ('SU200') THEN 6 ELSE 10 END AS available
  FROM f
  ON CONFLICT (flight_id) DO NOTHING
  RETURNING flight_id, total_seats
)
INSERT INTO seat_units (flight_id, seat_no, status)
SELECT x.flight_id, 'S' || lpad(gs::text, 3, '0'), 'available'
FROM (
  SELECT flight_id, total_seats FROM seats
) x
CROSS JOIN LATERAL generate_series(1, x.total_seats) gs
ON CONFLICT (flight_id, seat_no) DO NOTHING;

-- Passengers
INSERT INTO passengers (full_name, passport) VALUES
  ('Ivan Ivanov', 'AB123456'),
  ('Petr Petrov', 'CD987654'),
  ('Anna Smirnova', 'EF555111'),
  ('Olga Kuznetsova', 'GH333222')
ON CONFLICT (passport) DO NOTHING;

-- Users (password: "pass") linked with passengers
WITH p AS (
  SELECT id, passport FROM passengers
)
INSERT INTO users (email, password_hash, role, passenger_id)
SELECT * FROM (
  VALUES
   ('admin@example.com', '$2a$10$WOIGmA8lCCnB0YVYnuHXfesuttyZcJueSA2EJ9AQy.GWbsNgilfh.', 'admin', (SELECT id FROM p WHERE passport='AB123456')),
   ('agent@example.com', '$2a$10$WOIGmA8lCCnB0YVYnuHXfesuttyZcJueSA2EJ9AQy.GWbsNgilfh.', 'agent', (SELECT id FROM p WHERE passport='CD987654')),
   ('customer@example.com', '$2a$10$WOIGmA8lCCnB0YVYnuHXfesuttyZcJueSA2EJ9AQy.GWbsNgilfh.', 'customer', (SELECT id FROM p WHERE passport='EF555111'))
) x(email, password_hash, role, passenger_id)
ON CONFLICT (email) DO NOTHING;

-- Create some bookings deterministically if not exist.
-- Booking #1: Ivan -> SU100, Booking #2: Petr -> SU100, Booking #3: Anna -> S710
WITH p AS (
  SELECT id, passport FROM passengers
), f AS (
  SELECT id, flight_number FROM flights
), desired AS (
  SELECT
    (SELECT id FROM p WHERE passport='AB123456') AS passenger_id,
    (SELECT id FROM f WHERE flight_number='SU100') AS flight_id
  UNION ALL
  SELECT
    (SELECT id FROM p WHERE passport='CD987654'),
    (SELECT id FROM f WHERE flight_number='SU100')
  UNION ALL
  SELECT
    (SELECT id FROM p WHERE passport='EF555111'),
    (SELECT id FROM f WHERE flight_number='S710')
),
ins AS (
  INSERT INTO bookings (passenger_id, flight_id)
  SELECT passenger_id, flight_id FROM desired
  WHERE passenger_id IS NOT NULL AND flight_id IS NOT NULL
  ON CONFLICT DO NOTHING
  RETURNING id, flight_id, passenger_id
)
SELECT 1;

-- Assign seat_units to bookings that don't have seats yet; mark as held.
WITH b AS (
  SELECT b.id AS booking_id, b.flight_id
  FROM bookings b
  LEFT JOIN seat_units su ON su.booking_id=b.id
  WHERE su.id IS NULL
  ORDER BY b.id
),
pick AS (
  SELECT b.booking_id, su.id AS seat_unit_id
  FROM b
  JOIN LATERAL (
    SELECT id
    FROM seat_units
    WHERE flight_id=b.flight_id AND status='available' AND booking_id IS NULL
    ORDER BY id
    LIMIT 1
    FOR UPDATE SKIP LOCKED
  ) su ON true
)
UPDATE seat_units su
SET status='held', booking_id=pick.booking_id
FROM pick
WHERE su.id=pick.seat_unit_id;

-- Update seats availability counters to match held/sold (simple recalculation)
UPDATE seats s
SET available_seats = x.available
FROM (
  SELECT flight_id, COUNT(*) FILTER (WHERE status='available')::int AS available
  FROM seat_units
  GROUP BY flight_id
) x
WHERE s.flight_id = x.flight_id;

-- Create sales for first two bookings if missing, mark seats sold.
WITH b AS (
  SELECT id FROM bookings ORDER BY id LIMIT 2
),
price AS (
  SELECT b.id AS booking_id, t.price::numeric AS amount
  FROM b
  JOIN bookings bo ON bo.id=b.id
  JOIN flights f ON f.id=bo.flight_id
  JOIN tariffs t ON t.id=f.tariff_id
)
INSERT INTO sales (booking_id, amount)
SELECT booking_id, amount FROM price
ON CONFLICT (booking_id) DO NOTHING;

UPDATE seat_units su
SET status='sold'
WHERE su.booking_id IN (SELECT booking_id FROM sales);

-- One refund (partial) for the first sale
WITH s AS (
  SELECT id, amount FROM sales ORDER BY id LIMIT 1
)
INSERT INTO refunds (sale_id, amount)
SELECT s.id, (s.amount::numeric * 0.2) FROM s
ON CONFLICT DO NOTHING;

-- Notifications
INSERT INTO notifications (passenger_id, message)
SELECT p.id, msg
FROM (
  VALUES
    ('AB123456', 'Добро пожаловать! Ваше бронирование создано.'),
    ('AB123456', 'Билет оплачен. Спасибо за покупку!'),
    ('CD987654', 'Билет оплачен. Спасибо за покупку!'),
    ('EF555111', 'Добро пожаловать! Ваше бронирование создано.')
) v(passport, msg)
JOIN passengers p ON p.passport=v.passport
ON CONFLICT DO NOTHING;

-- Reports (examples)
INSERT INTO reports (report_type, payload)
VALUES
  ('seed-note', '{"note":"seed loaded","ts":"now"}'::jsonb)
ON CONFLICT DO NOTHING;

COMMIT;

