import { useEffect, useRef, useState } from "react";
import * as api from "../api";
import type { PolicyMode, ProxySettings, RoutingIpType } from "../types";
import { useUI } from "../store";
import { Card, Spinner } from "./ui";
import { SystemConfigPanel } from "./SystemConfigPanel";

export function SettingsPanel({ settings, onChanged }: { settings: ProxySettings | null; onChanged: () => void }) {
  const push = useUI((s) => s.push);
  const [form, setForm] = useState<ProxySettings | null>(settings);
  const [busy, setBusy] = useState(false);
  const dirty = useRef(false);

  useEffect(() => {
    if (!settings) return;
    setForm((current) => {
      if (!dirty.current || !current) return settings;

      // The application polls settings every eight seconds. Keep the user's
      // unsaved policy fields while still reflecting favorite changes made in
      // another panel.
      return { ...current, favorite_node_ids: settings.favorite_node_ids };
    });
  }, [settings]);

  if (!form) return null;

  const set = (patch: Partial<ProxySettings>) => {
    dirty.current = true;
    setForm((current) => current ? { ...current, ...patch } : current);
  };

  async function save() {
    if (!form) return;
    setBusy(true);
    try {
      const saved = await api.updateSettings({
        routing_mode: form.routing_mode,
        force_country: form.force_country,
        routing_ip_type: form.routing_ip_type,
        connection_enabled: form.connection_enabled,
        fixed_node_id: form.fixed_node_id,
      });
      dirty.current = false;
      setForm(saved);
      push("ok", "策略已保存");
      onChanged();
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="grid gap-4">
      <Card
        title="路由策略"
        actions={<button className="btn btn-primary" disabled={busy} onClick={save}>{busy ? <Spinner /> : "保存"}</button>}
      >
        <div className="grid sm:grid-cols-2 gap-4">
          <label className="block">
            <span className="text-sm text-ink-2">路由模式</span>
            <select className="field mt-1" value={form.routing_mode}
              onChange={(e) => set({ routing_mode: e.target.value as PolicyMode })}>
              <option value="auto">延迟优先</option>
              <option value="speed_first">速度优先</option>
              <option value="smart">智能（综合）</option>
              <option value="residential_first">住宅优先</option>
              <option value="country">指定国家</option>
              <option value="fixed">固定节点</option>
              <option value="favorites">仅收藏</option>
            </select>
          </label>
          <label className="block">
            <span className="text-sm text-ink-2">IP 类型</span>
            <select className="field mt-1" value={form.routing_ip_type}
              onChange={(e) => set({ routing_ip_type: e.target.value as RoutingIpType })}>
              <option value="all">全部</option>
              <option value="residential">住宅 / 移动</option>
              <option value="hosting">机房</option>
            </select>
          </label>
          {form.routing_mode === "country" && (
            <label className="block">
              <span className="text-sm text-ink-2">国家（英文名或代码）</span>
              <input className="field mt-1" value={form.force_country}
                onChange={(e) => set({ force_country: e.target.value })} placeholder="例如 Japan 或 JP" />
            </label>
          )}
          {form.routing_mode === "fixed" && (
            <label className="block">
              <span className="text-sm text-ink-2">固定节点 ID</span>
              <input className="field mt-1" value={form.fixed_node_id ?? ""}
                onChange={(e) => set({ fixed_node_id: e.target.value })} placeholder="节点 ID" />
            </label>
          )}
          <label className="flex items-center gap-3 mt-2">
            <input type="checkbox" checked={form.connection_enabled}
              onChange={(e) => set({ connection_enabled: e.target.checked })} />
            <span className="text-sm text-ink-2">启用自动连接出口</span>
          </label>
        </div>
        <p className="text-xs text-ink-3 mt-4">
          延迟优先选择响应最快的节点；速度优先选择来源标注带宽最高的节点；智能策略综合延迟（40%）、速度（40%）和 VPN Gate 会话数（20%，越少越好）。手动切换节点后会自动锁定为固定节点，避免后台自动切回其他节点。
          收藏节点数：{form.favorite_node_ids.length}。修改策略后系统会自动校验当前出口是否仍符合规则。
        </p>
      </Card>

      <SystemConfigPanel />
    </div>
  );
}
