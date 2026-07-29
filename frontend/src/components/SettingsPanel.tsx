import { useEffect, useState } from "react";
import * as api from "../api";
import type { AccessConfig, PolicyMode, ProxySettings, RoutingIpType } from "../types";
import { useUI } from "../store";
import { Card, Spinner } from "./ui";

export function SettingsPanel({ settings, onChanged }: { settings: ProxySettings | null; onChanged: () => void }) {
  const push = useUI((s) => s.push);
  const [form, setForm] = useState<ProxySettings | null>(settings);
  const [busy, setBusy] = useState(false);

  const [access, setAccess] = useState<AccessConfig | null>(null);
  const [accessBusy, setAccessBusy] = useState(false);

  useEffect(() => setForm(settings), [settings]);
  useEffect(() => { api.getAccess().then(setAccess).catch(() => {}); }, []);

  if (!form) return null;

  const set = (patch: Partial<ProxySettings>) => setForm({ ...form, ...patch });

  async function save() {
    if (!form) return;
    setBusy(true);
    try {
      await api.updateSettings({
        routing_mode: form.routing_mode,
        force_country: form.force_country,
        routing_ip_type: form.routing_ip_type,
        connection_enabled: form.connection_enabled,
        fixed_node_id: form.fixed_node_id,
      });
      push("ok", "策略已保存");
      onChanged();
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  async function saveAccess() {
    if (!access) return;
    setAccessBusy(true);
    try {
      const updated = await api.updateAccess({
        web_external_access: access.web_external_access,
        proxy_external_access: access.proxy_external_access,
      });
      setAccess(updated);
      push("ok", "外网访问设置已保存（即时生效）");
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setAccessBusy(false);
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

      <Card
        title="外网访问"
        actions={<button className="btn btn-primary" disabled={accessBusy || !access} onClick={saveAccess}>{accessBusy ? <Spinner /> : "保存"}</button>}
      >
        {!access ? (
          <div className="text-sm text-ink-3">加载中…</div>
        ) : (
          <div className="grid gap-3">
            <label className="flex items-center gap-3">
              <input type="checkbox" checked={access.web_external_access}
                onChange={(e) => setAccess({ ...access, web_external_access: e.target.checked })} />
              <span className="text-sm text-ink-2">允许网页后台外网访问</span>
            </label>
            <label className="flex items-center gap-3">
              <input type="checkbox" checked={access.proxy_external_access}
                onChange={(e) => setAccess({ ...access, proxy_external_access: e.target.checked })} />
              <span className="text-sm text-ink-2">允许代理端口外网访问</span>
            </label>
            {access.proxy_external_access && !access.proxy_auth_configured && (
              <div className="text-xs text-warn">
                ⚠ 代理外网访问已开启，但尚未设置代理用户名密码——为防止“开放代理”被滥用，外网客户端仍会被拒绝。
                请先配置 <code>FREE_PROXY_PROXY_USERNAME</code> / <code>FREE_PROXY_PROXY_PASSWORD</code> 并重启后生效。
              </div>
            )}
            <p className="text-xs text-ink-3">
              监听已绑定 <code>0.0.0.0</code>；本机与 SSH 隧道始终可用，开关即时生效、无需重启。
              网页后台有登录保护，默认允许外网；代理默认仅限本机，开启外网前请务必设置代理密码。
            </p>
          </div>
        )}
      </Card>
    </div>
  );
}
