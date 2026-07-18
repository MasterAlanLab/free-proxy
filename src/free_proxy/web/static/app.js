const api = "./api/v1";
const selected = new Set();
let nodes = [];
let settings = null;
let gateway = null;

const element = (id) => document.querySelector(`#${id}`);

async function requestJson(path, options = {}) {
  const response = await fetch(`${api}${path}`, options);
  if (response.status === 204) return null;
  const payload = await response.json();
  if (response.status === 401) {
    window.location.reload();
    throw new Error("登录已失效");
  }
  if (!response.ok) throw new Error(payload.detail || payload.error || `请求失败: ${response.status}`);
  return payload;
}

async function waitForJob(jobId) {
  while (true) {
    const job = await requestJson(`/jobs/${jobId}`);
    element("job-message").textContent = job.status;
    if (["succeeded", "failed", "cancelled"].includes(job.status)) {
      if (job.status === "failed") throw new Error(job.error || "后台任务失败");
      return job.result;
    }
    await new Promise((resolve) => window.setTimeout(resolve, 800));
  }
}

async function loadAll() {
  const [system, page, currentSettings, gatewayStatus] = await Promise.all([
    requestJson("/system/status"),
    requestJson("/proxies?limit=500"),
    requestJson("/settings"),
    requestJson("/gateway/status"),
  ]);
  nodes = page.items;
  settings = currentSettings;
  gateway = gatewayStatus;
  renderStatus(system, gatewayStatus);
  renderNodes();
  renderSettings();
}

function renderStatus(system, gatewayStatus) {
  element("service-status").textContent = system.status === "running" ? "运行中" : system.status;
  element("node-count").textContent = String(system.nodes);
  element("active-node").textContent = gatewayStatus.active_node_id || "未连接";
  element("exit-ip").textContent = gatewayStatus.exit_ip || "-";
  element("active-latency").textContent = gatewayStatus.active_latency_ms ? `${gatewayStatus.active_latency_ms} ms` : "-";
  element("socks-listener").textContent = gatewayStatus.socks_listener;
  element("http-listener").textContent = gatewayStatus.http_listener;
}

function renderNodes() {
  const favorites = new Set(settings?.favorite_node_ids || []);
  const body = element("proxy-table-body");
  if (!nodes.length) {
    body.innerHTML = '<tr><td colspan="9" class="empty-state">尚未发现节点</td></tr>';
    return;
  }
  body.innerHTML = nodes.map((node) => `
    <tr class="${gateway?.active_node_id === node.id ? "active-row" : ""}">
      <td><input class="node-checkbox" type="checkbox" data-id="${escapeHtml(node.id)}" ${selected.has(node.id) ? "checked" : ""}></td>
      <td><button class="icon-action favorite-action" data-id="${escapeHtml(node.id)}">${favorites.has(node.id) ? "★" : "☆"}</button></td>
      <td>${escapeHtml(node.country || node.country_code || "未知")}</td>
      <td>${escapeHtml(node.ip_address)}</td>
      <td>${escapeHtml(node.transport.toUpperCase())}</td>
      <td>${escapeHtml(node.ip_type)}</td>
      <td>${node.latency_ms ? `${node.latency_ms} ms` : "-"}</td>
      <td>${escapeHtml(node.status)}</td>
      <td class="row-actions">
        <button class="small-action probe-action" data-id="${escapeHtml(node.id)}">测试</button>
        <button class="small-action activate-action" data-id="${escapeHtml(node.id)}">连接</button>
        <a class="small-action" href="${api}/proxies/${encodeURIComponent(node.id)}/config">配置</a>
      </td>
    </tr>`).join("");
  document.querySelectorAll(".node-checkbox").forEach((checkbox) => checkbox.addEventListener("change", () => {
    checkbox.checked ? selected.add(checkbox.dataset.id) : selected.delete(checkbox.dataset.id);
  }));
  document.querySelectorAll(".favorite-action").forEach((button) => button.addEventListener("click", () => toggleFavorite(button.dataset.id)));
  document.querySelectorAll(".probe-action").forEach((button) => button.addEventListener("click", () => probeNode(button.dataset.id)));
  document.querySelectorAll(".activate-action").forEach((button) => button.addEventListener("click", () => activateNode(button.dataset.id)));
}

function renderSettings() {
  element("routing-mode").value = settings.routing_mode;
  element("routing-ip-type").value = settings.routing_ip_type;
  element("fixed-node-id").value = settings.fixed_node_id || "";
  element("connection-enabled").checked = settings.connection_enabled;
  const countries = [...new Set(nodes.map((node) => node.country).filter(Boolean))].sort();
  element("force-country").innerHTML = '<option value="">不限</option>' + countries.map((country) => `<option value="${escapeHtml(country)}">${escapeHtml(country)}</option>`).join("");
  element("force-country").value = settings.force_country || "";
}

