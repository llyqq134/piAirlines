import { useState } from "react";
import { api, setTokens } from "../api";
import { Link, useNavigate } from "react-router-dom";

export default function LoginPage() {
  const nav = useNavigate();
  const [email, setEmail] = useState("a@b.c");
  const [password, setPassword] = useState("pass");
  const [error, setError] = useState<string | null>(null);

  async function doLogin() {
    setError(null);
    try {
      const r = await api<{ access_token: string; refresh_token: string }>("/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      setTokens(r);
      nav("/");
    } catch (e: any) {
      setError(e.message || "login failed");
    }
  }

  return (
    <div className="auth-page">
      <div className="card auth-card">
        <h2>Вход</h2>
        <div className="row">
          <input placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} />
          <input placeholder="Пароль" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </div>
        <div className="auth-actions">
          <button className="login-btn" onClick={doLogin}>Войти</button>
          <Link className="badge" to="/register">Перейти к регистрации</Link>
        </div>
        {error ? <p style={{ color: "#b42318" }}>{error}</p> : null}
      </div>
    </div>
  );
}

