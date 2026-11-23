import { useEffect, useState } from "react";
import { Navigate, NavLink, Outlet, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { api, setUnauthorizedHandler } from "./api";
import { type Dict, useI18n } from "./i18n";
import { Icon, type IconName } from "./icons";
import { Spinner } from "./ui";
import Login from "./pages/Login";
import Logs from "./pages/Logs";
import Overview from "./pages/Overview";
import SessionDetail from "./pages/SessionDetail";
import Sessions from "./pages/Sessions";
import SettingsPage from "./pages/Settings";
import Webhooks from "./pages/Webhooks";

type AuthState = "checking" | "in" | "out";

export default function App() {
  const [auth, setAuth] = useState<AuthState>("checking");
  const [version, setVersion] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    setUnauthorizedHandler(() => setAuth("out"));
    api
      .me()
      .then((r) => {
        setVersion(r.version);
        setAuth(r.authenticated ? "in" : "out");
      })
      .catch(() => setAuth("out"));
  }, []);

  if (auth === "checking") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner className="h-6 w-6" />
      </div>
    );
  }

  return (
    <Routes>
      <Route
        path="/login"
        element={
          auth === "in" ? (
            <Navigate to="/" replace />
          ) : (
            <Login
              onDone={() => {
                setAuth("in");
                navigate("/", { replace: true });
              }}
            />
          )
        }
      />
      <Route element={auth === "in" ? <Layout version={version} onSignOut={() => setAuth("out")} /> : <Navigate to="/login" replace />}>
        <Route index element={<Overview />} />
        <Route path="sessions" element={<Sessions />} />
        <Route path="sessions/:name" element={<SessionDetail />} />
        <Route path="webhooks" element={<Webhooks />} />
        <Route path="logs" element={<Logs />} />
        <Route path="settings" element={<SettingsPage version={version} />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}

const NAV: { to: string; label: keyof Dict; icon: IconName; end?: boolean }[] = [
  { to: "/", label: "navOverview", icon: "grid", end: true },
  { to: "/sessions", label: "navSessions", icon: "phone" },
  { to: "/webhooks", label: "navWebhooks", icon: "webhook" },
  { to: "/logs", label: "navLogs", icon: "terminal" },
  { to: "/settings", label: "navSettings", icon: "settings" },
];

function Layout({ version, onSignOut }: { version: string; onSignOut: () => void }) {
  const { t } = useI18n();
  const [menuOpen, setMenuOpen] = useState(false);
  const location = useLocation();
  useEffect(() => setMenuOpen(false), [location.pathname]);

  const signOut = async () => {
    await api.logout().catch(() => {});
    onSignOut();
  };

  const nav = (
    <nav className="flex h-full flex-col">
      <div className="flex items-center gap-2.5 px-5 py-5">
        <img src="/logo.svg" alt="" className="h-7 w-7 dark:invert" />
        <div className="leading-tight">
          <p className="text-[15px] font-bold tracking-tight text-heading">Arcanum</p>
          <p className="text-[11px] text-fg-subtle">{t("appTagline")}</p>
        </div>
      </div>
      <ul className="flex-1 space-y-0.5 px-3">
        {NAV.map((item) => (
          <li key={item.to}>
            <NavLink
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                `flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  isActive ? "bg-surface-3 text-heading" : "text-fg-muted hover:bg-surface-3 hover:text-fg"
                }`
              }
            >
              <Icon name={item.icon} />
              {t(item.label)}
            </NavLink>
          </li>
        ))}
        <li>
          <a
            href="https://arcanum.rafaeldourado.tech"
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-3 hover:text-fg"
          >
            <Icon name="book" />
            {t("navDocs")}
            <Icon name="external" className="ml-auto h-3.5 w-3.5 text-fg-subtle" />
          </a>
        </li>
      </ul>
      <div className="border-t border-line px-3 py-3">
        <button
          type="button"
          onClick={signOut}
          className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-3 hover:text-fg"
        >
          <Icon name="logout" />
          {t("signOut")}
        </button>
        {version && <p className="px-3 pt-2 text-[11px] text-fg-subtle">Arcanum {version}</p>}
      </div>
    </nav>
  );

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[15rem_1fr]">
      <aside className="hidden border-r border-line bg-surface lg:sticky lg:top-0 lg:block lg:h-screen">{nav}</aside>

      <div className="flex items-center justify-between border-b border-line bg-surface px-4 py-3 lg:hidden">
        <div className="flex items-center gap-2">
          <img src="/logo.svg" alt="" className="h-6 w-6 dark:invert" />
          <span className="font-bold text-heading">Arcanum</span>
        </div>
        <button type="button" onClick={() => setMenuOpen((v) => !v)} className="rounded-md p-1.5 text-fg-muted hover:bg-surface-3" aria-label="Menu">
          <Icon name={menuOpen ? "x" : "menu"} className="h-5 w-5" />
        </button>
      </div>
      {menuOpen && <div className="border-b border-line bg-surface lg:hidden">{nav}</div>}

      <main className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-8 sm:py-8">
        <Outlet />
      </main>
    </div>
  );
}