async function runJob(path, options = { method: "POST" }) {
  element("job-message").textContent = "pending";
  const job = await requestJson(path, options);
  const result = await waitForJob(job.id);
  await loadAll();
  return result;
}

async function refreshNodes() {
  await runJob("/proxies/refresh");
}

async function probeNode(nodeId) {
  await runJob(`/proxies/${encodeURIComponent(nodeId)}/probe`);
}

async function probeSelected() {
  if (!selected.size) throw new Error("请先选择节点");
  await runJob("/proxies/probe", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ids: [...selected] }),
  });
  selected.clear();
}

async function activateNode(nodeId) {
  await runJob(`/proxies/${encodeURIComponent(nodeId)}/activate`);
}

async function toggleFavorite(nodeId) {
  await requestJson(`/proxies/${encodeURIComponent(nodeId)}/favorite`, { method: "POST" });
  await loadAll();
}

async function disconnect() {
  await requestJson("/gateway/current", { method: "DELETE" });
  await loadAll();
}

async function checkHealth() {
  const result = await requestJson("/gateway/check", { method: "POST" });
  element("job-message").textContent = result.ok ? `出口 ${result.exit_ip}` : result.error;
  await loadAll();
}

async function saveSettings(event) {
  event.preventDefault();
  settings = await requestJson("/settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      routing_mode: element("routing-mode").value,
      force_country: element("force-country").value,
      routing_ip_type: element("routing-ip-type").value,
      connection_enabled: element("connection-enabled").checked,
      fixed_node_id: element("fixed-node-id").value || null,
    }),
  });
  element("job-message").textContent = "设置已保存";
  await loadAll();
}

async function loadCredentials() {
  const config = await requestJson("/auth/config");
  element("admin-username").value = config.username;
  element("admin-secret-path").value = config.secret_path;
  element("admin-host").value = config.host;
  element("admin-port").value = config.port;
}

async function saveCredentials(event) {
  event.preventDefault();
  const result = await requestJson("/auth/credentials", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      username: element("admin-username").value,
      password: element("admin-password").value,
      secret_path: element("admin-secret-path").value,
      host: element("admin-host").value,
      port: Number(element("admin-port").value),
    }),
  });
  element("job-message").textContent = result.restart_needed ? "配置已保存，服务即将重启" : "管理配置已保存";
}

async function loadLogs() {
  const params = new URLSearchParams();
  const level = element("log-level").value;
  const module = element("log-module").value.trim();
  if (level) params.set("level", level);
  if (module) params.set("module", module);
  const payload = await requestJson(`/logs?${params}`);
  const list = element("log-list");
  if (!payload.logs.length) {
    list.innerHTML = '<p class="empty-state">暂无日志</p>';
    return;
  }
  list.innerHTML = payload.logs.slice(-300).reverse().map((entry) => `<div class="log-entry"><time>${escapeHtml(entry.timestamp)}</time><strong>${escapeHtml(entry.level)}</strong><span>${escapeHtml(entry.module)}</span><p>${escapeHtml(entry.message)}</p></div>`).join("");
}

function exportLogs() {
  const params = new URLSearchParams();
  const level = element("log-level").value;
  const module = element("log-module").value.trim();
  if (level) params.set("level", level);
  if (module) params.set("module", module);
  window.location.assign(`${api}/logs/export?${params}`);
}

async function logout() {
  await requestJson("/auth/logout", { method: "POST" });
  window.location.reload();
}

function handle(action) {
  return async (...args) => {
    try { await action(...args); }
    catch (error) { element("job-message").textContent = error.message; }
  };
}

function escapeHtml(value) {
  return String(value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

element("refresh-button").addEventListener("click", handle(refreshNodes));
element("probe-selected-button").addEventListener("click", handle(probeSelected));
element("health-button").addEventListener("click", handle(checkHealth));
element("disconnect-button").addEventListener("click", handle(disconnect));
element("logout-button").addEventListener("click", handle(logout));
element("reload-logs-button").addEventListener("click", handle(loadLogs));
element("export-logs-button").addEventListener("click", exportLogs);
element("log-level").addEventListener("change", handle(loadLogs));
element("settings-form").addEventListener("submit", handle(saveSettings));
element("credentials-form").addEventListener("submit", handle(saveCredentials));

Promise.all([loadAll(), loadCredentials(), loadLogs()]).catch((error) => {
  element("service-status").textContent = "不可用";
  element("job-message").textContent = error.message;
});
window.setInterval(() => loadAll().catch(() => {}), 5000);
window.setInterval(() => loadLogs().catch(() => {}), 2500);
