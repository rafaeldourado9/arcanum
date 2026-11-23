/**
 * Componentes do painel (mesma linha do ui.tsx do EasySun): cabecalho de pagina, cartao,
 * indicador, selo, botao, campo, chave liga/desliga, modal e estados vazios.
 * Cores so por token semantico (index.css): o tema escuro vem de graca.
 */
import { type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode, useEffect, useRef } from "react";
import type { Status } from "./api";
import { useI18n } from "./i18n";
import { Icon, type IconName } from "./icons";

export type Tone = "neutral" | "info" | "success" | "warning" | "danger";

const TONE: Record<Tone, string> = {
  neutral: "bg-surface-3 text-fg-muted",
  info: "bg-info-soft text-info",
  success: "bg-success-soft text-success",
  warning: "bg-warning-soft text-warning",
  danger: "bg-danger-soft text-danger",
};
const DOT: Record<Tone, string> = {
  neutral: "bg-fg-subtle",
  info: "bg-info",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
};

export function PageHeader({ title, description, actions }: { title: ReactNode; description?: ReactNode; actions?: ReactNode }) {
  return (
    <header className="mb-6 flex flex-wrap items-start justify-between gap-3">
      <div className="min-w-0 space-y-1">
        <h1 className="text-xl font-bold tracking-tight text-heading sm:text-2xl">{title}</h1>
        {description && <p className="max-w-2xl text-sm text-fg-muted">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </header>
  );
}

export function Card({
  title,
  description,
  actions,
  children,
  padded = true,
}: {
  title?: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  padded?: boolean;
}) {
  return (
    <section className="rounded-xl border border-line bg-surface shadow-card">
      {(title || actions) && (
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-line px-5 py-4">
          <div className="min-w-0">
            {title && <h2 className="text-[15px] font-semibold text-heading">{title}</h2>}
            {description && <p className="mt-0.5 text-[13px] text-fg-muted">{description}</p>}
          </div>
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
      )}
      <div className={padded ? "p-5" : ""}>{children}</div>
    </section>
  );
}

export function Badge({ tone = "neutral", dot = false, children }: { tone?: Tone; dot?: boolean; children: ReactNode }) {
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ${TONE[tone]}`}>
      {dot && <span className={`h-1.5 w-1.5 rounded-full ${DOT[tone]}`} />}
      {children}
    </span>
  );
}

const STATUS_TONE: Record<Status, Tone> = { connected: "success", qr: "warning", connecting: "info", disconnected: "neutral" };

export function StatusBadge({ status }: { status: Status }) {
  const { t } = useI18n();
  const label = { connected: t("connected"), qr: t("waitingScan"), connecting: t("connecting"), disconnected: t("disconnected") }[status];
  return (
    <Badge tone={STATUS_TONE[status]} dot>
      {label}
    </Badge>
  );
}

export function StatTile({ label, value, hint, tone }: { label: string; value: ReactNode; hint?: ReactNode; tone?: Tone }) {
  return (
    <div className="rounded-xl border border-line bg-surface p-4 shadow-card">
      <p className="text-[13px] font-medium text-fg-muted">{label}</p>
      <p className={`mt-1.5 text-2xl font-bold tracking-tight tabular-nums ${tone === "danger" ? "text-danger" : "text-heading"}`}>{value}</p>
      {hint && <p className="mt-1 text-xs text-fg-subtle">{hint}</p>}
    </div>
  );
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  size?: "sm" | "md";
  icon?: IconName;
};

export function Button({ variant = "secondary", size = "md", icon, className = "", children, ...rest }: ButtonProps) {
  const base =
    "inline-flex items-center justify-center gap-2 rounded-lg font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-50";
  const sizes = { sm: "h-8 px-2.5 text-[13px]", md: "h-9 px-3.5 text-sm" };
  const variants = {
    primary: "bg-strong text-strong-fg hover:opacity-90",
    secondary: "border border-line-strong bg-surface text-fg hover:bg-surface-3",
    ghost: "text-fg-muted hover:bg-surface-3 hover:text-fg",
    danger: "border border-line-strong bg-surface text-danger hover:bg-danger-soft",
  };
  return (
    <button className={`${base} ${sizes[size]} ${variants[variant]} ${className}`} {...rest}>
      {icon && <Icon name={icon} className="h-4 w-4" />}
      {children}
    </button>
  );
}

export function Field({ label, hint, error, children }: { label: string; hint?: ReactNode; error?: string | null; children: ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className="text-[13px] font-semibold text-fg">{label}</span>
      {children}
      {error ? <span className="block text-xs text-danger">{error}</span> : hint && <span className="block text-xs text-fg-subtle">{hint}</span>}
    </label>
  );
}

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`h-9 w-full rounded-lg border border-line-strong bg-surface px-3 text-sm text-fg placeholder:text-fg-subtle focus:border-strong focus:outline-none ${props.className ?? ""}`}
    />
  );
}

export function Toggle({ checked, disabled, label, onChange }: { checked: boolean; disabled?: boolean; label: string; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors disabled:opacity-50 ${checked ? "bg-strong" : "bg-line-strong"}`}
    >
      <span className={`inline-block h-4 w-4 rounded-full bg-surface shadow transition-transform ${checked ? "translate-x-[18px]" : "translate-x-0.5"}`} />
    </button>
  );
}

