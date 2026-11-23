/**
 * Ingles e portugues. O idioma vem do navegador e pode ser trocado na tela (fica salvo).
 * Texto novo entra nos dois dicionarios: o tipo garante que nenhuma chave falte em pt.
 */
import { createContext, type ReactNode, useContext, useMemo, useState } from "react";

const en = {
  appTagline: "WhatsApp gateway",
  navOverview: "Overview",
  navSessions: "Sessions",
  navWebhooks: "Webhooks",
  navLogs: "Logs",
  navSettings: "Settings",
  navDocs: "Documentation",
  signOut: "Sign out",
  themeLight: "Light",
  themeDark: "Dark",
  language: "Language",

  loginTitle: "Sign in to Arcanum",
  loginHint: "Use the dashboard password (GATEWAY_DASHBOARD_PASSWORD) or, if none is set, the API key.",
  password: "Password",
  signIn: "Sign in",
  signingIn: "Signing in...",

  overviewTitle: "Overview",
  overviewDesc: "Sessions, messages and webhook deliveries since the gateway started.",
  connected: "Connected",
  waitingScan: "Waiting for QR",
  connecting: "Connecting",
  disconnected: "Disconnected",
  received: "Messages received",
  sent: "Messages sent",
  sendErrors: "Send failures",
  delivered: "Webhooks delivered",
  failed: "Webhooks failed",
  uptime: "Uptime",
  version: "Version",
  recentSessions: "Sessions",
  viewAll: "View all",

  sessionsTitle: "Sessions",
  sessionsDesc: "Each session is one WhatsApp number. Sessions reconnect on their own after a restart.",
  newSession: "New session",
  noSessions: "No sessions yet",
  noSessionsHint: "Create a session and scan the QR code with WhatsApp to connect a number.",
  name: "Name",
  nameHint: "Letters, digits, - or _ (up to 64).",
  webhookOptional: "Webhook URL (optional)",
  create: "Create",
  cancel: "Cancel",
  phone: "Number",
  status: "Status",
  created: "Created",
  actions: "Actions",
  connect: "Connect",
  restart: "Restart",
  disconnect: "Disconnect",
  delete: "Delete",
  open: "Open",
  deleteConfirm: "Delete session {name}? The device is unlinked from the phone and its files are removed. This cannot be undone.",
  deleteTitle: "Delete session",

  qrTitle: "Connect {name}",
  qrScan: "Open WhatsApp > Linked devices > Link a device and scan the code. It refreshes every ~20 seconds.",
  qrWaiting: "Generating QR code...",
  qrConnected: "Connected. You can close this window.",
  qrTab: "QR code",
  pairTab: "Pairing code",
  pairHint: "No camera? Enter the number with country code and type the code in WhatsApp > Linked devices > Link with phone number.",
  getCode: "Get code",
  close: "Close",

  sessionTitle: "Session",
  back: "Sessions",
  webhook: "Webhook",
  webhookDesc: "Incoming messages are posted to this URL, signed with GATEWAY_META_APP_SECRET.",
  url: "URL",
  enabled: "Enabled",
  save: "Save",
  saved: "Saved",
  sendTest: "Send test",
  testOk: "Delivered: HTTP {status} in {ms} ms",
  testFail: "Failed: {error}",
  deliveries: "Recent deliveries",
  noDeliveries: "No deliveries yet.",
  event: "Event",
  result: "Result",
  duration: "Duration",
  time: "Time",
  settings: "Settings",
  groupsIgnore: "Ignore group messages",
  groupsIgnoreHint: "Group messages are not forwarded to the webhook.",
  planned: "Planned",
  plannedHint: "Stored now, takes effect in an upcoming release.",
  rejectCalls: "Reject calls",
  alwaysOnline: "Always online",
  readMessages: "Auto read receipts",

  webhooksTitle: "Webhooks",
  webhooksDesc: "Where each session delivers its messages. Export to back up, import to restore.",
  export: "Export",
  import: "Import",
  importDone: "{n} updated",
  importSkipped: "Skipped",
  notSet: "Not set",
  on: "On",
  off: "Off",

  logsTitle: "Logs",
  logsDesc: "Live gateway log. Message text, phone numbers and secrets never appear here.",
  live: "Live",
  paused: "Paused",
  pause: "Pause",
  resume: "Resume",
  clear: "Clear",
  filterAll: "All",
  noLogs: "No log lines yet.",

  settingsTitle: "Settings",
  settingsDesc: "Appearance and useful links. Gateway options are environment variables.",
  appearance: "Appearance",
  about: "About",
  apiDocs: "Interactive API (Swagger)",
  guide: "Guide",
  repo: "GitHub",
  securityTip: "Security tip",
  securityTipBody: "Keep port 3100 behind a reverse proxy with HTTPS and set a long GATEWAY_DASHBOARD_PASSWORD.",

  error: "Something went wrong",
  retry: "Try again",
  loading: "Loading...",
};

export type Dict = typeof en;

