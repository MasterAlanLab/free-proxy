import { useEffect, useState } from "react";
import * as api from "../api";
import type { AppSettings } from "../types";
import { useUI } from "../store";
import { Card, Spinner } from "./ui";

type Section = keyof AppSettings;

export function SystemConfigPanel() {
  const push = useUI((s) => s.push);
  const [form, setForm] = useState<AppSettings | null>(null);
  const [adminPassword, setAdminPassword] = useState("");
  const [proxyPassword, setProxyPassword] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => { api.getSystemConfig().then(setForm).catch((e) => push("error", (e as Error).message)); }, [push]);
  if (!form) return <Card title="系统配置"><div className="text-sm text-ink-3">加载中…</div></Card>;

  const set = <S extends Section>(section: S, key: keyof AppSettings[S], value: unknown) =>
    setForm({ ...form, [section]: { ...form[section], [key]: value } });
  const num = <S extends Section>(section: S, key: keyof AppSettings[S]) => (e: React.ChangeEvent<HTMLInputElement>) =>
    set(section, key, Number(e.target.value));

  async function save() {
    setBusy(true);
    try {
      const result = await api.updateSystemConfig(form!, adminPassword, proxyPassword);
      setForm(result.settings);
      setAdminPassword(""); setProxyPassword("");
      push("ok", "系统配置已保存，服务将在 2 秒后重启");
    } catch (e) { push("error", (e as Error).message); }
    finally { setBusy(false); }
  }

  return <div className="grid gap-4">
    <Card title="后台与代理服务" actions={<button className="btn btn-primary" disabled={busy} onClick={save}>{busy ? <Spinner /> : "保存全部配置"}</button>}>
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <Field label="后台用户名"><input className="field mt-1" value={form.admin.username} onChange={(e) => set("admin", "username", e.target.value)} /></Field>
        <Field label={`后台新密码${form.admin.password_set ? "（留空保持）" : ""}`}><input type="password" className="field mt-1" value={adminPassword} onChange={(e) => setAdminPassword(e.target.value)} /></Field>
        <Field label="管理路径"><input className="field mt-1" value={form.admin.secret_path} onChange={(e) => set("admin", "secret_path", e.target.value)} /></Field>
        <Field label="网页端口"><input type="number" className="field mt-1" value={form.admin.web_port} onChange={num("admin", "web_port")} /></Field>
        <Field label="登录有效期（秒）"><input type="number" className="field mt-1" value={form.admin.session_ttl_seconds} onChange={num("admin", "session_ttl_seconds")} /></Field>
        <Check label="允许网页后台外网访问" checked={form.admin.web_external_access} onChange={(v) => set("admin", "web_external_access", v)} />
        <Field label="代理用户名"><input className="field mt-1" value={form.proxy.username} onChange={(e) => set("proxy", "username", e.target.value)} /></Field>
        <Field label={`代理新密码${form.proxy.password_set ? "（留空保持）" : ""}`}><input type="password" className="field mt-1" value={proxyPassword} onChange={(e) => setProxyPassword(e.target.value)} /></Field>
        <Field label="代理端口"><input type="number" className="field mt-1" value={form.proxy.port} onChange={num("proxy", "port")} /></Field>
        <Field label="最大连接数"><input type="number" className="field mt-1" value={form.proxy.max_connections} onChange={num("proxy", "max_connections")} /></Field>
        <Field label="连接超时（秒）"><input type="number" className="field mt-1" value={form.proxy.connect_timeout_seconds} onChange={num("proxy", "connect_timeout_seconds")} /></Field>
        <Field label="空闲超时（秒）"><input type="number" className="field mt-1" value={form.proxy.idle_timeout_seconds} onChange={num("proxy", "idle_timeout_seconds")} /></Field>
        <Field label="代理 DNS"><input className="field mt-1" value={form.proxy.dns_server} onChange={(e) => set("proxy", "dns_server", e.target.value)} /></Field>
        <Check label="启用代理服务" checked={form.proxy.enabled} onChange={(v) => set("proxy", "enabled", v)} />
        <Check label="允许代理端口外网访问" checked={form.proxy.external_access} onChange={(v) => set("proxy", "external_access", v)} />
      </div>
      <p className="text-xs text-ink-3 mt-4">监听地址固定为 <code>0.0.0.0</code>；外部代理访问需同时开启开关并配置代理用户名和密码。密码只保存 scrypt 哈希。</p>
    </Card>

    <Card title="节点发现">
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <Field label="每次发现节点上限"><input type="number" className="field mt-1" value={form.discovery.discovery_limit} onChange={num("discovery", "discovery_limit")} /></Field>
        <Field label="请求超时（秒）"><input type="number" className="field mt-1" value={form.discovery.request_timeout_seconds} onChange={num("discovery", "request_timeout_seconds")} /></Field>
        <Field label="IP 信息缓存（秒）"><input type="number" className="field mt-1" value={form.discovery.ip_info_cache_seconds} onChange={num("discovery", "ip_info_cache_seconds")} /></Field>
        <Field label="VPN Gate API" wide><input className="field mt-1" value={form.discovery.vpngate_api_url} onChange={(e) => set("discovery", "vpngate_api_url", e.target.value)} /></Field>
        <Field label="IP 信息 API" wide><input className="field mt-1" value={form.discovery.ip_info_api_url} onChange={(e) => set("discovery", "ip_info_api_url", e.target.value)} /></Field>
      </div>
    </Card>

    <Card title="检测与维护">
      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Check label="启用自动维护" checked={form.maintenance.enabled} onChange={(v) => set("maintenance", "enabled", v)} />
        <N label="完整维护间隔（秒）" value={form.maintenance.maintenance_interval_seconds} change={num("maintenance", "maintenance_interval_seconds")} />
        <N label="健康检查间隔（秒）" value={form.maintenance.health_check_interval_seconds} change={num("maintenance", "health_check_interval_seconds")} />
        <N label="活动节点延迟检测（秒）" value={form.maintenance.active_ping_interval_seconds} change={num("maintenance", "active_ping_interval_seconds")} />
        <N label="断线重试（秒）" value={form.maintenance.disconnected_retry_seconds} change={num("maintenance", "disconnected_retry_seconds")} />
        <N label="探测并发数" value={form.maintenance.max_probe_concurrency} change={num("maintenance", "max_probe_concurrency")} />
        <N label="首次连接检测数" value={form.maintenance.initial_connect_test_limit} change={num("maintenance", "initial_connect_test_limit")} />
        <N label="手动检测节点上限" value={form.maintenance.manual_test_node_limit} change={num("maintenance", "manual_test_node_limit")} />
        <N label="OpenVPN 测试超时" value={form.maintenance.openvpn_test_timeout_seconds} change={num("maintenance", "openvpn_test_timeout_seconds")} />
        <N label="OpenVPN 连接超时" value={form.maintenance.openvpn_connect_timeout_seconds} change={num("maintenance", "openvpn_connect_timeout_seconds")} />
        <N label="无效节点退避（秒）" value={form.maintenance.invalid_backoff_seconds} change={num("maintenance", "invalid_backoff_seconds")} />
        <N label="历史节点保留（秒）" value={form.maintenance.stale_node_grace_seconds} change={num("maintenance", "stale_node_grace_seconds")} />
      </div>
    </Card>

    <Card title="网络与路由">
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <Check label="启用 DNS 修复" checked={form.network.dns_repair_enabled} onChange={(v) => set("network", "dns_repair_enabled", v)} />
        <Check label="严格反向路径过滤" checked={form.network.routing_strict_rp_filter} onChange={(v) => set("network", "routing_strict_rp_filter", v)} />
        <Field label="DNS 修复服务器"><input className="field mt-1" value={form.network.dns_repair_servers} onChange={(e) => set("network", "dns_repair_servers", e.target.value)} /></Field>
        <N label="路由设置重试次数" value={form.network.routing_setup_retries} change={num("network", "routing_setup_retries")} />
        <N label="路由重试间隔（秒）" value={form.network.routing_retry_interval_seconds} change={num("network", "routing_retry_interval_seconds")} />
      </div>
    </Card>
  </div>;
}

function Field({ label, wide, children }: { label: string; wide?: boolean; children: React.ReactNode }) {
  return <label className={`block ${wide ? "sm:col-span-2 lg:col-span-3" : ""}`}><span className="text-sm text-ink-2">{label}</span>{children}</label>;
}
function N({ label, value, change }: { label: string; value: number; change: (e: React.ChangeEvent<HTMLInputElement>) => void }) {
  return <Field label={label}><input type="number" className="field mt-1" value={value} onChange={change} /></Field>;
}
function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return <label className="flex items-center gap-3 self-end min-h-10"><input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} /><span className="text-sm text-ink-2">{label}</span></label>;
}
