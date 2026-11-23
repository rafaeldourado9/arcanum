# Tutorial 3: notify customers from your system

Not every use of WhatsApp is a chatbot. Most systems just need to tell customers things:
"your order shipped", "here is your invoice", "see you tomorrow at 3pm". This lesson sends
those messages from your own code.

## 1. A text with the order status

=== "Python"

    ```python
    import os, httpx

    arcanum = httpx.Client(base_url=os.environ["ARCANUM_URL"],
                           headers={"X-Api-Key": os.environ["ARCANUM_KEY"]}, timeout=30)

    def order_shipped(phone: str, name: str, tracking: str) -> None:
        text = f"Hi {name}! Your order shipped. Tracking code: {tracking}"
        r = arcanum.post("/api/message/sendText/sales", json={"number": phone, "text": text})
        r.raise_for_status()   # 202 = queued
    ```

=== "Node.js"

    ```js
    async function orderShipped(phone, name, tracking) {
      const res = await fetch(`${process.env.ARCANUM_URL}/api/message/sendText/sales`, {
        method: "POST",
        headers: { "X-Api-Key": process.env.ARCANUM_KEY, "Content-Type": "application/json" },
        body: JSON.stringify({ number: phone, text: `Hi ${name}! Your order shipped. Tracking code: ${tracking}` }),
      });
      if (!res.ok) throw new Error(`Arcanum: HTTP ${res.status}`);
    }
    ```

=== "Go"

    ```go
    func orderShipped(phone, name, tracking string) error {
        body, _ := json.Marshal(map[string]string{
            "number": phone,
            "text":   fmt.Sprintf("Hi %s! Your order shipped. Tracking code: %s", name, tracking),
        })
        req, _ := http.NewRequest("POST", os.Getenv("ARCANUM_URL")+"/api/message/sendText/sales", bytes.NewReader(body))
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

`202` means the message is **queued**: Arcanum sends it a few seconds later with the
anti-ban pacing. You do not need to wait.

## 2. The invoice as a PDF

```python
import base64, pathlib

def send_invoice(phone: str, pdf: pathlib.Path) -> None:
    arcanum.post("/api/message/sendMedia/sales", json={
        "number": phone,
        "mediatype": "document",
        "mimetype": "application/pdf",
        "fileName": pdf.name,
        "caption": "Your invoice",
        "media": base64.b64encode(pdf.read_bytes()).decode(),
    }).raise_for_status()
```

## 3. Know when something failed

A queued text that still fails after 3 attempts comes back to your webhook:

```json
{"event": "send_failed", "to": "5511999999999", "text": "Hi Maria! Your order shipped..."}
```

Record it and show it in your system ("WhatsApp not delivered"), or fall back to email.

## 4. Check numbers before a campaign

```bash
curl -X POST http://localhost:3100/api/chat/checkWhatsAppNumbers/sales \
  -H "X-Api-Key: $ARCANUM_KEY" -H "Content-Type: application/json" \
  -d '{"numbers": ["5511999999999", "5511888888888"]}'
```

Skip the ones without WhatsApp instead of letting the sends fail.

## Rules that keep your number alive

- Only message people who gave you their number for this purpose.
- Send at reasonable hours and keep the volume steady; `429` means you hit the per-minute
  limit, so slow down instead of retrying in a loop.
- Personalize: name, order number, date. Identical blasts look like spam.
- Always offer a way to stop receiving messages.

**Next:** [Tutorial 4: move to a new server](move-server.md).
