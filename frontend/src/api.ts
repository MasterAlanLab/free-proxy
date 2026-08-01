import type {
  AccessConfig, AppSettings, AuthConfig, GatewayStatus, Job, LogEntry, PoolStatistics, ProxyHealthResult,
  ProxyNodePage, ProxySettings, SystemDiagnostics, SystemStatus,
} from "./types";

const API = "./api/v1";

let onUnauthorized: (() => void) | null = null;
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {};
  if (init?.body) headers["Content-Type"] = "application/json";
  const res = await fetch(`${API}${path}`, { headers, ...init });
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  let body: unknown = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
  }
  if (res.status === 401) {
    onUnauthorized?.();
    throw new ApiError("未授权", 401);
  }
  if (!res.ok) {
    const detail = (body as { detail?: string; error?: string } | null);
    throw new ApiError(detail?.detail || detail?.error || `请求失败 (${res.status})`, res.status);
  }
  return body as T;
}

const post = (path: string, body?: unknown) =>
  request(path, { method: "POST", body: body ? JSON.stringify(body) : undefined });

// ---- auth ----
export const login = (username: string, password: string) =>
  request<{ ok: boolean }>("/auth/login", { method: "POST", body: JSON.stringify({ username, password }) });
export const logout = () => post("/auth/logout");
export const authConfig = () => request<AuthConfig>("/auth/config");
export const updateCredentials = (payload: Record<string, unknown>) =>
  request("/auth/credentials", { method: "PUT", body: JSON.stringify(payload) });

// ---- proxies ----
export function listProxies(params: Record<string, string | number | boolean>) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v !== "" && v !== undefined) q.set(k, String(v));
  return request<ProxyNodePage>(`/proxies?${q.toString()}`);
}
export const discover = () => post("/proxies/discover") as Promise<Job>;
export const refresh = () => post("/proxies/refresh") as Promise<Job>;
export const probeMany = (ids: string[]) => post("/proxies/probe", { ids }) as Promise<Job>;
export const probeOne = (id: string) => post(`/proxies/${id}/probe`) as Promise<Job>;
export const activate = (id: string) => post(`/proxies/${id}/activate`) as Promise<Job>;
export const toggleFavorite = (id: string) =>
  post(`/proxies/${id}/favorite`) as Promise<{ ok: boolean; favorite_node_ids: string[] }>;
export const configUrl = (id: string) => `${API}/proxies/${id}/config`;

// ---- gateway ----
export const gatewayStatus = () => request<GatewayStatus>("/gateway/status");
export const gatewayCheck = () => post("/gateway/check") as Promise<ProxyHealthResult>;
export const gatewayRotate = () => post("/gateway/rotate") as Promise<Job>;
export const gatewayDisconnect = () => request("/gateway/current", { method: "DELETE" });

// ---- pool / settings / system / logs ----
export const poolStats = () => request<PoolStatistics>("/pool/statistics");
export const getSettings = () => request<ProxySettings>("/settings");
export const updateSettings = (payload: Partial<ProxySettings>) =>
  request<ProxySettings>("/settings", { method: "PUT", body: JSON.stringify(payload) });
export const systemStatus = () => request<SystemStatus>("/system/status");
export const systemDiagnostics = () => request<SystemDiagnostics>("/system/diagnostics");
export const dnsRepair = () => post("/system/dns/repair");
export const getAccess = () => request<AccessConfig>("/system/access");
export const updateAccess = (payload: { web_external_access: boolean; proxy_external_access: boolean }) =>
  request<AccessConfig>("/system/access", { method: "PUT", body: JSON.stringify(payload) });
export const getSystemConfig = () => request<AppSettings>("/system/config");
export const updateSystemConfig = (settings: AppSettings, adminPassword: string, proxyPassword: string) =>
  request<{ ok: boolean; restart_needed: boolean; reauth_required: boolean; settings: AppSettings }>("/system/config", {
    method: "PUT", body: JSON.stringify({ settings, admin_password: adminPassword, proxy_password: proxyPassword }),
  });
export const getLogs = (params: Record<string, string | number>) => {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v !== "" && v !== undefined) q.set(k, String(v));
  return request<{ logs: LogEntry[] }>(`/logs?${q.toString()}`);
};
export const exportLogsUrl = (params: Record<string, string>) => {
  const q = new URLSearchParams(params);
  return `${API}/logs/export?${q.toString()}`;
};

// ---- jobs ----
export const getJob = (id: string) => request<Job>(`/jobs/${id}`);

export async function waitJob(id: string): Promise<Job> {
  for (;;) {
    const job = await getJob(id);
    if (["succeeded", "failed", "cancelled"].includes(job.status)) {
      if (job.status !== "succeeded") throw new ApiError(job.error || `任务 ${job.status}`, 200);
      return job;
    }
    await new Promise((r) => setTimeout(r, 700));
  }
}
