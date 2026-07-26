import { StrictMode, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import * as api from "./api";
import { App } from "./App";
import { Login } from "./components/Login";
import { Toasts } from "./components/ui";
import "./styles.css";

type View = "loading" | "login" | "app";

function Root() {
  const [view, setView] = useState<View>("loading");

  useEffect(() => {
    api.setUnauthorizedHandler(() => setView("login"));
    api.authConfig().then(
      () => setView("app"),
      () => setView("login"),
    );
  }, []);

  if (view === "loading") {
    return (
      <div className="min-h-full grid place-items-center text-ink-3">
        <span className="inline-block w-6 h-6 border-2 border-rule-strong border-t-ink rounded-full animate-spin" />
      </div>
    );
  }
  if (view === "login") {
    return (
      <>
        <Login onSuccess={() => setView("app")} />
        <Toasts />
      </>
    );
  }
  return <App onLogout={() => setView("login")} />;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
