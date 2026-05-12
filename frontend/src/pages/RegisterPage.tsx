import { useState } from "react";
import { api, setTokens } from "../api";
import { Link, useNavigate } from "react-router-dom";

export default function RegisterPage() {
  const nav = useNavigate();
  const [email, setEmail] = useState("new@example.com");
  const [password, setPassword] = useState("pass");
  const [name, setName] = useState("Иван");
  const [lastname, setLastname] = useState("Иванов");
  const [contactNumber, setContactNumber] = useState("+79990001122");
  const [error, setError] = useState<string | null>(null);

  async function doRegister() {
    setError(null);
    try {
      const r = await api<{ access_token: string; refresh_token: string }>("/api/auth/register", {
        method: "POST",
        body: JSON.stringify({ email, password, name, lastname, contact_number: contactNumber }),
      });
      setTokens(r);
      nav("/customer");
    } catch (e: any) {
      setError(e.message || "Ошибка регистрации");
    }
  }

  return (
    <div className="auth-page">
      <div className="card auth-card">
        <h2>Регистрация</h2>
        <div className="row">
          <input placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} />
          <input placeholder="Пароль" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          <input placeholder="Имя" value={name} onChange={(e) => setName(e.target.value)} />
          <input placeholder="Фамилия" value={lastname} onChange={(e) => setLastname(e.target.value)} />
          <input placeholder="Контактный номер" value={contactNumber} onChange={(e) => setContactNumber(e.target.value)} />
        </div>
        <div className="auth-actions">
          <button className="login-btn" onClick={doRegister}>Создать аккаунт</button>
          <Link className="badge" to="/login">Уже есть аккаунт</Link>
        </div>
        {error ? <p style={{ color: "#b42318" }}>{error}</p> : null}
      </div>
    </div>
  );
}

