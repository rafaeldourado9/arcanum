# Arcanum API

WhatsApp Gateway open-source em Go. Sucessor do Evolution API.

## Visão

Gateway WhatsApp standalone, provider-agnostic, com anti-ban nativo, suporte completo a mídia, multi-instância, e API REST compatível com qualquer backend.

## Stack

- Go + whatsmeow (engine de protocolo)
- Chi (HTTP router)
- SQLite (sessões)
- Docker-first

## Roadmap

### v1.0 — Core (atual)
- [x] Conexão via QR code e phone pairing
- [x] Envio/recebimento de texto
- [x] Envio/recebimento de mídia (imagem, áudio, documento, vídeo)
- [x] Anti-ban (delay, typing, recording, read receipts, rate limit)
- [x] Webhook forwarding com HMAC
- [x] Swagger UI
- [ ] Multi-instância
- [ ] Webhook/WebSocket configurável via API
- [ ] Send: buttons, list, location, contact, poll, reaction, template
- [ ] Chat: archive, find messages/contacts, mark read, profile update
- [ ] Groups: create, info, participants, update
- [ ] Labels: get, handle
- [ ] Settings/Proxy via API
- [ ] Business: catalog, collections
- [ ] Calls: offer call

### v2.0 — Open Source
- [ ] README completo com badges
- [ ] Site institucional de documentação (Docusaurus ou VitePress)
- [ ] Dashboard web para gerenciar instâncias
- [ ] Changelog automático (conventional commits)
- [ ] Releases no GitHub com binários
- [ ] Docker Hub image publicada
- [ ] CI/CD (GitHub Actions)
- [ ] Testes automatizados
- [ ] Licença MIT ou Apache 2.0
- [ ] Contributing guide

### v3.0 — Enterprise
- [ ] Autenticação por API key
- [ ] Multi-tenant
- [ ] Métricas e observabilidade (Prometheus)
- [ ] Rate limiting por instância
- [ ] Transcrição de áudio embutida
- [ ] Integração com S3/MinIO para mídia
- [ ] Suporte a Chatwoot/Typebot

## Regras

- Zero dependência de qualquer projeto específico (Civix ou outro)
- Provider-agnostic: whatsmeow é o engine atual, mas a interface permite trocar
- Anti-ban é obrigatório em todo envio — nunca enviar sem delay/presence
- Toda mensagem recebida deve ser encaminhada via webhook
- Mídia recebida: sempre baixar, converter para base64, incluir no webhook
- Configuração 100% por env vars ou API
- Docker-first, single binary
