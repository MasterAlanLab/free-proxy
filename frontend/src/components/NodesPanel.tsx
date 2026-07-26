import { useCallback, useEffect, useState } from "react";
import * as api from "../api";
import type { ProxyNode, ProxySettings } from "../types";
import { useUI } from "../store";
import { Badge, Card, Spinner } from "./ui";

const PAGE = 20;

export function NodesPanel({ settings, onChanged }: { settings: ProxySettings | null; onChanged: () => void }) {
  const push = useUI((s) => s.push);
  const [items, setItems] = useState<ProxyNode[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [search, setSearch] = useState("");
  const [ipType, setIpType] = useState("");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState("");

  const favorites = new Set(settings?.favorite_node_ids ?? []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.listProxies({
        limit: PAGE, offset: page * PAGE, search, ip_type: ipType, status,
      });
      setItems(res.items);
      setTotal(res.total);
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [page, search, ipType, status, push]);

  useEffect(() => {
    load();
  }, [load]);

  async function runJob(label: string, fn: () => Promise<{ id: string }>) {
    setBusy(label);
    try {
      const job = await fn();
      await api.waitJob(job.id);
      push("ok", `${label}完成`);
      await load();
      onChanged();
    } catch (e) {
      push("error", `${label}失败：${(e as Error).message}`);
    } finally {
      setBusy("");
    }
  }

  async function favorite(id: string) {
    try {
      await api.toggleFavorite(id);
      onChanged();
      await load();
    } catch (e) {
      push("error", (e as Error).message);
    }
  }

  const toggleSel = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });

  const pages = Math.max(1, Math.ceil(total / PAGE));

  return (
    <Card
      title={`节点池（${total}）`}
      actions={
        <>
          <button className="btn btn-primary" disabled={!!busy}
            onClick={() => runJob("更新并检测", api.refresh)}>
            {busy === "更新并检测" ? <Spinner /> : "更新并检测节点"}
          </button>
          <button className="btn" disabled={!!busy} onClick={() => runJob("发现节点", api.discover)}>
            {busy === "发现节点" ? <Spinner /> : "仅发现"}
          </button>
          <button className="btn" disabled={!!busy || selected.size === 0}
            onClick={() => runJob("探测选中", () => api.probeMany([...selected]))}>
            探测选中（{selected.size}）
          </button>
        </>
      }
    >
      <div className="flex flex-wrap gap-2 mb-4">
        <input className="field flex-1 min-w-[200px]" placeholder="搜索 IP / 主机名 / 国家 / ASN"
          value={search} onChange={(e) => { setPage(0); setSearch(e.target.value); }} />
        <select className="field w-auto" value={ipType} onChange={(e) => { setPage(0); setIpType(e.target.value); }}>
          <option value="">全部类型</option>
          <option value="residential">住宅</option>
          <option value="mobile">移动</option>
          <option value="hosting">机房</option>
          <option value="unknown">未知</option>
        </select>
        <select className="field w-auto" value={status} onChange={(e) => { setPage(0); setStatus(e.target.value); }}>
          <option value="">全部状态</option>
          <option value="ready">可用</option>
          <option value="discovered">已发现</option>
          <option value="unavailable">不可用</option>
          <option value="cooldown">冷却</option>
        </select>
        <button className="btn" onClick={load} disabled={loading}>{loading ? <Spinner /> : "刷新"}</button>
      </div>

      <div className="overflow-x-auto rounded-md border border-rule">
        <table className="w-full min-w-[880px] border-collapse">
          <thead>
            <tr>
              <th className="th w-8"></th>
              <th className="th">国家 / 主机</th>
              <th className="th">IP</th>
              <th className="th">类型</th>
              <th className="th">状态</th>
              <th className="th">延迟</th>
              <th className="th">来源 Ping</th>
              <th className="th text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            {items.length === 0 && (
              <tr><td className="td text-center text-ink-3 py-8" colSpan={8}>
                {loading ? "加载中…" : "暂无节点，点击“更新并检测节点”开始。"}
              </td></tr>
            )}
            {items.map((n) => (
              <tr key={n.id} className="hover:bg-paper-2/50">
                <td className="td">
                  <input type="checkbox" checked={selected.has(n.id)} onChange={() => toggleSel(n.id)} />
                </td>
                <td className="td">
                  <div className="font-medium">{n.country || n.country_code || "—"}</div>
                  <div className="text-xs text-ink-3 truncate max-w-[220px]">{n.host_name || n.owner || n.as_name}</div>
                </td>
                <td className="td font-mono text-[0.8rem]">{n.ip_address}<div className="text-xs text-ink-3 font-sans">{n.transport}</div></td>
                <td className="td"><Badge label={ipLabel(n.ip_type)} tone={n.ip_type} /></td>
                <td className="td"><Badge label={statusLabel(n.status)} tone={n.status} /></td>
                <td className="td tabular-nums">{n.latency_ms > 0 ? `${n.latency_ms} ms` : "—"}</td>
                <td className="td tabular-nums text-ink-3">{n.source_ping_ms > 0 ? `${n.source_ping_ms} ms` : "—"}</td>
                <td className="td text-right whitespace-nowrap">
                  <button className="btn btn-sm btn-primary mr-1" disabled={!!busy}
                    onClick={() => runJob("激活", () => api.activate(n.id))}>激活</button>
                  <button className="btn btn-sm mr-1" disabled={!!busy}
                    onClick={() => runJob("探测", () => api.probeOne(n.id))}>探测</button>
                  <button className="btn btn-sm mr-1" onClick={() => favorite(n.id)}>
                    {favorites.has(n.id) ? "★" : "☆"}
                  </button>
                  <a className="btn btn-sm" href={api.configUrl(n.id)}>下载</a>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between mt-4 text-sm text-ink-3">
        <span>第 {page + 1} / {pages} 页</span>
        <div className="flex gap-2">
          <button className="btn btn-sm" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>上一页</button>
          <button className="btn btn-sm" disabled={page + 1 >= pages} onClick={() => setPage((p) => p + 1)}>下一页</button>
        </div>
      </div>
    </Card>
  );
}

function ipLabel(t: string) {
  return { residential: "住宅", mobile: "移动", hosting: "机房", unknown: "未知" }[t] ?? t;
}
function statusLabel(s: string) {
  return { ready: "可用", discovered: "已发现", probing: "探测中", unavailable: "不可用", cooldown: "冷却" }[s] ?? s;
}
