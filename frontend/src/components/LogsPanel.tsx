import { useCallback, useEffect, useState } from "react";
import * as api from "../api";
import type { LogEntry } from "../types";
import { useUI } from "../store";
import { Card, Spinner } from "./ui";

const levelColor: Record<string, string> = {
  ERROR: "text-danger", WARN: "text-warn", INFO: "text-ink-2", DEBUG: "text-ink-3",
};

export function LogsPanel() {
  const push = useUI((s) => s.push);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [level, setLevel] = useState("");
  const [module, setModule] = useState("");
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.getLogs({ level, module, limit: 500 });
      setLogs(res.logs);
    } catch (e) {
      push("error", (e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [level, module, push]);

  useEffect(() => {
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, [load]);

  return (
    <Card
      title="实时日志"
      actions={
        <>
          <select className="field w-auto" value={level} onChange={(e) => setLevel(e.target.value)}>
            <option value="">全部级别</option>
            <option value="INFO">INFO</option>
            <option value="WARN">WARN</option>
            <option value="ERROR">ERROR</option>
          </select>
          <input className="field w-auto" placeholder="模块过滤" value={module}
            onChange={(e) => setModule(e.target.value)} />
          <button className="btn" onClick={load} disabled={loading}>{loading ? <Spinner /> : "刷新"}</button>
          <a className="btn" href={api.exportLogsUrl({ level, module })}>导出</a>
        </>
      }
    >
      <div className="max-h-[560px] overflow-auto rounded-md bg-paper border border-rule font-mono text-xs">
        {logs.length === 0 && <div className="p-4 text-ink-3">暂无日志</div>}
        {logs.map((l, i) => (
          <div key={i} className="grid grid-cols-[150px_60px_120px_1fr] gap-2 px-3 py-1.5 border-b border-rule">
            <span className="text-ink-3">{l.timestamp}</span>
            <span className={levelColor[l.level] ?? "text-ink-2"}>{l.level}</span>
            <span className="text-ink-2 truncate">{l.module}</span>
            <span className="text-ink break-words">{l.message}</span>
          </div>
        ))}
      </div>
    </Card>
  );
}
