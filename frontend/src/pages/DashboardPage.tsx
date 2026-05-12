import { useEffect, useMemo, useState } from "react";
import { api } from "../api";

type Airline = { id: number; name: string; code: string };
type Tariff = { id: number; name: string; price: string };
type Flight = {
  id: number;
  flight_number: string;
  airline: { id: number; code: string };
  tariff: { id: number; price: string };
  seats: { total: number; available: number };
};
type Passenger = { id: number; full_name: string; passport: string };
type Booking = { id: number; passenger: { id: number; full_name: string }; flight: { id: number; flight_number: string }; created_at: string; seat_no: string };
type Sale = { id: number; booking_id: number; amount: string; created_at: string };
type Refund = { id: number; sale_id: number; amount: string; created_at: string };
type Notification = { id: number; passenger_id: number; message: string; created_at: string };
type Report = { id: number; report_type: string; created_at: string };

export default function DashboardPage() {
  const [err, setErr] = useState<string | null>(null);
  const [me, setMe] = useState<{ id: number; email: string; role: string } | null>(null);

  const [airlines, setAirlines] = useState<Airline[]>([]);
  const [tariffs, setTariffs] = useState<Tariff[]>([]);
  const [flights, setFlights] = useState<Flight[]>([]);
  const [passengers, setPassengers] = useState<Passenger[]>([]);
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [sales, setSales] = useState<Sale[]>([]);
  const [refunds, setRefunds] = useState<Refund[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [reports, setReports] = useState<Report[]>([]);
  const canManage = me?.role === "admin";

  async function loadAll() {
    setErr(null);
    try {
      const [meR, a, t, f, p, b, s, r, n] = await Promise.all([
        api<{ id: number; email: string; role: string }>("/api/auth/me"),
        api<Airline[]>("/api/airlines"),
        api<Tariff[]>("/api/tariffs"),
        api<Flight[]>("/api/flights"),
        api<Passenger[]>("/api/passengers"),
        api<Booking[]>("/api/bookings"),
        api<Sale[]>("/api/sales"),
        api<Refund[]>("/api/refunds"),
        api<Notification[]>("/api/notifications"),
      ]);
      const rep = await api<Report[]>("/api/reports");
      setMe(meR);
      setAirlines(a);
      setTariffs(t);
      setFlights(f);
      setPassengers(p);
      setBookings(b);
      setSales(s);
      setRefunds(r);
      setNotifications(n);
      setReports(rep);
    } catch (e: any) {
      setErr(e.message || "load failed");
    }
  }

  useEffect(() => {
    loadAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // forms
  const [airlineName, setAirlineName] = useState("Aeroflot");
  const [airlineCode, setAirlineCode] = useState("SU");

  const [tariffName, setTariffName] = useState("Economy");
  const [tariffPrice, setTariffPrice] = useState(10000);

  const [flightNum, setFlightNum] = useState("SU100");
  const [flightAirlineID, setFlightAirlineID] = useState<number>(1);
  const [flightTariffID, setFlightTariffID] = useState<number>(1);
  const [flightSeats, setFlightSeats] = useState(10);

  const [passName, setPassName] = useState("Ivan Ivanov");
  const [passPassport, setPassPassport] = useState("AB123456");

  const [bookingPassengerID, setBookingPassengerID] = useState<number>(1);
  const [bookingFlightID, setBookingFlightID] = useState<number>(1);

  const [saleBookingID, setSaleBookingID] = useState<number>(1);
  const [refundSaleID, setRefundSaleID] = useState<number>(1);
  const [refundAmount, setRefundAmount] = useState<number>(0);

  useEffect(() => {
    if (airlines[0]) setFlightAirlineID(airlines[0].id);
  }, [airlines]);
  useEffect(() => {
    if (tariffs[0]) setFlightTariffID(tariffs[0].id);
  }, [tariffs]);
  useEffect(() => {
    if (passengers[0]) setBookingPassengerID(passengers[0].id);
  }, [passengers]);
  useEffect(() => {
    if (flights[0]) setBookingFlightID(flights[0].id);
  }, [flights]);

  const bookingIds = useMemo(() => bookings.map((b) => b.id), [bookings]);
  const saleIds = useMemo(() => sales.map((s) => s.id), [sales]);
  useEffect(() => {
    if (bookingIds[0]) setSaleBookingID(bookingIds[0]);
  }, [bookingIds]);
  useEffect(() => {
    if (saleIds[0]) setRefundSaleID(saleIds[0]);
  }, [saleIds]);

  async function addAirline() {
    await api<{ id: number }>("/api/airlines", { method: "POST", body: JSON.stringify({ name: airlineName, code: airlineCode }) });
    await loadAll();
  }
  async function addTariff() {
    await api<{ id: number }>("/api/tariffs", { method: "POST", body: JSON.stringify({ name: tariffName, price: tariffPrice }) });
    await loadAll();
  }
  async function addFlight() {
    await api<{ id: number }>("/api/flights", {
      method: "POST",
      body: JSON.stringify({
        flight_number: flightNum,
        airline_id: flightAirlineID,
        tariff_id: flightTariffID,
        total_seats: flightSeats,
        available_seats: flightSeats,
      }),
    });
    await loadAll();
  }
  async function addPassenger() {
    await api<{ id: number }>("/api/passengers", { method: "POST", body: JSON.stringify({ full_name: passName, passport: passPassport }) });
    await loadAll();
  }
  async function addBooking() {
    await api<{ id: number }>("/api/bookings", { method: "POST", body: JSON.stringify({ passenger_id: bookingPassengerID, flight_id: bookingFlightID }) });
    await loadAll();
  }
  async function addSale() {
    await api<{ id: number }>("/api/sales", { method: "POST", body: JSON.stringify({ booking_id: saleBookingID }) });
    await loadAll();
  }
  async function addRefund() {
    const sale = sales.find((s) => s.id === refundSaleID);
    const amount = refundAmount > 0 ? refundAmount : sale ? Number(sale.amount) : 0;
    await api<{ id: number }>("/api/refunds", { method: "POST", body: JSON.stringify({ sale_id: refundSaleID, amount }) });
    await loadAll();
  }

  async function genSalesByAirline() {
    await api<{ id: number; rows: any[] }>("/api/reports/generate/sales-by-airline", { method: "POST" });
    await loadAll();
  }

  async function genFlightOccupancy() {
    await api<{ id: number; rows: any[] }>("/api/reports/generate/flight-occupancy", { method: "POST" });
    await loadAll();
  }

  return (
    <div className="row" style={{ alignItems: "flex-start" }}>
      <div className="card" style={{ flex: "1 1 420px" }}>
        <h2>{canManage ? "Управление" : "Обзор"}</h2>
        {me ? <p className="badge">Пользователь: {me.email} ({me.role})</p> : null}
        {err ? <p style={{ color: "#ffb4b4" }}>{err}</p> : null}
        <div className="row">
          <button onClick={loadAll}>Обновить</button>
        </div>

        {!canManage ? (
          <p>Для управления сущностями и продажами нужна роль admin. Для покупки билетов используйте раздел "Пассажир".</p>
        ) : (
          <>
            <h3>Airlines</h3>
            <div className="row">
              <input value={airlineName} onChange={(e) => setAirlineName(e.target.value)} placeholder="название" />
              <input value={airlineCode} onChange={(e) => setAirlineCode(e.target.value)} placeholder="код" />
              <button onClick={addAirline}>Добавить</button>
            </div>

            <h3>Tariffs</h3>
            <div className="row">
              <input value={tariffName} onChange={(e) => setTariffName(e.target.value)} placeholder="название" />
              <input value={tariffPrice} onChange={(e) => setTariffPrice(Number(e.target.value))} placeholder="цена" />
              <button onClick={addTariff}>Добавить</button>
            </div>

            <h3>Flights</h3>
            <div className="row">
              <input value={flightNum} onChange={(e) => setFlightNum(e.target.value)} placeholder="номер рейса" />
              <select value={flightAirlineID} onChange={(e) => setFlightAirlineID(Number(e.target.value))}>
                {airlines.map((a) => <option key={a.id} value={a.id}>{a.code}</option>)}
              </select>
              <select value={flightTariffID} onChange={(e) => setFlightTariffID(Number(e.target.value))}>
                {tariffs.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
              <input value={flightSeats} onChange={(e) => setFlightSeats(Number(e.target.value))} placeholder="мест" />
              <button onClick={addFlight}>Добавить</button>
            </div>

            <h3>Passengers</h3>
            <div className="row">
              <input value={passName} onChange={(e) => setPassName(e.target.value)} placeholder="имя и фамилия" />
              <input value={passPassport} onChange={(e) => setPassPassport(e.target.value)} placeholder="паспорт" />
              <button onClick={addPassenger}>Добавить</button>
            </div>

            <h3>Процесс: бронь → продажа → возврат</h3>
            <div className="row">
              <select value={bookingPassengerID} onChange={(e) => setBookingPassengerID(Number(e.target.value))}>
                {passengers.map((p) => <option key={p.id} value={p.id}>{p.full_name}</option>)}
              </select>
              <select value={bookingFlightID} onChange={(e) => setBookingFlightID(Number(e.target.value))}>
                {flights.map((f) => <option key={f.id} value={f.id}>{f.flight_number}</option>)}
              </select>
              <button onClick={addBooking}>Создать бронь</button>
            </div>
            <div className="row">
              <select value={saleBookingID} onChange={(e) => setSaleBookingID(Number(e.target.value))}>
                {bookings.map((b) => <option key={b.id} value={b.id}>Booking #{b.id}</option>)}
              </select>
              <button onClick={addSale}>Создать продажу</button>
            </div>
            <div className="row">
              <select value={refundSaleID} onChange={(e) => setRefundSaleID(Number(e.target.value))}>
                {sales.map((s) => <option key={s.id} value={s.id}>Sale #{s.id}</option>)}
              </select>
              <input value={refundAmount} onChange={(e) => setRefundAmount(Number(e.target.value))} placeholder="сумма возврата (0=полный)" />
              <button onClick={addRefund}>Создать возврат</button>
            </div>

            <h3>Reports</h3>
            <div className="row">
              <button className="secondary" onClick={genSalesByAirline}>Генерировать продажи по авиакомпаниям</button>
              <button className="secondary" onClick={genFlightOccupancy}>Генерировать загрузку рейсов</button>
            </div>
          </>
        )}
      </div>

      <div className="card" style={{ flex: "1 1 600px" }}>
        <h2>Данные</h2>
        <h3>Flights</h3>
        <table>
          <thead>
            <tr><th>ID</th><th>Number</th><th>Airline</th><th>Tariff</th><th>Seats</th></tr>
          </thead>
          <tbody>
            {flights.map((f) => (
              <tr key={f.id}>
                <td>{f.id}</td>
                <td>{f.flight_number}</td>
                <td>{f.airline.code}</td>
                <td>{f.tariff.price}</td>
                <td>{f.seats.available}/{f.seats.total}</td>
              </tr>
            ))}
          </tbody>
        </table>

        <h3>Bookings</h3>
        <table>
          <thead><tr><th>ID</th><th>Passenger</th><th>Flight</th><th>Seat</th><th>Created</th></tr></thead>
          <tbody>
            {bookings.map((b) => (
              <tr key={b.id}><td>{b.id}</td><td>{b.passenger.full_name}</td><td>{b.flight.flight_number}</td><td>{b.seat_no}</td><td>{new Date(b.created_at).toLocaleString()}</td></tr>
            ))}
          </tbody>
        </table>

        <h3>Sales</h3>
        <table>
          <thead><tr><th>ID</th><th>Booking</th><th>Amount</th><th>Created</th></tr></thead>
          <tbody>
            {sales.map((s) => (
              <tr key={s.id}><td>{s.id}</td><td>#{s.booking_id}</td><td>{s.amount}</td><td>{new Date(s.created_at).toLocaleString()}</td></tr>
            ))}
          </tbody>
        </table>

        <h3>Refunds</h3>
        <table>
          <thead><tr><th>ID</th><th>Sale</th><th>Amount</th><th>Created</th></tr></thead>
          <tbody>
            {refunds.map((r) => (
              <tr key={r.id}><td>{r.id}</td><td>#{r.sale_id}</td><td>{r.amount}</td><td>{new Date(r.created_at).toLocaleString()}</td></tr>
            ))}
          </tbody>
        </table>

        <h3>Notifications (last 200)</h3>
        <table>
          <thead><tr><th>ID</th><th>Passenger</th><th>Message</th><th>Created</th></tr></thead>
          <tbody>
            {notifications.map((n) => (
              <tr key={n.id}><td>{n.id}</td><td>{n.passenger_id}</td><td>{n.message}</td><td>{new Date(n.created_at).toLocaleString()}</td></tr>
            ))}
          </tbody>
        </table>

        <h3>Reports (last 200)</h3>
        <table>
          <thead><tr><th>ID</th><th>Type</th><th>Created</th></tr></thead>
          <tbody>
            {reports.map((r) => (
              <tr key={r.id}><td>{r.id}</td><td>{r.report_type}</td><td>{new Date(r.created_at).toLocaleString()}</td></tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

