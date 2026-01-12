# Health Check da API

## Endpoint
- **Método:** `GET`
- **Caminho:** `/api/healthz`
- **Autenticação:** não requer tokens nem cookies.

## Resposta esperada
```
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

## Exemplo de uso
```bash
curl -s https://localhost:8080/api/healthz
```
