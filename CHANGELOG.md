# Changelog

Todas as mudanças notaveis do projeto serao documentadas aqui.

O formato segue [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/),
e o projeto adere ao [Versionamento Semantico](https://semver.org/lang/pt-BR/).

## [Não lançado]

### Corrigido

- **Três configurações da sessão apareciam no painel e na API mas não faziam nada.** Agora
  funcionam de verdade, por sessão e sem reiniciar:
  - **Recusar chamadas** (`rejectCalls`): a chamada recebida é recusada.
  - **Sempre online** (`alwaysOnline`): a sessão reenvia a presença "disponível" a cada minuto
    enquanto conectada.
  - **Confirmação de leitura automática** (`readMessages`): mensagens de contatos são marcadas
    como lidas depois de 1 a 3 s (não marca as que o próprio número enviou).
- Atenção: `readMessages` já vinha ligado por padrão (e sem efeito). Sessões existentes passam
  a confirmar leitura; desligue em `POST /api/settings/{instance}` se não quiser.

## [1.1.1] - 2026-11-25

Correções de instalação, achadas testando todas as formas de instalar do zero.

### Corrigido

- **Docker Compose reiniciava sem parar** com o `.env` copiado do exemplo: o `.env.example`
  trazia `GATEWAY_DB_PATH=./data`, que sobrescrevia o `/data` da imagem e falhava no sistema
  de arquivos somente leitura. Os dois compose agora fixam `/data` (vale até com `.env` antigo).
- `install.sh` com `ARCANUM_BIN_DIR` numa pasta que ainda não existe dava um erro enganoso
  ("run as root"); agora cria a pasta.
- `.env.example` vinha com o webhook preenchido (`your-app.example.com`) e o gateway tentava
  entregar lá; agora vem comentado.

### Adicionado

- `deploy/.env.example` próprio para Docker Compose, só com o que vale em container, e
  `GATEWAY_DASHBOARD_PASSWORD` nos dois exemplos.
- Documentação e site no domínio próprio **arcanumgateway.com.br**; instaladores em
  `https://arcanumgateway.com.br/install.sh` e `/install.ps1`.
- Docs: o `npx` precisa de Git; comando para instalar sem sudo; plataformas testadas.

## [1.1.0] - 2026-11-23

Foco: segurança e confiabilidade. **Mudança incompatível:** toda rota agora exige chave de API.

### Segurança

- **Chave de API obrigatória** em toda rota, menos `/health` e `/docs` (`X-Api-Key` ou
  `Authorization: Bearer`, comparação em tempo constante). Sem `GATEWAY_API_KEY`, uma chave
  temporária é gerada e mostrada no log. `GATEWAY_AUTH_DISABLED=true` só para desenvolvimento.
- Nome de instância validado (`A-Z a-z 0-9 - _`, até 64): antes, `../` gravava a sessão fora
  da pasta de dados.
- Webhook protegido contra SSRF: só `http`/`https`, metadados da nuvem e link-local bloqueados
  na hora da conexão, sem seguir redirecionamento, sem credencial na URL.
- Logs sem número, nome ou texto de mensagem (LGPD/GDPR).
- Limite de tamanho de requisição (`GATEWAY_MAX_BODY_MB`), timeouts HTTP e cabeçalhos de
  segurança (`nosniff`, `X-Frame-Options`, CSP, `no-store`).
- Imagem Docker sem root, pasta de dados `700`, build reprodutível e `HEALTHCHECK`.
- Removido `middleware.RealIP` (aceitava `X-Forwarded-For` falsificado).
- Dependências atualizadas sem vulnerabilidade conhecida (Go 1.26.11, chi v5.5.0, whatsmeow
  de 16/11/2026).
- CI com detector de corrida, `staticcheck`, `govulncheck`, `gosec` e Trivy; Dependabot;
  `SECURITY.md`.

### Corrigido

- QR code não aparecia: a whatsmeow desatualizada era recusada pelo WhatsApp e os erros do QR
  eram descartados sem log. Agora a biblioteca está atualizada e todo evento de QR com erro
  vai para o log.
- "Conectar" repetido abria vários clientes ao mesmo tempo, sem fechar os anteriores.
- `updateProfileName` trocava o recado em vez do nome.
- Trocar o webhook pela API mudava o formato do payload para `simple` sem aviso.
- `archiveChat` respondia sucesso sem arquivar; agora responde `501`.
- Especificação OpenAPI listava 18 rotas inexistentes; agora um teste garante que ela bate com
  o roteador.
- Erros de validação ao criar instância voltam `400` (antes `409`).

### Painel web (novo)

- Painel embutido no binário em `/` (React, Vite, Tailwind; tema claro e escuro; inglês e
  português): visão geral, sessões com QR code que se atualiza sozinho e código de pareamento,
  página da sessão com webhook, teste de entrega, histórico das últimas 100 entregas e
  configurações, exportar/importar webhooks e logs ao vivo.
- Login com `GATEWAY_DASHBOARD_PASSWORD` (sem ela, a chave de API): cookie HttpOnly +
  SameSite=Strict, cabeçalho anti-CSRF obrigatório, bloqueio após 10 erros por IP, CSP
  restrita, sessões guardadas por hash em memória. `GATEWAY_DASHBOARD=false` desliga.
- Rotas novas: `/api/stats`, `/api/logs`, `/api/logs/stream` (SSE), `/api/auth/*`,
  `/api/events/webhook/{instance}/test`, `/api/events/webhook/{instance}/deliveries`,
  `/api/events/webhooks/export` e `/import`. Logs do painel mascaram segredos.

### Instalação (nova)

- SQLite em Go puro (`modernc.org/sqlite`): sem CGO, binário estático, compila para qualquer
  sistema. Sessões já pareadas continuam abrindo (mesmo formato de arquivo).
- Binários para Linux, macOS e Windows (amd64 e arm64) a cada versão, com `checksums.txt` e
  atestado de proveniência (GoReleaser).
- Script de instalação (`install.sh`, confere SHA-256) com `--systemd`, que cria usuário,
  pasta, `.env` com chave e senha aleatórias e serviço do systemd isolado.
- Imagem distroless (34 MB, sem shell), usuário nonroot (UID 65532), funciona somente leitura;
  comandos `arcanum healthcheck` e `arcanum version`.
- `deploy/`: Compose local, Compose com HTTPS (Caddy), manifesto do Kubernetes, unit do systemd.
- `npx arcanum-gateway` (pacote npm sem dependências: baixa o binário cru da release, confere o
  SHA-256 e roda; cache em `~/.arcanum`) e `npm create arcanum` (projeto Compose com chaves
  geradas). Publicados no npm a cada tag, com proveniência.
- Treinamento com 4 aulas e projeto de exemplo `examples/faq-bot` (bot em Python com testes).
- **Atualizando da 1.0:** o volume de dados precisa passar para o UID 65532
  (`docker run --rm -v arcanum-data:/data alpine chown -R 65532:65532 /data`).

### Corrigido (painel)

- Configuração de webhook e de cada sessão se perdia ao reiniciar; agora fica em `<nome>.json`.
- Apagar a instância não apagava o `<nome>.db`, e ela voltava no reinício; agora desconecta o
  aparelho do celular e remove os arquivos.
- Leitura e escrita da configuração da instância sem trava (condição de corrida).
- Arquivos da sessão eram criados legíveis por outros usuários; agora `600`.

### Adicionado

- `fromMe` no webhook: mensagens enviadas pelo próprio número (atendente no celular), com
  `from` igual ao contato.
- Correções que estavam só na cópia usada pelo Prospecta: envio serial por conversa, filtro de
  grupo/status/transmissão, remetente pelo telefone em vez do `@lid`, mensagens com mais de 10
  minutos ignoradas, eventos `typing` e `send_failed`.
- `GET /metrics` (Prometheus) e `/health` com versão e sessões por status.
- Logs estruturados (`GATEWAY_LOG_FORMAT=json`, `GATEWAY_LOG_LEVEL`) e desligamento limpo.
- Documentação completa em inglês e português (MkDocs) em
  https://arcanumgateway.com.br, `.env.example` e imagem publicada no GHCR por tag,
  multi-arquitetura, com SBOM, proveniência e assinatura Cosign.

## [1.0.0] - 2026-06-20

### Adicionado

- **Multi-instancia** — crie e gerencie multiplas conexoes WhatsApp simultaneamente
- **Autenticacao** via QR Code e Phone Pairing (codigo de 8 digitos)
- **Envio de mensagens** com anti-ban integrado:
  - Texto (com delay aleatorio + indicador "digitando...")
  - Imagens (JPEG, PNG) com caption
  - Documentos (PDF, DOCX) com filename
  - Audio/voz (OGG, MP3) com indicador "gravando..."
  - Video (MP4) com caption
  - Localizacao (GPS com nome e endereco)
  - Contato (vCard)
  - Reacao (emoji em mensagem existente)
  - Enquete (poll com opcoes)
- **Recebimento de mensagens** com download automatico de midia
- **Webhook forwarding** com assinatura HMAC-SHA256
- **Anti-ban nativo**:
  - Delay aleatorio antes de responder (1.5-4s configuravel)
  - Indicador "digitando..." proporcional ao tamanho do texto
  - Indicador "gravando..." para audios
  - Read receipts antes de responder
  - Rate limiting (15 msg/min por instancia)
- **Configuracoes por instancia** (rejectCalls, groupsIgnore, alwaysOnline, readMessages)
- **Webhook configuravel via API** (URL, eventos, habilitar/desabilitar)
- **Swagger UI** com documentacao interativa em `/docs`
- **Docker** com multi-stage build (Go 1.25 + Alpine)
- **Verificacao de numeros** (checkWhatsAppNumbers)
- **Gerenciamento de perfil** (nome, status, foto)
- **Chat operations** (marcar como lido, arquivar)

### Arquitetura

- Go 1.25 + whatsmeow (engine de protocolo WhatsApp)
- Chi v5 (HTTP router)
- SQLite (sessoes persistentes por instancia)
- Interface `WhatsAppProvider` — provider-agnostico, permite trocar engine
- Zero dependencia de projetos externos
