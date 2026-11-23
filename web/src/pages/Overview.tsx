import { Link } from "react-router-dom";
import { api } from "../api";
import { usePoll } from "../hooks";
import { useI18n } from "../i18n";
import { Icon } from "../icons";
import { Card, EmptyState, ErrorBox, PageHeader, Spinner, StatTile, StatusBadge, formatDuration, formatPhone } from "../ui";

export default function Overview() {
  const { t } = useI18n();
  const stats = usePoll(api.stats, 5000);
  const sessions = usePoll(api.sessions, 5000);
  const s = stats.data;
  const fmt = (n: number) => n.toLocaleString();

  return (
    <>
      <PageHeader title={t("overviewTitle")} description={t("overviewDesc")} />
      {stats.error && <ErrorBox message={stats.error} onRetry={stats.refresh} />}

      {!s ? (
        !stats.error && <Spinner className="h-5 w-5" />
      ) : (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <StatTile label={t("connected")} value={s.sessions.connected ?? 0} hint={`${Object.values(s.sessions).reduce((a, b) => a + (b ?? 0), 0)} total`} />
            <StatTile label={t("waitingScan")} value={s.sessions.qr ?? 0} />
            <StatTile label={t("uptime")} value={formatDuration(s.uptimeSeconds)} />
            <StatTile label={t("version")} value={<span className="text-lg">{s.version}</span>} />
          </div>
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
            <StatTile label={t("received")} value={fmt(s.counters.messagesReceived)} />
            <StatTile label={t("sent")} value={fmt(s.counters.messagesSent)} />
            <StatTile label={t("sendErrors")} value={fmt(s.counters.messagesSendError)} tone={s.counters.messagesSendError ? "danger" : undefined} />
            <StatTile label={t("delivered")} value={fmt(s.counters.webhookDelivered)} />
            <StatTile label={t("failed")} value={fmt(s.counters.webhookFailed)} tone={s.counters.webhookFailed ? "danger" : undefined} />
          </div>

          <Card
            title={t("recentSessions")}
            padded={false}
            actions={
              <Link to="/sessions" className="inline-flex items-center gap-1 text-[13px] font-semibold text-fg-muted hover:text-fg">
                {t("viewAll")} <Icon name="arrowRight" className="h-3.5 w-3.5" />
              </Link>
            }
          >
            {sessions.data && sessions.data.length === 0 ? (
              <EmptyState icon="phone" title={t("noSessions")} hint={t("noSessionsHint")} />
            ) : (
              <ul className="divide-y divide-line">
                {(sessions.data ?? []).slice(0, 6).map((x) => (
                  <li key={x.instanceName}>
                    <Link to={`/sessions/${encodeURIComponent(x.instanceName)}`} className="flex items-center justify-between gap-3 px-5 py-3 hover:bg-surface-2">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold text-heading">{x.instanceName}</p>
                        <p className="text-xs text-fg-subtle">{formatPhone(x.phone) || "—"}</p>
                      </div>
                      <StatusBadge status={x.status} />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      )}
    </>
  );
}
