# Tutorial 2: human takeover

A bot that keeps talking while a person is helping the customer is worse than no bot. This
lesson makes your bot step aside the moment someone from the team answers from the phone,
and come back on its own later.

## How Arcanum tells you

Every message the **connected number itself** sends is forwarded with `from_me: true`,
including messages someone types on the phone. In that case `from` is the **customer** (the
conversation), not you:

```json
{
  "from": "5511999999999@s.whatsapp.net",
  "from_me": true,
  "type": "text",
  "text": "Hi Maria, I'll check your order now."
}
```

So the same key identifies the conversation in both directions.

## The rule

1. Message with `from_me: true` → a human is here → pause the bot **in this conversation**.
2. Message from the customer while paused → do nothing; the human will answer.
3. After N minutes without the human, the bot answers again.

## In code

=== "Python"

    ```python
    import time

    PAUSE = 30 * 60
    paused_until: dict[str, float] = {}   # use Redis/your DB in production

    def handle(event: dict) -> str | None:
        chat = event["from"]
        if event.get("from_me"):
            paused_until[chat] = time.time() + PAUSE
            return None
        if time.time() < paused_until.get(chat, 0):
            return None
        return answer_for(event.get("text", ""))
    ```

=== "Node.js"

    ```js
    const PAUSE_MS = 30 * 60 * 1000;
    const pausedUntil = new Map(); // use Redis/your DB in production

    function handle(event) {
      const chat = event.from;
      if (event.from_me) {
        pausedUntil.set(chat, Date.now() + PAUSE_MS);
        return null;
      }
      if (Date.now() < (pausedUntil.get(chat) ?? 0)) return null;
      return answerFor(event.text ?? "");
    }
    ```

=== "Redis (production)"

    ```python
    # one key per conversation, expiring by itself: no cleanup job needed
    if event.get("from_me"):
        redis.set(f"human:{chat}", 1, ex=30 * 60)
        return
    if redis.exists(f"human:{chat}"):
        return
    ```

The [FAQ bot](first-bot.md) already does this: try it. While the bot is running, answer a
customer from the bot phone, then send another message from the customer phone: the bot
stays quiet for 30 minutes (`HUMAN_PAUSE_MINUTES`).

## Good practices

- **Let the customer ask for a person.** A keyword like "human" or "agent" should notify your
  team and pause the bot right away (`faq.json` already answers it).
- **Tell the team.** When a human is needed, send them a notification (email, Slack, a
  message to an internal group) with the conversation.
- **Only the phone counts.** `from_me` comes from messages typed on another device of the
  account (the phone, WhatsApp Web, Desktop). Messages your bot sends through the API are
  not echoed back, so the bot never pauses itself.

**Next:** [Tutorial 3: notify customers from your system](notifications.md).
