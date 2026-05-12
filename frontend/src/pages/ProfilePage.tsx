import { useEffect, useState } from "react";
import { api } from "../api";

type Profile = {
  user_id: number;
  email: string;
  role: string;
  passenger_id: number;
  name: string;
  lastname: string;
  contact_number: string;
  sales_history: Array<{
    id: number;
    amount: string;
    created_at: string;
    booking_id: number;
    flight_number: string;
  }>;
  refund_history: Array<{
    id: number;
    amount: string;
    created_at: string;
    sale_id: number;
    booking_id: number;
    flight_number: string;
  }>;
};

type CustomerBooking = {
  id: number;
  flight: { id: number; flight_number: string };
  sale_id: number | null;
  sale_amount: string;
  refund_id: number | null;
  is_completed: boolean;
};

export default function ProfilePage() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [name, setName] = useState("");
  const [lastname, setLastname] = useState("");
  const [contact, setContact] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [bookings, setBookings] = useState<CustomerBooking[]>([]);

  async function load() {
    setError(null);
    try {
      const [p, b] = await Promise.all([
        api<Profile>("/api/customer/profile"),
        api<CustomerBooking[]>("/api/customer/bookings"),
      ]);
      setProfile(p);
      setName(p.name || "");
      setLastname(p.lastname || "");
      setContact(p.contact_number || "");
      setBookings(Array.isArray(b) ? b : []);
    } catch (e: any) {
      setError(e.message || "Не удалось загрузить профиль");
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function save() {
    setError(null);
    setOk(null);
    try {
      await api("/api/customer/profile", {
        method: "PUT",
        body: JSON.stringify({ name, lastname, contact_number: contact }),
      });
      setOk("Профиль сохранен");
      await load();
    } catch (e: any) {
      setError(e.message || "Не удалось сохранить профиль");
    }
  }

  async function refundSale(saleID: number, amount: string) {
    setError(null);
    setOk(null);
    try {
      await api(`/api/customer/sales/${saleID}/refund`, {
        method: "POST",
        body: JSON.stringify({ amount: Number(amount) }),
      });
      setOk("Возврат успешно оформлен");
      await load();
    } catch (e: any) {
      setError(e.message || "Не удалось оформить возврат");
    }
  }

  return (
    <div className="card">
      <h2>Личный профиль</h2>
      {profile ? <p className="badge">{profile.email} ({profile.role})</p> : null}
      {error ? <p style={{ color: "#b42318" }}>{error}</p> : null}
      {ok ? <p style={{ color: "#067647" }}>{ok}</p> : null}
      <div className="row">
        <input placeholder="Имя" value={name} onChange={(e) => setName(e.target.value)} />
        <input placeholder="Фамилия" value={lastname} onChange={(e) => setLastname(e.target.value)} />
        <input placeholder="Контактный номер" value={contact} onChange={(e) => setContact(e.target.value)} />
        <button onClick={save}>Сохранить</button>
      </div>

      <h3>История покупок</h3>
      <table>
        <thead><tr><th>ID продажи</th><th>Рейс</th><th>Бронь</th><th>Сумма</th><th>Дата</th></tr></thead>
        <tbody>
          {(profile?.sales_history || []).map((s) => (
            <tr key={s.id}>
              <td>{s.id}</td>
              <td>{s.flight_number}</td>
              <td>{s.booking_id}</td>
              <td>{s.amount}</td>
              <td>{new Date(s.created_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h3>Доступные возвраты (невыполненные рейсы)</h3>
      <table>
        <thead><tr><th>Бронь</th><th>Рейс</th><th>Сумма</th><th>Статус рейса</th><th>Действие</th></tr></thead>
        <tbody>
          {bookings
            .filter((b) => b.sale_id && !b.refund_id)
            .map((b) => (
              <tr key={b.id}>
                <td>{b.id}</td>
                <td>{b.flight.flight_number}</td>
                <td>{b.sale_amount}</td>
                <td>{b.is_completed ? "Выполнен" : "Не выполнен"}</td>
                <td>
                  {b.is_completed ? "Возврат недоступен" : (
                    <button className="secondary" onClick={() => refundSale(b.sale_id!, b.sale_amount)}>Оформить возврат</button>
                  )}
                </td>
              </tr>
            ))}
        </tbody>
      </table>

      <h3>История возвратов</h3>
      <table>
        <thead><tr><th>ID возврата</th><th>Рейс</th><th>ID продажи</th><th>Сумма</th><th>Дата</th></tr></thead>
        <tbody>
          {(profile?.refund_history || []).map((r) => (
            <tr key={r.id}>
              <td>{r.id}</td>
              <td>{r.flight_number}</td>
              <td>{r.sale_id}</td>
              <td>{r.amount}</td>
              <td>{new Date(r.created_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

