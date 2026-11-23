import { type FormEvent, useState } from "react";
import { api } from "../api";
import { useTheme } from "../hooks";
import { useI18n } from "../i18n";
import { Icon } from "../icons";
import { Button, Field, Input } from "../ui";

export default function Login({ onDone }: { onDone: () => void }) {
  const { t, lang, setLang } = useI18n();
  const [theme, setTheme] = useTheme();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.login(password);
      setPassword("");
      onDone();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen flex-col items-center justify-center px-4">
      <div className="absolute top-4 right-4 flex items-center gap-1">
        <button
          type="button"
          onClick={() => setLang(lang === "pt" ? "en" : "pt")}
          className="rounded-md px-2 py-1 text-xs font-semibold text-fg-muted hover:bg-surface-3"
        >
          {lang === "pt" ? "EN" : "PT"}
        </button>
        <button
          type="button"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          className="rounded-md p-1.5 text-fg-muted hover:bg-surface-3"
          aria-label={theme === "dark" ? t("themeLight") : t("themeDark")}
        >
          <Icon name={theme === "dark" ? "sun" : "moon"} />
        </button>
      </div>

      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center text-center">
          <img src="/logo.svg" alt="" className="mb-4 h-12 w-12 dark:invert" />
          <h1 className="text-xl font-bold tracking-tight text-heading">{t("loginTitle")}</h1>
          <p className="mt-2 text-sm text-fg-muted">{t("loginHint")}</p>
        </div>
        <form onSubmit={submit} className="space-y-4 rounded-2xl border border-line bg-surface p-6 shadow-card">
          <Field label={t("password")} error={error}>
            <Input type="password" autoComplete="current-password" autoFocus required value={password} onChange={(e) => setPassword(e.target.value)} />
          </Field>
          <Button type="submit" variant="primary" className="w-full" disabled={busy || !password}>
            {busy ? t("signingIn") : t("signIn")}
          </Button>
        </form>
      </div>
    </div>
  );
}
