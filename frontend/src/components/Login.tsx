import { useState } from "react";
import * as api from "../api";
import { Spinner } from "./ui";

export function Login({ onSuccess }: { onSuccess: () => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.login(username, password);
      onSuccess();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="min-h-full grid place-items-center p-4">
      <form onSubmit={submit} className="card p-8 w-[min(400px,92vw)]">
        <div className="mb-6">
          <div className="text-xl font-semibold tracking-tight">Free Proxy</div>
          <div className="text-sm text-ink-2 mt-1">管理控制台登录</div>
        </div>
        <label className="block text-sm text-ink-2 mb-1">用户名</label>
        <input className="field mb-4" value={username} autoFocus
          onChange={(e) => setUsername(e.target.value)} autoComplete="username" />
        <label className="block text-sm text-ink-2 mb-1">密码</label>
        <input className="field mb-5" type="password" value={password}
          onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" />
        {error && <div className="text-sm text-danger mb-4">{error}</div>}
        <button className="btn btn-primary w-full" disabled={busy}>
          {busy ? <Spinner /> : "登录"}
        </button>
      </form>
    </div>
  );
}
