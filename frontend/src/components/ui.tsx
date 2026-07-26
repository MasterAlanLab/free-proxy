import type { ReactNode } from "react";
import { useUI } from "../store";

export function Card({ title, actions, children, className = "" }: {
  title?: ReactNode; actions?: ReactNode; children: ReactNode; className?: string;
}) {
  return (
    <section className={`card p-4 sm:p-5 ${className}`}>
      {(title || actions) && (
        <header className="flex items-center justify-between gap-3 mb-4">
          {title && <h2 className="text-base font-semibold text-ink">{title}</h2>}
          {actions && <div className="flex items-center gap-2 flex-wrap">{actions}</div>}
        </header>
      )}
      {children}
    </section>
  );
}

export function StatTile({ label, value, tone = "default" }: {
  label: string; value: ReactNode; tone?: "default" | "ok" | "warn" | "danger";
}) {
  const color = {
    default: "text-ink", ok: "text-ok", warn: "text-warn", danger: "text-danger",
  }[tone];
  return (
    <div className="card px-4 py-3">
      <div className="text-xs text-ink-3">{label}</div>
      <div className={`mt-1 text-xl font-semibold tabular-nums ${color}`}>{value}</div>
    </div>
  );
}

// Status text: verdict colors only (ready/unavailable/cooldown); categories
// like residential/mobile/hosting are neutral facts and stay ink.
const statusTones: Record<string, string> = {
  ready: "text-ok",
  unavailable: "text-danger",
  cooldown: "text-warn",
  probing: "text-ink-2",
  discovered: "text-ink-2",
  residential: "text-ink-2",
  mobile: "text-ink-2",
  hosting: "text-ink-2",
  unknown: "text-ink-3",
};

export function Badge({ label, tone }: { label: string; tone?: string }) {
  return <span className={`text-sm ${statusTones[tone ?? label] ?? "text-ink-2"}`}>{label}</span>;
}

export function Toasts() {
  const { toasts, dismiss } = useUI();
  return (
    <div className="fixed bottom-5 right-5 z-50 flex flex-col gap-2 w-[min(360px,90vw)]">
      {toasts.map((t) => (
        <div
          key={t.id}
          onClick={() => dismiss(t.id)}
          className="card px-4 py-3 text-sm cursor-pointer flex items-baseline gap-2.5"
        >
          <span className={`inline-block w-2 h-2 rounded-full shrink-0 ${
            t.kind === "ok" ? "bg-ok" : t.kind === "error" ? "bg-danger" : "bg-ink-3"
          }`} />
          <span>{t.message}</span>
        </div>
      ))}
    </div>
  );
}

export function Spinner() {
  return (
    <span className="inline-block w-4 h-4 border-2 border-rule-strong border-t-ink rounded-full animate-spin" />
  );
}
