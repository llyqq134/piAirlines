import { Link, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";
import "./styles.css";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import OAuthCallbackPage from "./pages/OAuthCallbackPage";
import DashboardPage from "./pages/DashboardPage";
import AdminPage from "./pages/AdminPage";
import CustomerPage from "./pages/CustomerPage";
import CartPage from "./pages/CartPage";
import ProfilePage from "./pages/ProfilePage";
import FlightsAdminPage from "./pages/FlightsAdminPage";
import { api, setTokens } from "./api";

export default function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const [isAuthed, setIsAuthed] = useState(false);
  const [role, setRole] = useState<string>("");

  async function refreshAuthState() {
    const access = localStorage.getItem("access_token");
    if (!access) {
      setIsAuthed(false);
      setRole("");
      return;
    }
    try {
      const me = await api<{ role: string }>("/api/auth/me");
      setIsAuthed(true);
      setRole(me.role);
    } catch {
      setIsAuthed(false);
      setRole("");
    }
  }

  useEffect(() => {
    refreshAuthState();
  }, [location.pathname]);

  function logout() {
    setTokens(null);
    setIsAuthed(false);
    setRole("");
    navigate("/login");
  }

  return (
    <div className="container">
      <div className="topbar">
        <div className="nav-left">
          {role === "admin" ? <Link className="badge" to="/">Панель</Link> : <Link className="badge" to="/">Рейсы</Link>}
          {isAuthed && role !== "admin" ? <Link className="badge" to="/cart">Корзина</Link> : null}
          {isAuthed && role !== "admin" ? <Link className="badge" to="/profile">Профиль</Link> : null}
          {role === "admin" ? <Link className="badge" to="/admin">Управление</Link> : null}
          {role === "admin" ? <Link className="badge" to="/flights">Полеты</Link> : null}
        </div>
        <div className="nav-right">
          {!isAuthed ? (
            <>
              <Link className="badge" to="/login">Вход</Link>
              <Link className="badge" to="/register">Регистрация</Link>
            </>
          ) : (
            <button className="secondary" onClick={logout}>Выход</button>
          )}
        </div>
      </div>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/oauth/callback" element={<OAuthCallbackPage />} />
        <Route path="/" element={isAuthed ? (role === "admin" ? <DashboardPage /> : <CustomerPage />) : <LoginPage />} />
        <Route path="/customer" element={<CustomerPage />} />
        <Route path="/cart" element={<CartPage />} />
        <Route path="/profile" element={<ProfilePage />} />
        <Route path="/admin" element={<AdminPage />} />
        <Route path="/flights" element={<FlightsAdminPage />} />
      </Routes>
    </div>
  );
}

