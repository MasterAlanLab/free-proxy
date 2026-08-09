import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "./api";
import type { GatewayStatus, PoolStatistics, ProxySettings } from "./types";
import { GatewayPanel } from "./components/GatewayPanel";
import { LogsPanel } from "./components/LogsPanel";
import { NodesPanel } from "./components/NodesPanel";
import { SettingsPanel } from "./components/SettingsPanel";
import { SystemPanel } from "./components/SystemPanel";
import { StatTile, Toasts } from "./components/ui";
import { useUI } from "./store";

type Tab = "nodes" | "favorites" | "gateway" | "settings" | "system" | "logs";

const TABS: { id: Tab; label: string }[] = [
  { id: "nodes", label: "节点" },
  { id: "favorites", label: "收藏" },
  { id: "gateway", label: "网关" },
  { id: "settings", label: "策略" },
  { id: "system", label: "系统" },
  { id: "logs", label: "日志" },
];

export function App({ onLogout }: { onLogout: () => void }) {
  const push = useUI((s) => s.push);
  const [tab, setTab] = useState<Tab>("nodes");
  const [gateway, setGateway] = useState<GatewayStatus | null>(null);
  const [stats, setStats] = useState<PoolStatistics | null>(null);
  const [settings, setSettings] = useState<ProxySettings | null>(null);
  const refreshSequence = useRef(0);

  const refresh = useCallback(async () => {
    const sequence = ++refreshSequence.current;
    try {
      const [g, s, cfg] = await Promise.all([api.gatewayStatus(), api.poolStats(), api.getSettings()]);
      // A manual refresh can be started while the periodic refresh is still in
      // flight. Do not let the older response overwrite newer server state.
      if (sequence !== refreshSequence.current) return;
      setGateway(g);
      setStats(s);
      setSettings(cfg);
    } catch {
      /* handled by 401 handler */
    }
  }, []);

  useEffect(() => {
    refresh();
    const t = setInterval(refresh, 8000);
    return () => clearInterval(t);
  }, [refresh]);

  async function doLogout() {
    try {
      await api.logout();
    } catch {
      /* ignore */
    }
    push("info", "已登出");
    onLogout();
  }

  const connected = gateway?.tunnel_status === "connected";

  return (
    <div className="max-w-[1400px] mx-auto p-4 sm:p-6">
      <header className="flex flex-wrap items-center justify-between gap-4 mb-6 pb-4 border-b border-rule">
        <div>
          <div className="text-xl font-semibold tracking-tight">Free Proxy</div>
          <div className={`text-sm mt-0.5 ${connected ? "text-ok" : "text-ink-3"}`}>
            {connected ? `已连接 · ${gateway?.exit_ip ?? ""}` : "未连接"}
          </div>
        </div>
        <button className="btn" onClick={doLogout}>登出</button>
      </header>

      <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-6 gap-3 mb-6">
        <StatTile label="当前节点" value={stats?.total ?? "—"} />
        <StatTile label="当前可用" value={stats?.ready ?? "—"} />
        <StatTile label="当前住宅" value={stats?.residential ?? "—"} />
        <StatTile label="当前移动" value={stats?.mobile ?? "—"} />
        <StatTile label="当前国家/地区" value={stats?.countries ?? "—"} />
        <StatTile label="黑名单" value={stats?.blacklisted ?? "—"} />
      </div>

      <nav className="flex gap-1 mb-5 border-b border-rule">
        {TABS.map((t) => (
          <button key={t.id} onClick={() => setTab(t.id)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
              tab === t.id ? "text-ink border-ink" : "text-ink-3 border-transparent hover:text-ink"
            }`}>
            {t.id === "favorites" ? `${t.label} (${settings?.favorite_node_ids.length ?? 0})` : t.label}
          </button>
        ))}
      </nav>

      {tab === "nodes" && <NodesPanel settings={settings} onChanged={refresh} />}
      {tab === "favorites" && <NodesPanel favoriteOnly settings={settings} onChanged={refresh} />}
      {tab === "gateway" && <GatewayPanel status={gateway} onChanged={refresh} />}
      {tab === "settings" && <SettingsPanel settings={settings} onChanged={refresh} />}
      {tab === "system" && <SystemPanel />}
      {tab === "logs" && <LogsPanel />}

      <Toasts />
    </div>
  );
}
