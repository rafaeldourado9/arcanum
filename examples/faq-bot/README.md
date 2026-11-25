# FAQ bot (Python)

A complete WhatsApp bot on top of Arcanum, small enough to read in five minutes:

- verifies the `X-Hub-Signature-256` signature of every webhook delivery;
- answers common questions from [`faq.json`](faq.json) (accents and case do not matter);
- **pauses itself for 30 minutes in a conversation when a human answers from the phone**
  (`from_me`);
- answers the webhook immediately and replies in the background, through the Arcanum API.

Step-by-step walkthrough: [Tutorial: your first bot](https://arcanumgateway.com.br/en/tutorials/first-bot/)
([em português](https://arcanumgateway.com.br/tutorials/first-bot/)).

## Run

```bash
cp .env.example .env
# fill ARCANUM_KEY, ARCANUM_WEBHOOK_SECRET and DASHBOARD_PASSWORD (openssl rand -hex 32)
docker compose up -d --build
```

1. Open http://localhost:3100 and sign in with `DASHBOARD_PASSWORD`.
2. **Sessions > New session**, name it `demo`, scan the QR code with the phone that will be
   the bot.
3. From another phone, send "what are your opening hours?" to that number.

The webhook is already wired: `GATEWAY_WEBHOOK_FORWARD_URL` points every session to the bot.
Use **Send test** on the session page to check the connection.

## Change the answers

Edit `faq.json`: each entry has `keywords` and a `reply`. Restart the bot
(`docker compose restart bot`).

## Tests

```bash
pip install -r requirements.txt pytest
pytest
```

## Going further

- Keep `paused_until` in Redis or your database instead of memory.
- Replace `answer_for` with a call to your favourite AI model, keeping the same structure.
- Put Arcanum behind HTTPS before exposing it: see
  [Deployment](https://arcanumgateway.com.br/en/deployment/).
