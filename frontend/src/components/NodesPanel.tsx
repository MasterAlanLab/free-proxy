import { useCallback, useEffect, useState } from "react";
import * as api from "../api";
import type { ProxyNode, ProxySettings } from "../types";
import { useUI } from "../store";
import { Badge, Card, Spinner } from "./ui";

const PAGE = 20;

export function NodesPanel({ settings, onChanged, favoriteOnly = false }: {
  settings: ProxySettings | null;
  onChanged: () => void;
  favoriteOnly?: boolean;
}) {
  const push = useUI((s) => s.push);
  const [items, setItems] = useState<ProxyNode[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [search, setSearch] = useState("");
  const [ipType, setIpType] = useState("");
  const [status, setStatus] = useState("");
  const [listedOnly, setListedOnly] = useState(false);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState("");

  const favorites = new Set(settings?.favorite_node_ids ?? []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.listProxies({
        limit: PAGE, offset: page * PAGE, search, ip_type: ipType, status,
        favorite: favoriteOnly,
        listed_only: !favoriteOnly && listedOnly,
      });
      setItems(res.items);
      setTotal(res.total);
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [page, search, ipType, status, listedOnly, favoriteOnly, push]);

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
      const updated = await api.toggleFavorite(id);
      const kept = updated.favorite_node_ids.includes(id);
      push("ok", kept ? "已加入收藏" : "已取消收藏");
      if (!kept) {
        setSelected((prev) => {
          const next = new Set(prev);
          next.delete(id);
          return next;
        });
      }
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

  useEffect(() => {
    if (page >= pages && page > 0) setPage(pages - 1);
  }, [page, pages]);

  return (
    <Card
      title={`${favoriteOnly ? "收藏节点" : listedOnly ? "来源最新名单" : "当前节点池"}（${total}）`}
      actions={
        <>
          {!favoriteOnly && <>
            <button className="btn btn-primary" disabled={!!busy}
              onClick={() => runJob("更新并检测", api.refresh)}>
              {busy === "更新并检测" ? <Spinner /> : "更新并检测节点"}
            </button>
            <button className="btn" disabled={!!busy} onClick={() => runJob("发现节点", api.discover)}>
              {busy === "发现节点" ? <Spinner /> : "仅发现"}
            </button>
          </>}
          <button className="btn" disabled={!!busy || selected.size === 0}
            onClick={() => runJob("测试节点", () => api.probeMany([...selected]))}
            title="测试节点是否能连接，并记录实际延迟">
            测试节点（{selected.size}）
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
        {!favoriteOnly && <label className="flex items-center gap-2 px-2 text-sm text-ink-2 whitespace-nowrap">
          <input type="checkbox" checked={listedOnly}
            onChange={(e) => { setPage(0); setListedOnly(e.target.checked); }} />
          仅来源最新名单
        </label>}
        <button className="btn" onClick={load} disabled={loading}>{loading ? <Spinner /> : "刷新"}</button>
      </div>

      <p className="text-xs text-ink-3 mb-3">
        {favoriteOnly ? <>
          这里包含全部收藏记录；标记为“未在名单”的节点只是不在来源最近一次公布的名单里，多数仍然可用，
          可以先测试确认后再切换。取消收藏后节点会从本页移除。
        </> : <>
          默认显示节点池全部节点，每页 {PAGE} 条。VPN Gate 每次只公布约 100 台轮换节点，节点池会持续累积，
          并由后台探活自动删除确认失效的节点，因此“未在名单”不代表不可用。
        </>}
        “切换节点”会立即使用该节点，并自动改为固定节点；“测试节点”只检查连接和延迟，不会切换当前节点。
      </p>
      <div className="overflow-x-auto rounded-md border border-rule">
        <table className="w-full min-w-[980px] border-collapse">
          <thead>
            <tr>
              <th className="th w-8"></th>
              <th className="th">国家 / 机构</th>
              <th className="th">IP</th>
              <th className="th">类型</th>
              <th className="th">状态</th>
              <th className="th">延迟</th>
              <th className="th">来源 Ping</th>
              <th className="th">来源速度</th>
              <th className="th" title="VPN Gate 最近一次抓取时的会话数，越少表示节点负载越低">会话数</th>
              <th className="th text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            {items.length === 0 && (
              <tr><td className="td text-center text-ink-3 py-8" colSpan={10}>
                {loading ? "加载中…" : favoriteOnly ? "暂无收藏节点，请先在节点页面收藏常用节点。" : "暂无节点，点击“更新并检测节点”开始。"}
              </td></tr>
            )}
            {items.map((n) => (
              <tr key={n.id} className="hover:bg-paper-2/50">
                <td className="td">
                  <input type="checkbox"
                    checked={selected.has(n.id)} onChange={() => toggleSel(n.id)} />
                </td>
                <td className="td">
                  <div className="font-medium">{n.country || n.country_code || "—"}</div>
                  <div className="text-xs text-ink-3 truncate max-w-[240px]" title={n.host_name || ""}>
                    {n.owner || n.as_name || n.host_name || "—"}
                  </div>
                </td>
                <td className="td font-mono text-[0.8rem]">{n.ip_address}<div className="text-xs text-ink-3 font-sans">{n.transport}</div></td>
                <td className="td"><Badge label={ipLabel(n.ip_type)} tone={n.ip_type} /></td>
                <td className="td">
                  <Badge label={statusLabel(n.status)} tone={n.status} />
                  {!n.source_present && <div className="text-xs text-ink-3 mt-1"
                    title="不在来源最近一次公布的名单里；来源每次只公布约 100 台轮换节点，这不代表节点不可用">未在名单</div>}
                </td>
                <td className="td tabular-nums">{n.latency_ms > 0 ? `${n.latency_ms} ms` : "—"}</td>
                <td className="td tabular-nums text-ink-3">{n.source_ping_ms > 0 ? `${n.source_ping_ms} ms` : "—"}</td>
                <td className="td tabular-nums text-ink-3">{formatSpeed(n.source_speed_bps)}</td>
                <td className="td tabular-nums text-ink-3">{n.source_sessions}</td>
                <td className="td text-right whitespace-nowrap">
                  <button className="btn btn-sm btn-primary mr-1" disabled={!!busy}
                    onClick={() => runJob("切换节点", () => api.activate(n.id))}
                    title="切换到此节点，并锁定为固定节点">
                    切换节点
                  </button>
                  <button className="btn btn-sm mr-1" disabled={!!busy}
                    onClick={() => runJob("测试节点", () => api.probeOne(n.id))}
                    title="测试此节点是否能连接，并记录实际延迟">
                    测试节点
                  </button>
                  <button className="btn btn-sm mr-1" onClick={() => favorite(n.id)}>
                    {favoriteOnly ? "取消收藏" : favorites.has(n.id) ? "★" : "☆"}
                  </button>
                  <a className="btn btn-sm" href={api.configUrl(n.id)}>下载</a>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between mt-4 text-sm text-ink-3">
        <span>
          第 {page + 1} / {pages} 页
          {total > 0 && ` · 当前显示 ${page * PAGE + 1}–${Math.min((page + 1) * PAGE, total)}，共 ${total} 个`}
        </span>
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
  return { ready: "可用", discovered: "已发现", probing: "测试中", unavailable: "不可用", cooldown: "冷却" }[s] ?? s;
}

function formatSpeed(bps: number) {
  if (!bps || bps <= 0) return "—";
  if (bps >= 1_000_000) return `${(bps / 1_000_000).toFixed(1)} Mbps`;
  if (bps >= 1_000) return `${Math.round(bps / 1_000)} Kbps`;
  return `${bps} bps`;
}
