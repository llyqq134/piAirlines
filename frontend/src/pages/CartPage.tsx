import { useEffect, useMemo, useState } from "react";
import { api } from "../api";

type CustomerBooking = {
  id: number;
  flight: { id: number; flight_number: string };
  created_at: string;
  seat_no: string;
  sale_id: number | null;
  sale_amount: string;
  booking_amount: string;
};

export default function CartPage() {
  const [error, setError] = useState<string | null>(null);
  const [bookings, setBookings] = useState<CustomerBooking[]>([]);
  const [selected, setSelected] = useState<number[]>([]);

  async function load() {
    setError(null);
    try {
      const b = await api<CustomerBooking[]>("/api/customer/bookings");
      const unpaid = (Array.isArray(b) ? b : []).filter((x) => !x.sale_id);
      setBookings(unpaid);
      setSelected((prev) => prev.filter((id) => unpaid.some((u) => u.id === id)));
    } catch (e: any) {
      setError(e.message || "Ошибка загрузки корзины");
      setBookings([]);
    }
  }

  useEffect(() => {
    load();
  }, []);

  const total = useMemo(
    () =>
      bookings
        .filter((b) => selected.includes(b.id))
        .reduce((acc, b) => acc + Number(b.booking_amount || b.sale_amount || 0), 0),
    [bookings, selected],
  );

  async function paySelected() {
    if (selected.length === 0) {
      setError("Выберите билеты для оплаты.");
      return;
    }
    setError(null);
    try {
      await api("/api/customer/cart/pay", { method: "POST", body: JSON.stringify({ booking_ids: selected }) });
      await load();
    } catch (e: any) {
      setError(e.message || "Не удалось оплатить выбранные билеты");
    }
  }

  async function removeSelected() {
    if (selected.length === 0) {
      setError("Выберите позиции для удаления.");
      return;
    }
    setError(null);
    try {
      await api("/api/customer/cart/remove", { method: "POST", body: JSON.stringify({ booking_ids: selected }) });
      await load();
    } catch (e: any) {
      setError(e.message || "Не удалось удалить выбранные позиции");
    }
  }

  async function removeOne(bookingID: number) {
    setError(null);
    try {
      await api("/api/customer/cart/remove", { method: "POST", body: JSON.stringify({ booking_ids: [bookingID] }) });
      await load();
    } catch (e: any) {
      setError(e.message || "Не удалось удалить позицию");
    }
  }

  return (
    <div className="cart-layout">
      <div className="card cart-main">
        <h2>Моя корзина ({bookings.length})</h2>
        {error ? <p style={{ color: "#b42318" }}>{error}</p> : null}
        <div className="row">
          <button className="secondary" onClick={load}>Обновить</button>
          <button className="secondary" onClick={removeSelected}>Удалить выбранные</button>
        </div>
        <table>
          <thead>
            <tr><th></th><th>Рейс</th><th>Место</th><th>Цена</th><th>Создано</th><th></th></tr>
          </thead>
          <tbody>
            {bookings.map((b) => (
              <tr key={b.id}>
                <td>
                  <input
                    type="checkbox"
                    checked={selected.includes(b.id)}
                    onChange={(e) =>
                      setSelected((prev) => (e.target.checked ? [...prev, b.id] : prev.filter((id) => id !== b.id)))
                    }
                  />
                </td>
                <td>{b.flight?.flight_number ?? "—"}</td>
                <td>{b.seat_no || "—"}</td>
                <td>{b.booking_amount || b.sale_amount}</td>
                <td>{new Date(b.created_at).toLocaleDateString()}</td>
                <td>
                  <button className="secondary" onClick={() => removeOne(b.id)} title="Удалить">✕</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="card cart-summary">
        <h3 style={{ textAlign: "center" }}>Итого</h3>
        <p className="cart-total">{total.toFixed(2)}</p>
        <button onClick={paySelected}>Оплатить</button>
      </div>
    </div>
  );
}

