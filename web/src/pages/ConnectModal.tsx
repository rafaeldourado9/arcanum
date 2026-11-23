import { type FormEvent, useEffect, useState } from "react";
import { api, type QRState } from "../api";
import { useI18n } from "../i18n";
import { Icon } from "../icons";
import { Button, Field, Input, Modal, Spinner } from "../ui";

/** Conecta uma sessao: QR code que se atualiza sozinho, ou codigo de pareamento. */
export default function ConnectModal({ name, onClose }: { name: string | null; onClose: () => void }) {
  const { t } = useI18n();
  const [tab, setTab] = useState<"qr" | "pair">("qr");
  const [state, setState] = useState<QRState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!name) return;
    setState(null);
    setError(null);
    setCode(null);
    setTab("qr");
    let stop = false;
    api
      .connect(name)
      .then((s) => !stop && setState(s))
      .catch((e) => !stop && setError(e.message));
    // O QR muda a cada ~20 s e a conexao fecha quando o celular le: consulta a cada 2,5 s.
    const id = setInterval(() => {
      api
        .state(name)
        .then((s) => !stop && setState(s))
        .catch(() => {});
    }, 2500);
    return () => {
      stop = true;
      clearInterval(id);
    };
  }, [name]);

  const pair = async (e: FormEvent) => {
    e.preventDefault();
    if (!name) return;
    setBusy(true);
    setError(null);
    try {
      setCode((await api.pair(name, phone.replace(/\D/g, ""))).code);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const connected = state?.status === "connected";

  return (
    <Modal open={!!name} title={t("qrTitle", { name: name ?? "" })} onClose={onClose} footer={<Button onClick={onClose}>{t("close")}</Button>}>
      {connected ? (
        <div className="flex flex-col items-center py-8 text-center">
          <div className="mb-3 rounded-full bg-success-soft p-3 text-success">
            <Icon name="check" className="h-6 w-6" />
          </div>
          <p className="text-sm font-semibold text-heading">{t("qrConnected")}</p>
        </div>
      ) : (
        <>
          <div className="mb-4 inline-flex rounded-lg border border-line bg-surface-2 p-0.5 text-[13px] font-semibold">
            {(["qr", "pair"] as const).map((k) => (
              <button
                key={k}
                type="button"
                onClick={() => setTab(k)}
                className={`rounded-md px-3 py-1.5 ${tab === k ? "bg-surface text-heading shadow-card" : "text-fg-muted hover:text-fg"}`}
              >
                {k === "qr" ? t("qrTab") : t("pairTab")}
              </button>
            ))}
          </div>

          {tab === "qr" ? (
            <div className="flex flex-col items-center">
              <div className="flex h-64 w-64 items-center justify-center rounded-xl border border-line bg-white p-3">
                {state?.qr ? <img src={state.qr} alt="QR code" className="h-full w-full" /> : <Spinner className="h-6 w-6" />}
              </div>
              <p className="mt-4 text-center text-[13px] text-fg-muted">{state?.qr ? t("qrScan") : t("qrWaiting")}</p>
            </div>
          ) : (
            <form onSubmit={pair} className="space-y-4">
              <p className="text-[13px] text-fg-muted">{t("pairHint")}</p>
              <Field label={t("phone")}>
                <Input inputMode="tel" placeholder="5511999999999" value={phone} onChange={(e) => setPhone(e.target.value)} />
              </Field>
              <Button type="submit" variant="primary" disabled={busy || phone.replace(/\D/g, "").length < 10}>
                {t("getCode")}
              </Button>
              {code && (
                <p className="rounded-xl border border-line bg-surface-2 py-4 text-center font-mono text-2xl font-bold tracking-[0.3em] text-heading">{code}</p>
              )}
            </form>
          )}
          {error && <p className="mt-3 text-sm text-danger">{error}</p>}
        </>
      )}
    </Modal>
  );
}
