import { useCallback, useEffect, useRef, useState } from "react";

/** Carrega dados e recarrega a cada `intervalMs` (0 = so uma vez). */
export function usePoll<T>(load: () => Promise<T>, intervalMs = 0) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const loadRef = useRef(load);
  loadRef.current = load;

  const refresh = useCallback(async () => {
    try {
      setData(await loadRef.current());
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void refresh();
    if (!intervalMs) return;
    const id = setInterval(() => {
      if (document.visibilityState === "visible") void refresh();
    }, intervalMs);
    return () => clearInterval(id);
  }, [refresh, intervalMs]);

  return { data, error, refresh, loading: data === null && error === null };
}

export type Theme = "light" | "dark";

export function useTheme(): [Theme, (t: Theme) => void] {
  const [theme, setThemeState] = useState<Theme>(() => (document.documentElement.classList.contains("dark") ? "dark" : "light"));
  const setTheme = useCallback((t: Theme) => {
    document.documentElement.classList.toggle("dark", t === "dark");
    try {
      localStorage.setItem("arcanum.theme", t);
    } catch {
      /* ok */
    }
    setThemeState(t);
  }, []);
  return [theme, setTheme];
}
