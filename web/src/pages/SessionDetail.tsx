import { type FormEvent, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api, type Delivery, type Session, type Settings } from "../api";
import { usePoll } from "../hooks";
import { useI18n } from "../i18n";
import { Icon } from "../icons";
import { Badge, Button, Card, ErrorBox, Field, Input, PageHeader, Spinner, StatusBadge, Toggle, formatPhone } from "../ui";
import ConnectModal from "./ConnectModal";

export default function SessionDetail() {
  const { t } = useI18n();
  const { name = "" } = useParams();
  const navigate = useNavigate();
  const sessions = usePoll(api.sessions, 4000);
  const session = sessions.data?.find((s) => s.instanceName === name);
  const [connecting, setConnecting] = useState(false);

  useEffect(() => {
    if (sessions.data && !session) navigate("/sessions", { replace: true });
  }, [sessions.data, session, navigate]);

  if (!session) {
    return sessions.error ? <ErrorBox message={sessions.error} onRetry={sessions.refresh} /> : <Spinner className="h-5 w-5" />;
  }

  return (
    <>
      <Link to="/sessions" className="mb-3 inline-flex items-center gap-1.5 text-[13px] font-medium text-fg-subtle hover:text-fg">
        <Icon name="arrowLeft" className="h-3.5 w-3.5" />
        {t("back")}
      </Link>
      <PageHeader
        title={
          <span className="flex flex-wrap items-center gap-3">
            {session.instanceName} <StatusBadge status={session.status} />
          </span>
        }
        description={[formatPhone(session.phone), session.createdAt && `${t("created")} ${new Date(session.createdAt).toLocaleDateString()}`].filter(Boolean).join(" · ")}
        actions={
          session.status !== "connected" ? (
            <Button variant="primary" icon="qr" onClick={() => setConnecting(true)}>
              {t("connect")}
            </Button>
          ) : (
            <Button icon="refresh" onClick={() => api.restart(name).then(sessions.refresh)}>
              {t("restart")}
            </Button>
          )
        }
      />
      <div className="grid gap-6 lg:grid-cols-[1fr_20rem]">
        <div className="space-y-6">
          <WebhookCard session={session} onSaved={sessions.refresh} />
          <DeliveriesCard name={name} />
        </div>
        <SettingsCard session={session} onSaved={sessions.refresh} />
      </div>
      <ConnectModal
        name={connecting ? name : null}
        onClose={() => {
          setConnecting(false);
          void sessions.refresh();
        }}
      />
    </>
  );
}

function WebhookCard({ session, onSaved }: { session: Session; onSaved: () => void }) {
  const { t } = useI18n();
  const name = session.instanceName;
  const [url, setUrl] = useState(session.webhook.url);
  const [enabled, setEnabled] = useState(session.webhook.enabled);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);
  const dirty = url !== session.webhook.url || enabled !== session.webhook.enabled;

  const save = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setMessage(null);
    try {
      await api.setWebhook(name, { url: url.trim(), enabled });
      setMessage({ ok: true, text: t("saved") });
      onSaved();
    } catch (err) {
      setMessage({ ok: false, text: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  };

  const test = async () => {
    setBusy(true);
    setMessage(null);
    try {
      const d = await api.testWebhook(name);
      setMessage(d.ok ? { ok: true, text: t("testOk", { status: d.status, ms: d.durationMs }) } : { ok: false, text: t("testFail", { error: d.error || `HTTP ${d.status}` }) });
    } catch (err) {
      setMessage({ ok: false, text: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card title={t("webhook")} description={t("webhookDesc")}>
      <form onSubmit={save} className="space-y-4">
        <Field label={t("url")}>
          <Input type="url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://your-app.example.com/webhooks/whatsapp" />
        </Field>
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-fg">{t("enabled")}</span>
          <Toggle checked={enabled} label={t("enabled")} onChange={setEnabled} />
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button type="submit" variant="primary" disabled={busy || !dirty}>
            {t("save")}
          </Button>
          <Button type="button" icon="send" disabled={busy || !session.webhook.url || dirty} onClick={test}>
            {t("sendTest")}
          </Button>
          {message && <span className={`text-sm ${message.ok ? "text-success" : "text-danger"}`}>{message.text}</span>}
        </div>
      </form>
    </Card>
  );
}

function DeliveriesCard({ name }: { name: string }) {
  const { t } = useI18n();
  const deliveries = usePoll(() => api.deliveries(name), 5000);
  const list: Delivery[] = deliveries.data?.deliveries ?? [];
  return (
    <Card title={t("deliveries")} padded={false}>
      {list.length === 0 ? (
        <p className="px-5 py-6 text-sm text-fg-muted">{t("noDeliveries")}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-line text-left text-xs font-semibold text-fg-subtle">
                <th className="px-5 py-2.5">{t("time")}</th>
                <th className="px-5 py-2.5">{t("event")}</th>
                <th className="px-5 py-2.5">{t("result")}</th>
                <th className="px-5 py-2.5 text-right">{t("duration")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {list.slice(0, 25).map((d, i) => (
                <tr key={`${d.time}-${i}`}>
                  <td className="px-5 py-2.5 text-fg-muted tabular-nums">{new Date(d.time).toLocaleTimeString()}</td>
                  <td className="px-5 py-2.5 font-mono text-xs">{d.event}</td>
                  <td className="px-5 py-2.5">
                    {d.ok ? (
                      <Badge tone="success">HTTP {d.status}</Badge>
                    ) : (
                      <span title={d.error}>
                        <Badge tone="danger">{d.status ? `HTTP ${d.status}` : "error"}</Badge>
                      </span>
                    )}
                  </td>
                  <td className="px-5 py-2.5 text-right text-fg-muted tabular-nums">{d.durationMs} ms</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

function SettingsCard({ session, onSaved }: { session: Session; onSaved: () => void }) {
  const { t } = useI18n();
  const [s, setS] = useState<Settings>(session.settings);
  const update = async (patch: Partial<Settings>) => {
    setS((prev) => ({ ...prev, ...patch }));
    try {
      setS(await api.setSettings(session.instanceName, patch));
      onSaved();
    } catch {
      setS(session.settings);
    }
  };
  const planned: { key: keyof Settings; label: string }[] = [
    { key: "rejectCalls", label: t("rejectCalls") },
    { key: "alwaysOnline", label: t("alwaysOnline") },
    { key: "readMessages", label: t("readMessages") },
  ];
  return (
    <Card title={t("settings")}>
      <div className="space-y-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-fg">{t("groupsIgnore")}</p>
            <p className="mt-0.5 text-xs text-fg-subtle">{t("groupsIgnoreHint")}</p>
          </div>
          <Toggle checked={s.groupsIgnore} label={t("groupsIgnore")} onChange={(v) => update({ groupsIgnore: v })} />
        </div>
        <div className="border-t border-line pt-4">
          <p className="mb-3 flex items-center gap-2 text-xs font-semibold text-fg-subtle">
            <Badge>{t("planned")}</Badge> {t("plannedHint")}
          </p>
          <div className="space-y-3">
            {planned.map((p) => (
              <div key={p.key} className="flex items-center justify-between gap-4">
                <span className="text-sm text-fg-muted">{p.label}</span>
                <Toggle checked={s[p.key]} disabled label={p.label} onChange={() => {}} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </Card>
  );
}