export function Modal({ open, title, onClose, children, footer }: { open: boolean; title: ReactNode; onClose: () => void; children: ReactNode; footer?: ReactNode }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) d.showModal();
    if (!open && d.open) d.close();
  }, [open]);
  return (
    <dialog
      ref={ref}
      onClose={onClose}
      onCancel={onClose}
      className="m-auto w-[min(92vw,30rem)] rounded-2xl border border-line bg-surface p-0 text-fg shadow-raised backdrop:bg-black/40 backdrop:backdrop-blur-[2px]"
    >
      {open && (
        <div>
          <div className="flex items-center justify-between border-b border-line px-5 py-4">
            <h2 className="text-[15px] font-semibold text-heading">{title}</h2>
            <button type="button" onClick={onClose} className="rounded-md p-1 text-fg-subtle hover:bg-surface-3 hover:text-fg" aria-label="Close">
              <Icon name="x" />
            </button>
          </div>
          <div className="px-5 py-4">{children}</div>
          {footer && <div className="flex justify-end gap-2 border-t border-line px-5 py-3">{footer}</div>}
        </div>
      )}
    </dialog>
  );
}

export function EmptyState({ icon, title, hint, action }: { icon: IconName; title: string; hint?: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center px-6 py-14 text-center">
      <div className="mb-3 rounded-xl border border-line bg-surface-2 p-3 text-fg-muted">
        <Icon name={icon} className="h-6 w-6" />
      </div>
      <p className="text-[15px] font-semibold text-heading">{title}</p>
      {hint && <p className="mt-1 max-w-sm text-sm text-fg-muted">{hint}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

export function Spinner({ className = "h-4 w-4" }: { className?: string }) {
  return <span className={`inline-block animate-spin rounded-full border-2 border-line-strong border-t-strong ${className}`} aria-hidden="true" />;
}

export function ErrorBox({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { t } = useI18n();
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-danger/30 bg-danger-soft px-4 py-3 text-sm text-danger">
      <span className="flex items-center gap-2">
        <Icon name="alert" />
        {t("error")}: {message}
      </span>
      {onRetry && (
        <Button size="sm" onClick={onRetry}>
          {t("retry")}
        </Button>
      )}
    </div>
  );
}

export function formatDuration(seconds: number) {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function formatPhone(digits?: string) {
  if (!digits) return "";
  if (digits.startsWith("55") && digits.length >= 12) {
    const ddd = digits.slice(2, 4);
    const rest = digits.slice(4);
    return `+55 ${ddd} ${rest.slice(0, rest.length - 4)}-${rest.slice(-4)}`;
  }
  return `+${digits}`;
}

/** Host de uma URL para mostrar em lista; URL estranha aparece inteira. */
export function hostOf(url: string) {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}
