import { useEffect, useState } from "react";
import { api } from "../api";

type Flight = {
  id: number;
  flight_number: string;
  airline: { id: number; code: string };
  tariff: { id: number; price: string };
  seats: { total: number; available: number };
  origin: string;
  destination: string;
  departure_time: string | null;
  arrival_time: string | null;
};

export default function CustomerPage() {
  const [me, setMe] = useState<{ id: number; email: string; role: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [flights, setFlights] = useState<Flight[]>([]);

  function toRuError(msg: string): string {
    if (msg.includes("мест")) return msg;
    if (msg.includes("no seat units available")) return "Места недоступны, попробуйте обновить страницу.";
    if (msg.includes("no seats available")) return "На рейсе закончились места.";
    if (msg.includes("access denied")) return "Доступ запрещен для этой операции.";
    return msg;
  }

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const [m, f] = await Promise.all([
        api<{ id: number; email: string; role: string }>("/api/auth/me"),
        api<Flight[]>("/api/customer/flights"),
      ]);
      setMe(m);
      setFlights(Array.isArray(f) ? f : []);
    } catch (e: any) {
      setError(e.message || "load failed");
      setFlights([]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function bookFlight(flightID: number) {
    setError(null);
    try {
      await api("/api/customer/bookings", { method: "POST", body: JSON.stringify({ flight_id: flightID }) });
      await load();
    } catch (e: any) {
      setError(toRuError(e.message || "Не удалось забронировать"));
    }
  }

  if (!loading && me && me.role !== "customer" && me.role !== "admin") {
    return <div className="card"><h2>Пассажир</h2><p>Эта страница для роли customer/admin.</p></div>;
  }

  return (
    <div className="row" style={{ alignItems: "flex-start" }}>
      <div className="card" style={{ flex: "1 1 100%" }}>
        <h2>Рейсы и покупка билетов</h2>
        {me ? <p className="badge">Вы: {me.email} ({me.role})</p> : null}
        {error ? <p style={{ color: "#b42318" }}>{error}</p> : null}
        {loading ? <p>Загрузка...</p> : null}
        <div className="row"><button onClick={load}>Обновить</button></div>
        <table>
          <thead><tr><th>Рейс</th><th>Откуда</th><th>Куда</th><th>Вылет</th><th>Прилет</th><th>Авиакомпания</th><th>Тариф</th><th>Места</th><th /></tr></thead>
          <tbody>
            {flights.map((f) => (
              <tr key={f.id}>
                <td>{f.flight_number}</td>
                <td>{f.origin || "—"}</td>
                <td>{f.destination || "—"}</td>
                <td>{f.departure_time ? new Date(f.departure_time).toLocaleString() : "—"}</td>
                <td>{f.arrival_time ? new Date(f.arrival_time).toLocaleString() : "—"}</td>
                <td>{f.airline.code}</td>
                <td>{f.tariff.price}</td>
                <td>{f.seats.available}/{f.seats.total}</td>
                <td><button disabled={f.seats.available <= 0} onClick={() => bookFlight(f.id)}>Забронировать</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

