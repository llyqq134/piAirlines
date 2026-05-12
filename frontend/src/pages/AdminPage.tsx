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
type User = { id: number; email: string; role: "admin" | "agent" | "customer"; created_at: string };

export default function AdminPage() {
  const [err, setErr] = useState<string | null>(null);
  const [me, setMe] = useState<{ id: number; email: string; role: string } | null>(null);

  const [airlines, setAirlines] = useState<Airline[]>([]);
  const [tariffs, setTariffs] = useState<Tariff[]>([]);
  const [flights, setFlights] = useState<Flight[]>([]);
  const [passengers, setPassengers] = useState<Passenger[]>([]);
  const [users, setUsers] = useState<User[]>([]);

  const isAdmin = me?.role === "admin";

  async function load() {
    setErr(null);
    try {
      const meR = await api<{ id: number; email: string; role: string }>("/api/auth/me");
      setMe(meR);
      const [a, t, f, p] = await Promise.all([
        api<Airline[]>("/api/airlines"),
        api<Tariff[]>("/api/tariffs"),
        api<Flight[]>("/api/flights"),
        api<Passenger[]>("/api/passengers"),
      ]);
      setAirlines(a);
      setTariffs(t);
      setFlights(f);
      setPassengers(p);

      if (meR.role === "admin") {
        const u = await api<User[]>("/api/admin/users");
        setUsers(u);
      } else {
        setUsers([]);
      }
    } catch (e: any) {
      setErr(e.message || "load failed");
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const airlineById = useMemo(() => new Map(airlines.map((a) => [a.id, a])), [airlines]);
  const tariffById = useMemo(() => new Map(tariffs.map((t) => [t.id, t])), [tariffs]);

  async function saveAirline(a: Airline) {
    await api<{ ok: true }>(`/api/airlines/${a.id}`, { method: "PUT", body: JSON.stringify({ name: a.name, code: a.code }) });
    await load();
  }
  async function delAirline(id: number) {
    await api<{ ok: true }>(`/api/airlines/${id}`, { method: "DELETE" });
    await load();
  }

  async function saveTariff(t: Tariff) {
    await api<{ ok: true }>(`/api/tariffs/${t.id}`, { method: "PUT", body: JSON.stringify({ name: t.name, price: Number(t.price) }) });
    await load();
  }
  async function delTariff(id: number) {
    await api<{ ok: true }>(`/api/tariffs/${id}`, { method: "DELETE" });
    await load();
  }

  async function savePassenger(p: Passenger) {
    await api<{ ok: true }>(`/api/passengers/${p.id}`, { method: "PUT", body: JSON.stringify({ full_name: p.full_name, passport: p.passport }) });
    await load();
  }
  async function delPassenger(id: number) {
    await api<{ ok: true }>(`/api/passengers/${id}`, { method: "DELETE" });
    await load();
  }

  async function saveFlight(f: Flight) {
    await api<{ ok: true }>(`/api/flights/${f.id}`, {
      method: "PUT",
      body: JSON.stringify({ flight_number: f.flight_number, airline_id: f.airline.id, tariff_id: f.tariff.id }),
    });
    await load();
  }
  async function delFlight(id: number) {
    await api<{ ok: true }>(`/api/flights/${id}`, { method: "DELETE" });
    await load();
  }

  async function setUserRole(id: number, role: "admin" | "agent" | "customer") {
    await api<{ ok: true }>(`/api/admin/users/${id}/role`, { method: "PATCH", body: JSON.stringify({ role }) });
    await load();
  }

  if (!isAdmin) {
    return (
      <div className="card">
        <h2>Админка</h2>
        {err ? <p style={{ color: "#ffb4b4" }}>{err}</p> : null}
        <p>Доступ только для роли <code>admin</code>.</p>
      </div>
    );
  }

  return (
    <div className="row" style={{ alignItems: "flex-start" }}>
      <div className="card" style={{ flex: "1 1 1000px" }}>
        <h2>Админка</h2>
        {me ? <p className="badge">Пользователь: {me.email} ({me.role})</p> : null}
        {err ? <p style={{ color: "#ffb4b4" }}>{err}</p> : null}
        <div className="row">
          <button onClick={load}>Обновить</button>
        </div>

        <h3>Пользователи</h3>
        <table>
          <thead><tr><th>ID</th><th>Email</th><th>Роль</th><th>Создан</th><th /></tr></thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id}>
                <td>{u.id}</td>
                <td>{u.email}</td>
                <td>
                  <select value={u.role} onChange={(e) => setUserRole(u.id, e.target.value as any)}>
                    <option value="admin">admin</option>
                    <option value="agent">agent</option>
                    <option value="customer">customer</option>
                  </select>
                </td>
                <td>{new Date(u.created_at).toLocaleString()}</td>
                <td />
              </tr>
            ))}
          </tbody>
        </table>

        <h3>Авиакомпании (редактирование/удаление)</h3>
        <table>
          <thead><tr><th>ID</th><th>Name</th><th>Code</th><th /></tr></thead>
          <tbody>
            {airlines.map((a) => (
              <tr key={a.id}>
                <td>{a.id}</td>
                <td><input value={a.name} onChange={(e) => setAirlines((xs) => xs.map((x) => x.id === a.id ? { ...x, name: e.target.value } : x))} /></td>
                <td><input value={a.code} onChange={(e) => setAirlines((xs) => xs.map((x) => x.id === a.id ? { ...x, code: e.target.value } : x))} /></td>
                <td className="row">
                  <button onClick={() => saveAirline(a)}>Сохранить</button>
                  <button className="secondary" onClick={() => delAirline(a.id)}>Удалить</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        <h3>Тарифы (редактирование/удаление)</h3>
        <table>
          <thead><tr><th>ID</th><th>Name</th><th>Price</th><th /></tr></thead>
          <tbody>
            {tariffs.map((t) => (
              <tr key={t.id}>
                <td>{t.id}</td>
                <td><input value={t.name} onChange={(e) => setTariffs((xs) => xs.map((x) => x.id === t.id ? { ...x, name: e.target.value } : x))} /></td>
                <td><input value={t.price} onChange={(e) => setTariffs((xs) => xs.map((x) => x.id === t.id ? { ...x, price: e.target.value } : x))} /></td>
                <td className="row">
                  <button onClick={() => saveTariff(t)}>Сохранить</button>
                  <button className="secondary" onClick={() => delTariff(t.id)}>Удалить</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        <h3>Рейсы (редактирование/удаление)</h3>
        <table>
          <thead><tr><th>ID</th><th>Number</th><th>Airline</th><th>Tariff</th><th>Seats</th><th /></tr></thead>
          <tbody>
            {flights.map((f) => (
              <tr key={f.id}>
                <td>{f.id}</td>
                <td><input value={f.flight_number} onChange={(e) => setFlights((xs) => xs.map((x) => x.id === f.id ? { ...x, flight_number: e.target.value } : x))} /></td>
                <td>
                  <select value={f.airline.id} onChange={(e) => setFlights((xs) => xs.map((x) => x.id === f.id ? { ...x, airline: { ...x.airline, id: Number(e.target.value), code: airlineById.get(Number(e.target.value))?.code || "" } } : x))}>
                    {airlines.map((a) => <option key={a.id} value={a.id}>{a.code}</option>)}
                  </select>
                </td>
                <td>
                  <select value={f.tariff.id} onChange={(e) => setFlights((xs) => xs.map((x) => x.id === f.id ? { ...x, tariff: { ...x.tariff, id: Number(e.target.value), price: tariffById.get(Number(e.target.value))?.price || "0" } } : x))}>
                    {tariffs.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
                  </select>
                </td>
                <td>{f.seats.available}/{f.seats.total}</td>
                <td className="row">
                  <button onClick={() => saveFlight(f)}>Сохранить</button>
                  <button className="secondary" onClick={() => delFlight(f.id)}>Удалить</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        <h3>Пассажиры (редактирование/удаление)</h3>
        <table>
          <thead><tr><th>ID</th><th>Name</th><th>Passport</th><th /></tr></thead>
          <tbody>
            {passengers.map((p) => (
              <tr key={p.id}>
                <td>{p.id}</td>
                <td><input value={p.full_name} onChange={(e) => setPassengers((xs) => xs.map((x) => x.id === p.id ? { ...x, full_name: e.target.value } : x))} /></td>
                <td><input value={p.passport} onChange={(e) => setPassengers((xs) => xs.map((x) => x.id === p.id ? { ...x, passport: e.target.value } : x))} /></td>
                <td className="row">
                  <button onClick={() => savePassenger(p)}>Сохранить</button>
                  <button className="secondary" onClick={() => delPassenger(p.id)}>Удалить</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

