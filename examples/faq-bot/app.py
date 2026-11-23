"""
FAQ bot for Arcanum: a complete, small WhatsApp bot you can copy.

What it shows:
  - verifying the X-Hub-Signature-256 signature of every webhook delivery;
  - answering common questions from a keyword table (faq.json);
  - pausing itself in a conversation when a human answers from the phone (from_me);
  - replying through the Arcanum API with the API key;
  - answering the webhook right away and doing the work in the background.

Run it with the docker compose file next to it, or locally:
    pip install -r requirements.txt
    ARCANUM_URL=http://localhost:3100 ARCANUM_KEY=... ARCANUM_WEBHOOK_SECRET=... \
        uvicorn app:app --port 8000
"""

from __future__ import annotations

import hashlib
import hmac
import json
import logging
import os
import time
import unicodedata
from pathlib import Path

import httpx
from fastapi import BackgroundTasks, FastAPI, Header, HTTPException, Request

ARCANUM_URL = os.environ.get("ARCANUM_URL", "http://localhost:3100").rstrip("/")
ARCANUM_KEY = os.environ["ARCANUM_KEY"]
WEBHOOK_SECRET = os.environ["ARCANUM_WEBHOOK_SECRET"].encode()
HUMAN_PAUSE_SECONDS = int(os.environ.get("HUMAN_PAUSE_MINUTES", "30")) * 60

FAQ = json.loads((Path(__file__).parent / "faq.json").read_text(encoding="utf-8"))

log = logging.getLogger("faq-bot")
logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")

app = FastAPI(title="Arcanum FAQ bot")
arcanum = httpx.Client(base_url=ARCANUM_URL, headers={"X-Api-Key": ARCANUM_KEY}, timeout=30)

# conversation -> unix time until which the bot stays quiet (a human is answering).
# In memory to keep the example short; use Redis or your database in production.
paused_until: dict[str, float] = {}


def normalize(text: str) -> str:
    """Lowercase without accents, so 'Preço' matches 'preco'."""
    text = unicodedata.normalize("NFKD", text.lower())
    return "".join(c for c in text if not unicodedata.combining(c))


def answer_for(text: str) -> str:
    words = normalize(text)
    for entry in FAQ["answers"]:
        if any(normalize(k) in words for k in entry["keywords"]):
            return entry["reply"]
    return FAQ["fallback"]


def verify_signature(body: bytes, signature: str) -> bool:
    expected = "sha256=" + hmac.new(WEBHOOK_SECRET, body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)


def reply(instance: str, to: str, text: str) -> None:
    """Runs after the webhook was answered: Arcanum queues it and applies anti-ban."""
    try:
        r = arcanum.post(f"/api/message/sendText/{instance}", json={"number": to, "text": text})
        if r.status_code >= 400:
            log.warning("send failed: HTTP %s %s", r.status_code, r.text[:200])
    except httpx.HTTPError as exc:
        log.warning("send failed: %s", exc)


@app.post("/webhooks/whatsapp/{instance}")
async def webhook(
    instance: str,
    request: Request,
    background: BackgroundTasks,
    x_hub_signature_256: str = Header(default=""),
):
    body = await request.body()
    if not verify_signature(body, x_hub_signature_256):
        raise HTTPException(status_code=401, detail="bad signature")

    event = json.loads(body)
    kind = event.get("event")

    if kind == "test":  # "Send test" button in the dashboard
        log.info("test delivery from %s", instance)
        return {"ok": True}
    if kind in ("typing", "send_failed"):
        return {"ok": True}

    chat = event.get("from", "")
    if event.get("from_me"):
        # A person answered this customer from the phone: stay out of the way.
        paused_until[chat] = time.time() + HUMAN_PAUSE_SECONDS
        log.info("human took over a conversation on %s; bot paused", instance)
        return {"ok": True}

    if time.time() < paused_until.get(chat, 0):
        return {"ok": True}
    if event.get("type") != "text" or not event.get("text"):
        background.add_task(reply, instance, chat, FAQ["only_text"])
        return {"ok": True}

    background.add_task(reply, instance, chat, answer_for(event["text"]))
    return {"ok": True}


@app.get("/health")
def health():
    return {"ok": True}
