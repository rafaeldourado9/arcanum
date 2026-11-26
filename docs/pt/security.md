# Segurança

O Arcanum controla contas reais de WhatsApp. Quem alcança a API, ou copia um arquivo de
sessão, consegue ler e enviar mensagens como aquele número. Trate como um banco de dados.

## Antes de ir para o ar

- [x] `GATEWAY_API_KEY` com 32+ caracteres aleatórios.
- [x] `GATEWAY_AUTH_DISABLED` **não** definida.
- [x] `GATEWAY_DASHBOARD_PASSWORD` definida (ou `GATEWAY_DASHBOARD=false` se você não usa o painel).
- [x] Porta 3100 fechada para a internet: só um proxy reverso com HTTPS chega nela.
- [x] `GATEWAY_META_APP_SECRET` definido, e seu webhook confere o `X-Hub-Signature-256`.
- [x] Backup do volume de dados **criptografado** e com acesso restrito.
- [x] A chave de API fica só no seu backend, nunca em código de front-end.
- [x] Atualizações aplicadas com frequência (as versões novas corrigem falhas e acompanham o WhatsApp).

## O que o Arcanum faz por você

| Área | Proteção |
|---|---|
| Autenticação | Chave de API em toda rota menos `/health`, `/docs` e o login do painel; comparação em tempo constante; chave aleatória gerada quando nenhuma é definida, para nunca ficar aberto por esquecimento. |
| Entrada | Nome de sessão restrito a `A-Z a-z 0-9 - _` (vira nome de arquivo: sem path traversal). Limite de tamanho de pedido e timeouts HTTP. |
| Webhooks (SSRF) | Só `http`/`https`; endereços de metadados da nuvem e link-local recusados, conferidos de novo na conexão (DNS rebinding); redirecionamentos não são seguidos; sem credenciais na URL. |
| Mídia | Só base64: o Arcanum nunca baixa URLs por você. |
| Privacidade | Texto de mensagem, telefone e nome de contato não vão para o log no nível `info`. |
| Cabeçalhos HTTP | `nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy` restrita, `Cache-Control: no-store`, `Referrer-Policy: no-referrer`. |
| Painel | Senha própria, cookie de sessão `HttpOnly` + `SameSite=Strict`, cabeçalho anti-CSRF em todo pedido, bloqueio do login depois de 10 erros, CSP restrita, sessões guardadas por hash na memória. Veja [Painel](dashboard.md). |
| Container | Imagem distroless (sem shell, sem gerenciador de pacotes), usuário sem privilégio (UID 65532), funciona com sistema de arquivos somente leitura e sem capabilities; build reprodutível com `go mod verify`; assinada com Cosign, com SBOM e proveniência. |
| Arquivos | Arquivos de sessão e de configuração criados com permissão `600`, pasta de dados `700`. |
| systemd | O serviço fornecido roda isolado: sem novos privilégios, sistema somente leitura, `/tmp` privado, sem capabilities, chamadas de sistema filtradas. |
| Desligamento | Limpo no `SIGTERM`: os arquivos de sessão fecham direito. |
| Cadeia de dependências | Toda mudança passa por detector de corrida, `go vet`, `staticcheck`, `govulncheck`, `gosec` e varredura da imagem com Trivy. O Dependabot atualiza as dependências toda semana. |

## Hospedar para vários clientes (chaves com escopo)

Você pode rodar o Arcanum num servidor e dar a cada integrador (agente de IA, sistema de
terceiros, outra empresa) acesso **só às sessões dele**, sem entregar a chave mestra. O
integrador não precisa saber quem são os outros clientes: ele só enxerga as próprias sessões.

```bash
# 1. Você (admin, chave mestra) cria a sessão e a chave dela
curl -X POST http://localhost:3100/api/instance/create -H "X-Api-Key: $GATEWAY_API_KEY" \
  -d '{"instanceName":"loja-joao","webhook":"https://agente.exemplo.com/whatsapp"}'

curl -X POST http://localhost:3100/api/keys -H "X-Api-Key: $GATEWAY_API_KEY" \
  -d '{"name":"agente-loja-joao","instances":["loja-joao"],"readOnly":false}'
# resposta: {"secret":"arc_..."}  <- aparece UMA vez; entregue ao integrador

# 2. Segredo de assinatura só dessa sessão (o integrador confere o X-Hub-Signature-256 com ele)
curl -X POST http://localhost:3100/api/events/webhook/loja-joao/secret -H "X-Api-Key: $GATEWAY_API_KEY"
```

| Chave | O que alcança |
|---|---|
| Mestra (`GATEWAY_API_KEY`) e painel | Tudo. |
| Com escopo (`arc_...`) | Só as sessões listadas: envia, configura o webhook e lê o estado delas. Sessão de outro cliente responde `404`, igual a uma que não existe. Não cria nem apaga sessão, não gerencia chaves, não lê métricas, logs nem auditoria. |
| Com escopo `readOnly` | Só consulta: estado, configuração, entregas, busca. Não envia, não muda configuração e não inicia conexão. |

- **Revogar** vale na hora: `DELETE /api/keys/{id}`. A chave continua listada como revogada.
- No disco fica só o **hash** da chave (arquivo `600` em `<dados>/.arcanum/keys.json`).
- **Segredo de webhook por sessão:** cada sessão pode ter o seu; a API nunca o mostra depois
  (só `hasSecret`). Sem segredo próprio, vale o global (`GATEWAY_META_APP_SECRET`).
- **Bloqueio:** 20 chaves erradas do mesmo IP em 15 minutos bloqueiam o IP (`429`) por 15
  minutos. Pedido sem chave não conta (o painel pergunta antes do login). Atrás de proxy todos
  chegam do IP do proxy, então o limite vira global.
- **Auditoria:** `GET /api/audit` (só admin) lista quem fez o quê: login, criar/apagar/reconectar
  sessão, mudar configuração ou webhook, criar/revogar chave e acessos negados. Fica em
  `<dados>/.arcanum/audit.log` (`600`, gira em 5 MB). Nunca contém chaves, segredos, telefones
  nem texto de mensagens.

## Reportar uma vulnerabilidade

Não abra uma issue pública. Use **Security > Report a vulnerability** no
[repositório do GitHub](https://github.com/rafaeldourado9/arcanum/security). A resposta vem
em até 72 horas. Detalhes no
[SECURITY.md](https://github.com/rafaeldourado9/arcanum/blob/main/SECURITY.md).

## Privacidade e LGPD

As mensagens passam pela memória do Arcanum a caminho do seu webhook; o Arcanum não guarda
histórico de mensagens. Os arquivos de sessão têm chaves de criptografia e identificadores
de contatos. Se você atende pessoas no Brasil (LGPD) ou na Europa (GDPR), você é o
controlador dos dados: documente a finalidade, proteja o servidor e o endpoint do webhook e
atenda pedidos de descadastro.
