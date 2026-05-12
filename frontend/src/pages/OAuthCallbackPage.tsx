import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { setTokens } from "../api";

export default function OAuthCallbackPage() {
  const nav = useNavigate();
  const [params] = useSearchParams();
  const [msg, setMsg] = useState("Processing...");

  useEffect(() => {
    const access_token = params.get("access_token");
    const refresh_token = params.get("refresh_token");
    if (!access_token || !refresh_token) {
      setMsg("Missing tokens");
      return;
    }
    setTokens({ access_token, refresh_token });
    nav("/");
  }, [params, nav]);

  return (
    <div className="card">
      <h2>OAuth callback</h2>
      <p>{msg}</p>
    </div>
  );
}

