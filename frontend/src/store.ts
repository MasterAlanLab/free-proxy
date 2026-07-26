import { create } from "zustand";

export interface Toast {
  id: number;
  kind: "ok" | "error" | "info";
  message: string;
}

interface UIState {
  toasts: Toast[];
  push: (kind: Toast["kind"], message: string) => void;
  dismiss: (id: number) => void;
}

let seq = 1;

export const useUI = create<UIState>((set) => ({
  toasts: [],
  push: (kind, message) => {
    const id = seq++;
    set((s) => ({ toasts: [...s.toasts, { id, kind, message }] }));
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), 4200);
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));
