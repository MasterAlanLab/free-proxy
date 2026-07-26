import { useCallback, useEffect, useState } from "react";
import * as api from "../api";
import type { SystemDiagnostics, SystemStatus } from "../types";
import { useUI } from "../store";
import { Card, Spinner } from "./ui";

export function SystemPanel() {
  const push = useUI((s) => s.push);
  const [diag, setDiag] = useState<SystemDiagnostics | null>(null);
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [d, s] = await Promise.all([api.systemDiagnostics(), api.systemStatus()]);
      setDiag(d);
      setStatus(s);
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [push]);

  useEffect(() => {
    load();
  }, [load]);

  async function repair() {
    try {
      await api.dnsRepair();
      push("ok", "DNS 修复已执行");
      load();
    } catch (e) {
      push("error", (e as Error).message);
    }
  }

  return (
    <div className="grid gap-4">
      <Card
        title="系统诊断"
        actions={
          <>
            <button className="btn" onClick={load} disabled={loading}>{loading ? <Spinner /> : "刷新"}</button>
            <button className="btn" onClick={repair}>修复 DNS</button>
          </>
        }
      >
        <div className="grid sm:grid-cols-2 gap-2">
          {diag?.checks.map((c) => (
            <div key={c.name} className="flex items-start gap-3 p-3 rounded-md border border-rule">
              <span className={`mt-1 w-2 h-2 rounded-full shrink-0 ${c.ok ? "bg-ok" : "bg-danger"}`} />
              <div className="min-w-0">
                <div className="text-sm font-medium">{c.name}</div>
                <div className="text-xs text-ink-3 break-words">{c.detail}</div>
              </div>
            </div>
          ))}
          {!diag && <div className="text-ink-3 text-sm">加载中…</div>}
        </div>
      </Card>

      {status && (
        <Card title={`运行状态 · v${status.version}`}>
          <div className="grid sm:grid-cols-3 gap-3 text-sm">
            <Info label="环境" value={status.environment} />
            <Info label="节点数" value={String(status.nodes)} />
            <Info label="网关" value={status.gateway_running ? "运行中" : "停止"} />
            <Info label="Web" value={status.listeners.web} />
            <Info label="SOCKS5/HTTP" value={status.listeners.socks5} />
            <Info label="当前节点" value={status.active_node_id ?? "无"} />
          </div>
          <div className="mt-4">
            <div className="text-xs text-ink-3 mb-2">后台监控</div>
            <div className="flex flex-wrap gap-x-5 gap-y-1 text-sm">
              {Object.entries(status.monitors).map(([name, running]) => (
                <span key={name} className="text-ink-2">
                  {name}：<span className={running ? "text-ok" : "text-danger"}>{running ? "运行" : "停止"}</span>
                </span>
              ))}
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="p-3 rounded-md border border-rule">
      <div className="text-xs text-ink-3">{label}</div>
      <div className="mt-0.5 font-medium break-words">{value}</div>
    </div>
  );
}
