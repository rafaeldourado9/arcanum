# Autenticação

Toda rota exige a chave de API, menos `GET /health` e a documentação (`/docs` e
`/docs/openapi.json`).

## Enviando a chave

Use qualquer um destes cabeçalhos:

=== "X-Api-Key"

    ```bash
    curl -H "X-Api-Key: $ARCANUM_KEY" http://localhost:3100/api/instance/fetchInstances
    ```

=== "Bearer"

    ```bash
    curl -H "Authorization: Bearer $ARCANUM_KEY" http://localhost:3100/api/instance/fetchInstances
    ```

=== "Python"

    ```python
    import os, httpx

    arcanum = httpx.Client(
        base_url="http://localhost:3100",
        headers={"X-Api-Key": os.environ["ARCANUM_KEY"]},
        timeout=30,
    )
    print(arcanum.get("/api/instance/fetchInstances").json())
    ```

=== "Node.js"

    ```js
    const res = await fetch("http://localhost:3100/api/instance/fetchInstances", {
      headers: { "X-Api-Key": process.env.ARCANUM_KEY },
    });
    console.log(await res.json());
    ```

=== "Go"

    ```go
    req, _ := http.NewRequest("GET", "http://localhost:3100/api/instance/fetchInstances", nil)
    req.Header.Set("X-Api-Key", os.Getenv("ARCANUM_KEY"))
    resp, err := http.DefaultClient.Do(req)
    ```

Sem chave, ou com a chave errada, a resposta é:

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="arcanum"

{"error":"missing or invalid API key (send X-Api-Key or Authorization: Bearer)"}
```

## Escolhendo a chave

Defina `GATEWAY_API_KEY` com um valor longo e aleatório. 32 bytes aleatórios é um bom
padrão:

```bash
openssl rand -hex 32
```

| Situação | O que acontece |
|---|---|
| `GATEWAY_API_KEY` definida | Essa chave é exigida. Chave com menos de 32 caracteres gera um aviso no log. |
| Não definida | Uma chave aleatória é gerada a cada início e mostrada no log. |
| `GATEWAY_AUTH_DISABLED=true` | Nenhuma chave. **Só na sua própria máquina**: quem alcançar a porta controla seus números. |

A chave é comparada em tempo constante, então o tempo de resposta não a revela.

## Trocando a chave

1. Gere uma chave nova e atualize sua aplicação para enviá-la.
2. Reinicie o Arcanum com a nova `GATEWAY_API_KEY`. As sessões reconectam sozinhas;
   ninguém precisa ler QR code de novo.

!!! danger "Mantenha a chave longe do navegador"
    A chave dá controle total sobre todas as sessões. Chame o Arcanum pelo seu backend,
    nunca por código de front-end que o usuário consegue inspecionar.
