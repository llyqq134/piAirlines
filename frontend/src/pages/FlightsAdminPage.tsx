import { useEffect, useState } from "react";
import { api } from "../api";

type AdminFlight = {
  id: number;
  flight_number: string;
  origin: string;
  destination: string;
  departure_time: string | null;
  arrival_time: string | null;
  is_completed: boolean;
  completed_at: string | null;
};

export default function FlightsAdminPage() {
  const [me, setMe] = useState<{ role: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [flights, setFlights] = useState<AdminFlight[]>([]);

  async function load() {
    setError(null);
    try {
      const [m, f] = await Promise.all([
        api<{ role: string }>("/api/auth/me"),
        api<AdminFlight[]>("/api/admin/flights"),
      ]);
      setMe(m);
      setFlights(Array.isArray(f) ? f : []);
    } catch (e: any) {
      setError(e.message || "Ошибка загрузки");
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function toggleCompleted(f: AdminFlight) {
    setError(null);
    try {
      await api(`/api/admin/flights/${f.id}/completed`, {
        method: "PATCH",
        body: JSON.stringify({ is_completed: !f.is_completed }),
      });
      await load();
    } catch (e: any) {
      setError(e.message || "Ошибка обновления статуса");
    }
  }

  if (me && me.role !== "admin") {
    return <div className="card"><h2>Полеты</h2><p>Доступ только для admin.</p></div>;
  }

  return (
    <div className="card">
      <h2>Полеты (управление выполнением)</h2>
      {error ? <p style={{ color: "#b42318" }}>{error}</p> : null}
      <table>
        <thead>
          <tr><th>Рейс</th><th>Откуда</th><th>Куда</th><th>Вылет</th><th>Прилет</th><th>Статус</th><th>Отметить</th></tr>
        </thead>
        <tbody>
          {flights.map((f) => (
            <tr key={f.id}>
              <td>{f.flight_number}</td>
              <td>{f.origin || "—"}</td>
              <td>{f.destination || "—"}</td>
              <td>{f.departure_time ? new Date(f.departure_time).toLocaleString() : "—"}</td>
              <td>{f.arrival_time ? new Date(f.arrival_time).toLocaleString() : "—"}</td>
              <td>{f.is_completed ? "Выполнен" : "Запланирован"}</td>
              <td>
                <button className={f.is_completed ? "secondary" : ""} onClick={() => toggleCompleted(f)}>
                  {f.is_completed ? "Снять отметку" : "Отметить выполненным"}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

