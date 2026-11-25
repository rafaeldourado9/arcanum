/**
 * Cliente da API do gateway para o painel. A autenticacao e o cookie de sessao (HttpOnly,
 * o JavaScript nem ve) mais o header anti-CSRF em todo pedido. 401 manda para o login.
 */

export type Status = "disconnected" | "connecting" | "qr" | "connected";

export interface WebhookConfig {
  url: string;
  events: string[];
  enabled: boolean;
}

export interface Settings {
  rejectCalls: boolean;
  groupsIgnore: boolean;
  alwaysOnline: boolean;
  readMessages: boolean;
  readStatus: boolean;
  syncFullHistory: boolean;
}

export interface Session {
  instanceName: string;
  status: Status;
  phone?: string;
  createdAt: string;
  settings: Settings;
  webhook: WebhookConfig;
}

export interface Delivery {
  time: string;
  event: string;
  status: number;
  ok: boolean;
  error?: string;
  durationMs: number;
}

export interface Stats {
  version: string;
  uptimeSeconds: number;
  sessions: Partial<Record<Status, number>>;
  counters: {
    messagesReceived: number;
    messagesSent: number;
    messagesSendError: number;
    webhookDelivered: number;
    webhookFailed: number;
  };
}

export interface LogEntry {
  seq: number;
  time: string;
  level: "DEBUG" | "INFO" | "WARN" | "ERROR";
  msg: string;
  attrs?: Record<string, unknown>;
}

export interface QRState {
  instanceName: string;
  status: Status;
  qr: string;
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

const HEADERS = { "X-Requested-With": "arcanum" };

let onUnauthorized: () => void = () => {};
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    credentials: "same-origin",
    headers: body === undefined ? HEADERS : { ...HEADERS, "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (res.status === 401 && !path.startsWith("/api/auth/")) onUnauthorized();
  const text = await res.text();
  let data: unknown = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = null;
  }
  if (!res.ok) {
    const msg = (data as { error?: string } | null)?.error ?? `HTTP ${res.status}`;
    throw new ApiError(res.status, msg);
  }
  return data as T;
}

const enc = encodeURIComponent;

export const api = {
  me: () => request<{ authenticated: boolean; version: string }>("GET", "/api/auth/me"),
  login: (password: string) => request<{ authenticated: boolean }>("POST", "/api/auth/login", { password }),
  logout: () => request<unknown>("POST", "/api/auth/logout"),

  stats: () => request<Stats>("GET", "/api/stats"),
  sessions: () => request<Session[]>("GET", "/api/instance/fetchInstances"),
  create: (instanceName: string, webhook?: string) =>
    request<Session>("POST", "/api/instance/create", webhook ? { instanceName, webhook } : { instanceName }),
  connect: (name: string) => request<QRState>("GET", `/api/instance/connect/${enc(name)}`),
  state: (name: string) => request<QRState & { number?: string }>("GET", `/api/instance/connectionState/${enc(name)}`),
  pair: (name: string, phone: string) => request<{ code: string }>("POST", `/api/instance/pair/${enc(name)}`, { phone }),
  restart: (name: string) => request<unknown>("POST", `/api/instance/restart/${enc(name)}`),
  logoutSession: (name: string) => request<unknown>("DELETE", `/api/instance/logout/${enc(name)}`),
  remove: (name: string) => request<unknown>("DELETE", `/api/instance/delete/${enc(name)}`),

  webhook: (name: string) => request<WebhookConfig>("GET", `/api/events/webhook/${enc(name)}`),
  setWebhook: (name: string, cfg: Partial<WebhookConfig>) =>
    request<WebhookConfig>("POST", `/api/events/webhook/${enc(name)}`, cfg),
  testWebhook: (name: string) => request<Delivery>("POST", `/api/events/webhook/${enc(name)}/test`),
  deliveries: (name: string) =>
    request<{ deliveries: Delivery[] }>("GET", `/api/events/webhook/${enc(name)}/deliveries`),
  exportWebhooks: () => request<{ version: number; webhooks: Record<string, WebhookConfig> }>("GET", "/api/events/webhooks/export"),
  importWebhooks: (webhooks: Record<string, WebhookConfig>) =>
    request<{ updated: string[]; skipped: Record<string, string> }>("POST", "/api/events/webhooks/import", { webhooks }),

  setSettings: (name: string, s: Partial<Settings>) => request<Settings>("POST", `/api/settings/${enc(name)}`, s),

  logs: (limit = 300) => request<{ entries: LogEntry[] }>("GET", `/api/logs?limit=${limit}`),
};

/**
 * Logs ao vivo. EventSource nao manda header customizado (e o header anti-CSRF e
 * obrigatorio), entao le o stream SSE com fetch.
 */
export function streamLogs(onEntry: (e: LogEntry) => void, signal: AbortSignal): Promise<void> {
  return fetch("/api/logs/stream", { credentials: "same-origin", headers: HEADERS, signal }).then(async (res) => {
    if (res.status === 401) onUnauthorized();
    if (!res.ok || !res.body) throw new ApiError(res.status, `HTTP ${res.status}`);
    const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
    let buffer = "";
    for (;;) {
      const { value, done } = await reader.read();
      if (done) return;
      buffer += value;
      let idx: number;
      while ((idx = buffer.indexOf("\n\n")) >= 0) {
        const chunk = buffer.slice(0, idx);
        buffer = buffer.slice(idx + 2);
        const line = chunk.split("\n").find((l) => l.startsWith("data: "));
        if (line) {
          try {
            onEntry(JSON.parse(line.slice(6)) as LogEntry);
          } catch {
            /* linha quebrada: ignora */
          }
        }
      }
    }
  });
}

/** Guia no idioma do painel: portugues na raiz do site, ingles em /en/. */
export function docsUrl(lang: "en" | "pt") {
  return lang === "pt" ? "https://arcanumgateway.com.br/" : "https://arcanumgateway.com.br/en/";
}
