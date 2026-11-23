import { type ChangeEvent, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { api, type WebhookConfig } from "../api";
import { usePoll } from "../hooks";
import { useI18n } from "../i18n";
import { Badge, Button, Card, EmptyState, ErrorBox, PageHeader, Spinner } from "../ui";

export default function Webhooks() {
  const { t } = useI18n();
  const sessions = usePoll(api.sessions, 6000);
  const fileRef = useRef<HTMLInputElement>(null);
  const [result, setResult] = useState<{ ok: boolean; text: string } | null>(null);

  const exportFile = async () => {
    const data = await api.exportWebhooks();
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `arcanum-webhooks-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(a.href);
  };

  const importFile = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setResult(null);
    try {
      const parsed = JSON.parse(await file.text()) as { webhooks?: Record<string, WebhookConfig> };
      const res = await api.importWebhooks(parsed.webhooks ?? {});
      const skipped = Object.entries(res.skipped);
      setResult({
        ok: skipped.length === 0,
        text: [t("importDone", { n: res.updated.length }), skipped.length ? `${t("importSkipped")}: ${skipped.map(([n, why]) => `${n} (${why})`).join(", ")}` : ""]
          .filter(Boolean)
          .join(" · "),
      });
      void sessions.refresh();
    } catch (err) {
      setResult({ ok: false, text: err instanceof Error ? err.message : String(err) });
    }
  };

  return (
    <>
      <PageHeader
        title={t("webhooksTitle")}
        description={t("webhooksDesc")}
        actions={
          <>
            <Button icon="download" onClick={exportFile}>
              {t("export")}
            </Button>
            <Button icon="upload" onClick={() => fileRef.current?.click()}>
              {t("import")}
            </Button>
            <input ref={fileRef} type="file" accept="application/json,.json" className="hidden" onChange={importFile} />
          </>
        }
      />
      {result && <p className={`mb-4 text-sm ${result.ok ? "text-success" : "text-danger"}`}>{result.text}</p>}
      {sessions.error && <ErrorBox message={sessions.error} onRetry={sessions.refresh} />}

      <Card padded={false}>
        {sessions.loading ? (
          <div className="p-6">
            <Spinner className="h-5 w-5" />
          </div>
        ) : !sessions.data?.length ? (
          <EmptyState icon="webhook" title={t("noSessions")} hint={t("noSessionsHint")} />
        ) : (
          <ul className="divide-y divide-line">
            {sessions.data.map((s) => (
              <li key={s.instanceName}>
                <Link to={`/sessions/${encodeURIComponent(s.instanceName)}`} className="flex flex-wrap items-center justify-between gap-3 px-5 py-3.5 hover:bg-surface-2">
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-heading">{s.instanceName}</p>
                    <p className="truncate font-mono text-xs text-fg-subtle">{s.webhook.url || t("notSet")}</p>
                  </div>
                  <Badge tone={s.webhook.enabled && s.webhook.url ? "success" : "neutral"} dot>
                    {s.webhook.enabled && s.webhook.url ? t("on") : t("off")}
                  </Badge>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </>
  );
}
