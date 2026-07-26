import { useState } from "react";
import * as api from "../api";
import type { GatewayStatus } from "../types";
import { useUI } from "../store";
import { Card, Spinner, StatTile } from "./ui";

export function GatewayPanel({ status, onChanged }: { status: GatewayStatus | null; onChanged: () => void }) {
  const push = useUI((s) => s.push);
  const [busy, setBusy] = useState("");

  async function check() {
    setBusy("check");
    try {
      const res = await api.gatewayCheck();
      if (res.ok) push("ok", `出口 IP：${res.exit_ip}（${res.latency_ms} ms）`);
      else push("error", `出口检测失败：${res.error ?? "未知"}`);
      onChanged();
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setBusy("");
    }
  }

  async function rotate() {
    setBusy("rotate");
    try {
      const job = await api.gatewayRotate();
      await api.waitJob(job.id);
      push("ok", "已切换出口节点");
      onChanged();
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setBusy("");
    }
  }

  async function disconnect() {
    setBusy("disconnect");
    try {
      await api.gatewayDisconnect();
      push("info", "已断开出口");
      onChanged();
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setBusy("");
    }
  }

  const connected = status?.tunnel_status === "connected";
  return (
    <Card
      title="网关状态"
      actions={
        <>
          <button className="btn" disabled={!!busy} onClick={check}>
            {busy === "check" ? <Spinner /> : "检测出口"}
          </button>
          <button className="btn" disabled={!!busy} onClick={rotate}>
            {busy === "rotate" ? <Spinner /> : "切换节点"}
          </button>
          <button className="btn btn-danger" disabled={!!busy || !status?.active_node_id} onClick={disconnect}>
            断开
          </button>
        </>
      }
    >
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <StatTile label="隧道" value={connected ? "已连接" : status?.tunnel_status ?? "空闲"}
          tone={connected ? "ok" : status?.tunnel_status === "failed" ? "danger" : "default"} />
        <StatTile label="本地代理" value={status?.running ? "运行中" : "未运行"} tone={status?.running ? "ok" : "warn"} />
        <StatTile label="出口 IP" value={status?.exit_ip ?? "—"} />
        <StatTile label="活动延迟" value={status?.active_latency_ms ? `${status.active_latency_ms} ms` : "—"} />
      </div>
      <div className="mt-4 text-sm text-ink-2 space-y-1">
        <div>当前节点：<span className="text-ink">{status?.active_node_id ?? "无"}</span></div>
        <div>代理监听：<span className="text-ink font-mono text-[0.8rem]">{status?.proxy_listener}</span></div>
        {status?.last_error && <div className="text-danger">最近错误：{status.last_error}</div>}
      </div>
    </Card>
  );
}
