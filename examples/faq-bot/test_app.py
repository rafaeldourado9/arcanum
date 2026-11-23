"""Tests for the FAQ bot: signature, answers and human takeover. Run: pytest"""

import hashlib
import hmac
import json
import os

os.environ.setdefault("ARCANUM_KEY", "test-key")
os.environ.setdefault("ARCANUM_WEBHOOK_SECRET", "test-secret")

from fastapi.testclient import TestClient  # noqa: E402

import app as bot  # noqa: E402

client = TestClient(bot.app)
sent: list[tuple[str, str, str]] = []
bot.reply = lambda instance, to, text: sent.append((instance, to, text))  # no real API call


def post(event: dict, secret: bytes = b"test-secret"):
    body = json.dumps(event).encode()
    sig = "sha256=" + hmac.new(secret, body, hashlib.sha256).hexdigest()
    return client.post("/webhooks/whatsapp/demo", content=body,
                       headers={"X-Hub-Signature-256": sig, "Content-Type": "application/json"})


def setup_function():
    sent.clear()
    bot.paused_until.clear()


def test_rejects_bad_signature():
    assert post({"type": "text", "text": "hi"}, secret=b"wrong").status_code == 401


def test_answers_from_faq_without_accents():
    post({"from": "5511@s.whatsapp.net", "from_me": False, "type": "text", "text": "Qual o PREÇO?"})
    assert sent and "plans start" in sent[0][2]


def test_fallback_when_unknown():
    post({"from": "5511@s.whatsapp.net", "from_me": False, "type": "text", "text": "banana"})
    assert sent[0][2] == bot.FAQ["fallback"]


def test_pauses_when_human_answers():
    chat = "5511@s.whatsapp.net"
    post({"from": chat, "from_me": True, "type": "text", "text": "I'll handle it"})
    post({"from": chat, "from_me": False, "type": "text", "text": "price?"})
    assert sent == []


def test_ignores_typing_and_test_events():
    post({"event": "typing", "from": "5511@s.whatsapp.net", "state": "composing"})
    post({"event": "test", "instance": "demo"})
    assert sent == []
