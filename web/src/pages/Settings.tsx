import { docsUrl } from "../api";
import { useTheme } from "../hooks";
import { useI18n } from "../i18n";
import { Icon, type IconName } from "../icons";
import { Card, PageHeader } from "../ui";

export default function SettingsPage({ version }: { version: string }) {
  const { t, lang, setLang } = useI18n();
  const [theme, setTheme] = useTheme();

  const links: { href: string; label: string; icon: IconName }[] = [
    { href: "/docs", label: t("apiDocs"), icon: "terminal" },
    { href: docsUrl(lang), label: t("guide"), icon: "book" },
    { href: "https://github.com/rafaeldourado9/arcanum", label: t("repo"), icon: "globe" },
  ];

  return (
    <>
      <PageHeader title={t("settingsTitle")} description={t("settingsDesc")} />
      <div className="grid gap-6 lg:grid-cols-2">
        <Card title={t("appearance")}>
          <div className="space-y-5">
            <Choice
              label={t("themeLight") + " / " + t("themeDark")}
              value={theme}
              options={[
                { value: "light", label: t("themeLight"), icon: "sun" },
                { value: "dark", label: t("themeDark"), icon: "moon" },
              ]}
              onChange={(v) => setTheme(v as "light" | "dark")}
            />
            <Choice
              label={t("language")}
              value={lang}
              options={[
                { value: "en", label: "English" },
                { value: "pt", label: "Português" },
              ]}
              onChange={(v) => setLang(v as "en" | "pt")}
            />
          </div>
        </Card>

        <Card title={t("about")}>
          <p className="mb-4 text-sm text-fg-muted">
            Arcanum <span className="font-semibold text-heading">{version}</span> · MIT
          </p>
          <ul className="space-y-1">
            {links.map((l) => (
              <li key={l.href}>
                <a href={l.href} target="_blank" rel="noreferrer" className="flex items-center gap-2.5 rounded-lg px-2 py-2 text-sm font-medium text-fg hover:bg-surface-3">
                  <Icon name={l.icon} />
                  {l.label}
                  <Icon name="external" className="ml-auto h-3.5 w-3.5 text-fg-subtle" />
                </a>
              </li>
            ))}
          </ul>
          <div className="mt-5 flex gap-3 rounded-xl border border-line bg-surface-2 p-4">
            <Icon name="shield" className="mt-0.5 h-4 w-4 shrink-0 text-fg-muted" />
            <div>
              <p className="text-[13px] font-semibold text-heading">{t("securityTip")}</p>
              <p className="mt-0.5 text-[13px] text-fg-muted">{t("securityTipBody")}</p>
            </div>
          </div>
        </Card>
      </div>
    </>
  );
}

function Choice({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: { value: string; label: string; icon?: IconName }[];
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <p className="mb-2 text-[13px] font-semibold text-fg">{label}</p>
      <div className="inline-flex rounded-lg border border-line bg-surface-2 p-0.5 text-[13px] font-semibold">
        {options.map((o) => (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            className={`inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 ${value === o.value ? "bg-surface text-heading shadow-card" : "text-fg-muted hover:text-fg"}`}
          >
            {o.icon && <Icon name={o.icon} className="h-3.5 w-3.5" />}
            {o.label}
          </button>
        ))}
      </div>
    </div>
  );
}
