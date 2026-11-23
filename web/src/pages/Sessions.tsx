import { type FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { api, type Session } from "../api";
import { usePoll } from "../hooks";
import { useI18n } from "../i18n";
import { Button, Card, EmptyState, ErrorBox, Field, Input, Modal, PageHeader, Spinner, StatusBadge, formatPhone, hostOf } from "../ui";
import ConnectModal from "./ConnectModal";

export default function Sessions() {
  const { t } = useI18n();
  const sessions = usePoll(api.sessions, 4000);
  const [creating, setCreating] = useState(false);
  const [connecting, setConnecting] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);

  return (
    <>
      <PageHeader
        title={t("sessionsTitle")}
        description={t("sessionsDesc")}
        actions={
          <Button variant="primary" icon="plus" onClick={() => setCreating(true)}>
            {t("newSession")}
          </Button>
        }
      />
      {sessions.error && <ErrorBox message={sessions.error} onRetry={sessions.refresh} />}

      <Card padded={false}>
        {sessions.loading ? (
          <div className="p-6">
            <Spinner className="h-5 w-5" />
          </div>
        ) : sessions.data && sessions.data.length === 0 ? (
          <EmptyState
            icon="phone"
            title={t("noSessions")}
            hint={t("noSessionsHint")}
            action={
              <Button variant="primary" icon="plus" onClick={() => setCreating(true)}>
                {t("newSession")}
              </Button>
            }
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-line text-left text-xs font-semibold text-fg-subtle">
                  <th className="px-5 py-3">{t("name")}</th>
                  <th className="px-5 py-3">{t("status")}</th>
                  <th className="hidden px-5 py-3 sm:table-cell">{t("phone")}</th>
                  <th className="hidden px-5 py-3 md:table-cell">{t("webhook")}</th>
                  <th className="px-5 py-3 text-right">{t("actions")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {(sessions.data ?? []).map((s) => (
                  <Row key={s.instanceName} s={s} onConnect={setConnecting} onDelete={setDeleting} onChanged={sessions.refresh} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <CreateModal
        open={creating}
        onClose={() => setCreating(false)}
        onCreated={(name) => {
          setCreating(false);
          void sessions.refresh();
          setConnecting(name);
        }}
      />
      <ConnectModal
        name={connecting}
        onClose={() => {
          setConnecting(null);
          void sessions.refresh();
        }}
      />
      <DeleteModal
        name={deleting}
        onClose={() => setDeleting(null)}
        onDeleted={() => {
          setDeleting(null);
          void sessions.refresh();
        }}
      />
    </>
  );
}

function Row({ s, onConnect, onDelete, onChanged }: { s: Session; onConnect: (n: string) => void; onDelete: (n: string) => void; onChanged: () => void }) {
  const { t } = useI18n();
  const [busy, setBusy] = useState(false);
  const run = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await fn();
    } finally {
      setBusy(false);
      onChanged();
    }
  };
  return (
    <tr className="hover:bg-surface-2">
      <td className="px-5 py-3">
        <Link to={`/sessions/${encodeURIComponent(s.instanceName)}`} className="font-semibold text-heading hover:underline">
          {s.instanceName}
        </Link>
      </td>
      <td className="px-5 py-3">
        <StatusBadge status={s.status} />
      </td>
      <td className="hidden px-5 py-3 text-fg-muted tabular-nums sm:table-cell">{formatPhone(s.phone) || "—"}</td>
      <td className="hidden max-w-[16rem] truncate px-5 py-3 text-fg-muted md:table-cell">
        {s.webhook.url ? <span title={s.webhook.url}>{hostOf(s.webhook.url)}</span> : t("notSet")}
      </td>
      <td className="px-5 py-3">
        <div className="flex justify-end gap-1.5">
          {s.status !== "connected" ? (
            <Button size="sm" variant="primary" icon="qr" onClick={() => onConnect(s.instanceName)}>
              {t("connect")}
            </Button>
          ) : (
            <Button size="sm" icon="power" disabled={busy} onClick={() => run(() => api.logoutSession(s.instanceName))}>
              <span className="hidden lg:inline">{t("disconnect")}</span>
            </Button>
          )}
          <Button size="sm" variant="ghost" icon="refresh" disabled={busy} aria-label={t("restart")} title={t("restart")} onClick={() => run(() => api.restart(s.instanceName))} />
          <Button size="sm" variant="ghost" icon="trash" aria-label={t("delete")} title={t("delete")} onClick={() => onDelete(s.instanceName)} />
        </div>
      </td>
    </tr>
  );
}

function CreateModal({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: (name: string) => void }) {
  const { t } = useI18n();
  const [name, setName] = useState("");
  const [webhook, setWebhook] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const valid = /^[A-Za-z0-9_-]{1,64}$/.test(name);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.create(name, webhook.trim() || undefined);
      const created = name;
      setName("");
      setWebhook("");
      onCreated(created);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal open={open} title={t("newSession")} onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <Field label={t("name")} hint={t("nameHint")}>
          <Input autoFocus value={name} onChange={(e) => setName(e.target.value)} placeholder="sales" maxLength={64} />
        </Field>
        <Field label={t("webhookOptional")}>
          <Input type="url" value={webhook} onChange={(e) => setWebhook(e.target.value)} placeholder="https://your-app.example.com/webhooks/whatsapp" />
        </Field>
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" onClick={onClose}>
            {t("cancel")}
          </Button>
          <Button type="submit" variant="primary" disabled={!valid || busy}>
            {t("create")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function DeleteModal({ name, onClose, onDeleted }: { name: string | null; onClose: () => void; onDeleted: () => void }) {
  const { t } = useI18n();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const confirm = async () => {
    if (!name) return;
    setBusy(true);
    setError(null);
    try {
      await api.remove(name);
      onDeleted();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      open={!!name}
      title={t("deleteTitle")}
      onClose={onClose}
      footer={
        <>
          <Button onClick={onClose}>{t("cancel")}</Button>
          <Button variant="danger" icon="trash" disabled={busy} onClick={confirm}>
            {t("delete")}
          </Button>
        </>
      }
    >
      <p className="text-sm text-fg-muted">{t("deleteConfirm", { name: name ?? "" })}</p>
      {error && <p className="mt-3 text-sm text-danger">{error}</p>}
    </Modal>
  );
}
