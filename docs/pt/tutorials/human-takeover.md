# Aula 2: atendimento humano

Um bot que continua falando enquanto uma pessoa atende o cliente é pior que nenhum bot.
Nesta aula o bot sai da frente assim que alguém da equipe responde pelo celular, e volta
sozinho depois.

## Como o Arcanum avisa

Toda mensagem enviada **pelo próprio número conectado** é encaminhada com `from_me: true`,
inclusive o que alguém digita no celular. Nesse caso o `from` é o **cliente** (a conversa), e
não você:

```json
{
  "from": "5511999999999@s.whatsapp.net",
  "from_me": true,
  "type": "text",
  "text": "Oi Maria, vou ver seu pedido agora."
}
```

Assim a mesma chave identifica a conversa nos dois sentidos.

## A regra

1. Mensagem com `from_me: true` → tem um humano → pausa o bot **nesta conversa**.
2. Mensagem do cliente durante a pausa → não faz nada; o humano responde.
3. Depois de N minutos sem o humano, o bot volta a responder.

## No código

=== "Python"

    ```python
    import time

    PAUSA = 30 * 60
    pausado_ate: dict[str, float] = {}   # em produção, use Redis ou o seu banco

    def tratar(evento: dict) -> str | None:
        conversa = evento["from"]
        if evento.get("from_me"):
            pausado_ate[conversa] = time.time() + PAUSA
            return None
        if time.time() < pausado_ate.get(conversa, 0):
            return None
        return resposta_para(evento.get("text", ""))
    ```

=== "Node.js"

    ```js
    const PAUSA_MS = 30 * 60 * 1000;
    const pausadoAte = new Map(); // em produção, use Redis ou o seu banco

    function tratar(evento) {
      const conversa = evento.from;
      if (evento.from_me) {
        pausadoAte.set(conversa, Date.now() + PAUSA_MS);
        return null;
      }
      if (Date.now() < (pausadoAte.get(conversa) ?? 0)) return null;
      return respostaPara(evento.text ?? "");
    }
    ```

=== "Redis (produção)"

    ```python
    # uma chave por conversa, que expira sozinha: nenhuma rotina de limpeza
    if evento.get("from_me"):
        redis.set(f"humano:{conversa}", 1, ex=30 * 60)
        return
    if redis.exists(f"humano:{conversa}"):
        return
    ```

O [bot de FAQ](first-bot.md) já faz isso: experimente. Com o bot rodando, responda um cliente
pelo celular do bot e depois mande outra mensagem pelo celular do cliente: o bot fica quieto
por 30 minutos (`HUMAN_PAUSE_MINUTES`).

## Boas práticas

- **Deixe o cliente pedir uma pessoa.** Uma palavra como "atendente" ou "humano" deve avisar a
  equipe e pausar o bot na hora (o `faq.json` já responde isso).
- **Avise a equipe.** Quando precisar de um humano, mande uma notificação (e-mail, Slack, uma
  mensagem num grupo interno) com a conversa.
- **Só o celular conta.** O `from_me` vem de mensagens digitadas em outro aparelho da conta
  (celular, WhatsApp Web, Desktop). O que o bot envia pela API não volta como eco, então o bot
  nunca pausa a si mesmo.

**Próxima:** [Aula 3: avisar clientes pelo seu sistema](notifications.md).
