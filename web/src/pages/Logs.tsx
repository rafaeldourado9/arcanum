import { useEffect, useRef, useState } from "react";
import { api, type LogEntry, streamLogs } from "../api";
import { useI18n } from "../i18n";
import { Badge, Button, Card, PageHeader, type Tone } from "../ui";

const LEVELS = ["ALL", "INFO", "WARN", "ERROR"] as const;
type Level = (typeof LEVELS)[number];
const RANK: Record<LogEntry["level"], number> = { DEBUG: 0, INFO: 1, WARN: 2, ERROR: 3 };
const TONE: Record<LogEntry["level"], Tone> = { DEBUG: "neutral", INFO: "info", WARN: "warning", ERROR: "danger" };
const MAX = 1000;

export default function Logs() {
  const { t } = useI18n();
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [live, setLive] = useState(true);
  const [level, setLevel] = useState<Level>("ALL");
  const [connected, setConnected] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const liveRef = useRef(live);
  liveRef.current = live;

  useEffect(() => {
    api
      .logs(300)
      .then((r) => setEntries(r.entries))
      .catch(() => {});
    const ctrl = new AbortController();
    let retry: ReturnType<typeof setTimeout>;
    const open = () => {
      setConnected(true);
      streamLogs((e) => {
        if (!liveRef.current) return;
        setEntries((prev) => (prev.length && prev[prev.length - 1].seq >= e.seq ? prev : [...prev.slice(-(MAX - 1)), e]));
      }, ctrl.signal)
        .catch(() => {})
        .finally(() => {
          setConnected(false);
          if (!ctrl.signal.aborted) retry = setTimeout(open, 3000); // reconecta sozinho
        });
    };
    open();
    return () => {
      ctrl.abort();
      clearTimeout(retry);
    };
  }, []);

  useEffect(() => {
    if (live) bottomRef.current?.scrollIntoView({ block: "end" });
  }, [entries, live]);

  const shown = level === "ALL" ? entries : entries.filter((e) => RANK[e.level] >= RANK[level as LogEntry["level"]]);

  return (
    <>
      <PageHeader
        title={t("logsTitle")}
        description={t("logsDesc")}
        actions={
          <>
            <Badge tone={live && connected ? "success" : "neutral"} dot>
              {live && connected ? t("live") : t("paused")}
            </Badge>
            <Button size="sm" icon={live ? "pause" : "play"} onClick={() => setLive((v) => !v)}>
              {live ? t("pause") : t("resume")}
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setEntries([])}>
              {t("clear")}
            </Button>
          </>
        }
      />
      <div className="mb-3 inline-flex rounded-lg border border-line bg-surface p-0.5 text-[13px] font-semibold">
        {LEVELS.map((l) => (
          <button key={l} type="button" onClick={() => setLevel(l)} className={`rounded-md px-3 py-1 ${level === l ? "bg-surface-3 text-heading" : "text-fg-muted hover:text-fg"}`}>
            {l === "ALL" ? t("filterAll") : l}
          </button>
        ))}
      </div>
      <Card padded={false}>
        <div className="h-[62vh] overflow-auto bg-surface-2 font-mono text-[12.5px] leading-relaxed">
          {shown.length === 0 ? (
            <p className="p-5 font-sans text-sm text-fg-muted">{t("noLogs")}</p>
          ) : (
            <ul className="divide-y divide-line/60">
              {shown.map((e) => (
                <li key={e.seq} className="flex gap-3 px-4 py-1.5">
                  <span className="shrink-0 text-fg-subtle tabular-nums">{new Date(e.time).toLocaleTimeString()}</span>
                  <span className="w-14 shrink-0">
                    <Badge tone={TONE[e.level]}>{e.level}</Badge>
                  </span>
                  <span className="min-w-0 break-words text-fg">
                    {e.msg}
                    {e.attrs &&
                      Object.entries(e.attrs).map(([k, v]) => (
                        <span key={k} className="ml-2 text-fg-subtle">
                          {k}=<span className="text-fg-muted">{typeof v === "string" ? v : JSON.stringify(v)}</span>
                        </span>
                      ))}
                  </span>
                </li>
              ))}
            </ul>
          )}
          <div ref={bottomRef} />
        </div>
      </Card>
    </>
  );
}