const pt: Dict = {
  appTagline: "Gateway de WhatsApp",
  navOverview: "Visão geral",
  navSessions: "Sessões",
  navWebhooks: "Webhooks",
  navLogs: "Logs",
  navSettings: "Configurações",
  navDocs: "Documentação",
  signOut: "Sair",
  themeLight: "Claro",
  themeDark: "Escuro",
  language: "Idioma",

  loginTitle: "Entrar no Arcanum",
  loginHint: "Use a senha do painel (GATEWAY_DASHBOARD_PASSWORD) ou, se ela não foi definida, a chave de API.",
  password: "Senha",
  signIn: "Entrar",
  signingIn: "Entrando...",

  overviewTitle: "Visão geral",
  overviewDesc: "Sessões, mensagens e entregas de webhook desde que o gateway subiu.",
  connected: "Conectadas",
  waitingScan: "Esperando QR",
  connecting: "Conectando",
  disconnected: "Desconectadas",
  received: "Mensagens recebidas",
  sent: "Mensagens enviadas",
  sendErrors: "Falhas de envio",
  delivered: "Webhooks entregues",
  failed: "Webhooks com falha",
  uptime: "No ar há",
  version: "Versão",
  recentSessions: "Sessões",
  viewAll: "Ver todas",

  sessionsTitle: "Sessões",
  sessionsDesc: "Cada sessão é um número de WhatsApp. As sessões reconectam sozinhas depois de um reinício.",
  newSession: "Nova sessão",
  noSessions: "Nenhuma sessão ainda",
  noSessionsHint: "Crie uma sessão e leia o QR code com o WhatsApp para conectar um número.",
  name: "Nome",
  nameHint: "Letras, números, - ou _ (até 64).",
  webhookOptional: "URL do webhook (opcional)",
  create: "Criar",
  cancel: "Cancelar",
  phone: "Número",
  status: "Status",
  created: "Criada em",
  actions: "Ações",
  connect: "Conectar",
  restart: "Reiniciar",
  disconnect: "Desconectar",
  delete: "Apagar",
  open: "Abrir",
  deleteConfirm: "Apagar a sessão {name}? O aparelho é desconectado do celular e os arquivos são removidos. Não dá para desfazer.",
  deleteTitle: "Apagar sessão",

  qrTitle: "Conectar {name}",
  qrScan: "Abra o WhatsApp > Aparelhos conectados > Conectar um aparelho e leia o código. Ele muda a cada ~20 segundos.",
  qrWaiting: "Gerando o QR code...",
  qrConnected: "Conectado. Pode fechar esta janela.",
  qrTab: "QR code",
  pairTab: "Código de pareamento",
  pairHint: "Sem câmera? Informe o número com código do país e digite o código em WhatsApp > Aparelhos conectados > Conectar com número de telefone.",
  getCode: "Gerar código",
  close: "Fechar",

  sessionTitle: "Sessão",
  back: "Sessões",
  webhook: "Webhook",
  webhookDesc: "As mensagens recebidas vão para esta URL, assinadas com GATEWAY_META_APP_SECRET.",
  url: "URL",
  enabled: "Ligado",
  save: "Salvar",
  saved: "Salvo",
  sendTest: "Enviar teste",
  testOk: "Entregue: HTTP {status} em {ms} ms",
  testFail: "Falhou: {error}",
  deliveries: "Entregas recentes",
  noDeliveries: "Nenhuma entrega ainda.",
  event: "Evento",
  result: "Resultado",
  duration: "Duração",
  time: "Horário",
  settings: "Configurações",
  groupsIgnore: "Ignorar mensagens de grupo",
  groupsIgnoreHint: "Mensagens de grupo não vão para o webhook.",
  planned: "Planejado",
  plannedHint: "Fica guardado agora e passa a valer numa próxima versão.",
  rejectCalls: "Recusar chamadas",
  alwaysOnline: "Sempre online",
  readMessages: "Confirmação de leitura automática",

  webhooksTitle: "Webhooks",
  webhooksDesc: "Para onde cada sessão entrega as mensagens. Exporte para guardar, importe para restaurar.",
  export: "Exportar",
  import: "Importar",
  importDone: "{n} atualizadas",
  importSkipped: "Puladas",
  notSet: "Não definida",
  on: "Ligado",
  off: "Desligado",

  logsTitle: "Logs",
  logsDesc: "Log do gateway ao vivo. Texto de mensagem, telefones e segredos nunca aparecem aqui.",
  live: "Ao vivo",
  paused: "Pausado",
  pause: "Pausar",
  resume: "Continuar",
  clear: "Limpar",
  filterAll: "Todos",
  noLogs: "Nenhuma linha de log ainda.",

  settingsTitle: "Configurações",
  settingsDesc: "Aparência e links úteis. As opções do gateway são variáveis de ambiente.",
  appearance: "Aparência",
  about: "Sobre",
  apiDocs: "API interativa (Swagger)",
  guide: "Guia",
  repo: "GitHub",
  securityTip: "Dica de segurança",
  securityTipBody: "Deixe a porta 3100 atrás de um proxy reverso com HTTPS e defina uma GATEWAY_DASHBOARD_PASSWORD longa.",

  error: "Algo deu errado",
  retry: "Tentar de novo",
  loading: "Carregando...",
};

export type Lang = "en" | "pt";
const DICTS: Record<Lang, Dict> = { en, pt };

function initialLang(): Lang {
  try {
    const saved = localStorage.getItem("arcanum.lang");
    if (saved === "en" || saved === "pt") return saved;
  } catch {
    /* sem localStorage */
  }
  return navigator.language.toLowerCase().startsWith("pt") ? "pt" : "en";
}

type Ctx = { lang: Lang; setLang: (l: Lang) => void; t: (key: keyof Dict, vars?: Record<string, string | number>) => string };
const I18nContext = createContext<Ctx | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(initialLang);
  const value = useMemo<Ctx>(() => {
    const dict = DICTS[lang];
    return {
      lang,
      setLang: (l) => {
        setLangState(l);
        document.documentElement.lang = l === "pt" ? "pt-BR" : "en";
        try {
          localStorage.setItem("arcanum.lang", l);
        } catch {
          /* ok */
        }
      },
      t: (key, vars) => {
        let s = dict[key];
        if (vars) for (const [k, v] of Object.entries(vars)) s = s.replaceAll(`{${k}}`, String(v));
        return s;
      },
    };
  }, [lang]);
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n outside I18nProvider");
  return ctx;
}
