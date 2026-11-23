# Aula 3: avisar clientes pelo seu sistema

Nem todo uso do WhatsApp é um chatbot. A maioria dos sistemas só precisa avisar coisas:
"seu pedido saiu", "segue seu boleto", "até amanhã às 15h". Nesta aula você envia essas
mensagens pelo seu próprio código.

## 1. Um texto com o status do pedido

=== "Python"

    ```python
    import os, httpx

    arcanum = httpx.Client(base_url=os.environ["ARCANUM_URL"],
                           headers={"X-Api-Key": os.environ["ARCANUM_KEY"]}, timeout=30)

    def pedido_enviado(telefone: str, nome: str, rastreio: str) -> None:
        texto = f"Oi {nome}! Seu pedido saiu para entrega. Código de rastreio: {rastreio}"
        r = arcanum.post("/api/message/sendText/vendas", json={"number": telefone, "text": texto})
        r.raise_for_status()   # 202 = na fila
    ```

=== "Node.js"

    ```js
    async function pedidoEnviado(telefone, nome, rastreio) {
      const res = await fetch(`${process.env.ARCANUM_URL}/api/message/sendText/vendas`, {
        method: "POST",
        headers: { "X-Api-Key": process.env.ARCANUM_KEY, "Content-Type": "application/json" },
        body: JSON.stringify({ number: telefone, text: `Oi ${nome}! Seu pedido saiu para entrega. Código de rastreio: ${rastreio}` }),
      });
      if (!res.ok) throw new Error(`Arcanum: HTTP ${res.status}`);
    }
    ```

=== "Go"

    ```go
    func pedidoEnviado(telefone, nome, rastreio string) error {
        body, _ := json.Marshal(map[string]string{
            "number": telefone,
            "text":   fmt.Sprintf("Oi %s! Seu pedido saiu para entrega. Código de rastreio: %s", nome, rastreio),
        })
        req, _ := http.NewRequest("POST", os.Getenv("ARCANUM_URL")+"/api/message/sendText/vendas", bytes.NewReader(body))
        req.Header.Set("X-Api-Key", os.Getenv("ARCANUM_KEY"))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()
        if resp.StatusCode >= 300 {
            return fmt.Errorf("arcanum: HTTP %d", resp.StatusCode)
        }
        return nil
    }
    ```

`202` quer dizer que a mensagem está **na fila**: o Arcanum envia alguns segundos depois, no
ritmo do anti-ban. Você não precisa esperar.

## 2. O boleto em PDF

```python
import base64, pathlib

def enviar_boleto(telefone: str, pdf: pathlib.Path) -> None:
    arcanum.post("/api/message/sendMedia/vendas", json={
        "number": telefone,
        "mediatype": "document",
        "mimetype": "application/pdf",
        "fileName": pdf.name,
        "caption": "Seu boleto",
        "media": base64.b64encode(pdf.read_bytes()).decode(),
    }).raise_for_status()
```

## 3. Saiba quando algo falhou

Um texto da fila que ainda falhou depois de 3 tentativas volta para o seu webhook:

```json
{"event": "send_failed", "to": "5511999999999", "text": "Oi Maria! Seu pedido saiu..."}
```

Registre e mostre no seu sistema ("WhatsApp não entregue"), ou mande por e-mail.

## 4. Confira os números antes de uma campanha

```bash
curl -X POST http://localhost:3100/api/chat/checkWhatsAppNumbers/vendas \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"numbers": ["5511999999999", "5511888888888"]}'
```

Pule os que não têm WhatsApp em vez de deixar o envio falhar.

## Regras que mantêm seu número vivo

- Fale só com quem passou o número para esse fim.
- Envie em horário razoável e com volume constante; `429` quer dizer que você bateu o limite
  por minuto, então diminua o ritmo em vez de repetir em laço.
- Personalize: nome, número do pedido, data. Disparo idêntico parece spam.
- Sempre ofereça um jeito de parar de receber mensagens.

**Próxima:** [Aula 4: trocar de servidor](move-server.md).
