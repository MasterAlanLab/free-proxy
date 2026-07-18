const API = "./api/v1";

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API}${path}`, init);
  if (response.status === 204) return undefined as T;
  const body = await response.json();
  if (response.status === 401) location.reload();
  if (!response.ok) throw new Error(body.detail || body.error || `请求失败 (${response.status})`);
  return body as T;
}

export async function waitJob(id: string): Promise<void> {
  for (;;) {
    const job = await api<{status: string; error?: string}>(`/jobs/${id}`);
    if (["succeeded", "failed", "cancelled"].includes(job.status)) {
      if (job.status !== "succeeded") throw new Error(job.error || `任务${job.status}`);
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 700));
  }
}
