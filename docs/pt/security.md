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
