# Authentication

Every route requires the API key, except `GET /health` and the documentation
(`/docs`, `/docs/openapi.json`).

## Sending the key

Use either header:

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

A missing or wrong key returns:

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="arcanum"

{"error":"missing or invalid API key (send X-Api-Key or Authorization: Bearer)"}
```

## Choosing a key

Set `GATEWAY_API_KEY` to a long random value. 32 random bytes is a good default:

```bash
openssl rand -hex 32
```

| Situation | What happens |
|---|---|
| `GATEWAY_API_KEY` set | That key is required. Keys shorter than 32 characters log a warning. |
| Not set | A random key is generated on every start and printed in the log. |
| `GATEWAY_AUTH_DISABLED=true` | No key at all. **Only for your own machine**: anyone who reaches the port controls your numbers. |

The key is compared in constant time, so response timing does not leak it.

## Rotating the key

1. Generate a new key and update your application to send it.
2. Restart Arcanum with the new `GATEWAY_API_KEY`. Sessions reconnect on their own; nobody
   needs to scan a QR code again.

!!! danger "Keep the key out of the browser"
    The key gives full control over every session. Call Arcanum from your backend, never
    from front-end code that users can inspect.
