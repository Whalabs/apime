## Funcionamento

Quando uma mensagem com mídia é recebida:
1. A API baixa e descriptografa o arquivo automaticamente.
2. O arquivo é salvo localmente com um TTL de 2 horas.
3. O webhook inclui um campo `mediaUrl` apontando para a API.

---

## Endpoint de Download

- **Método:** `GET`
- **Caminho:** `/api/media/{instanceId}/{mediaId}`
- **Autenticação:** Não requer.

**Resposta:** O arquivo binário com o `Content-Type` correto.

---

## Expiração (TTL)

Os arquivos de mídia são temporários e removidos automaticamente após **2 horas**. 

Consuma a URL assim que receber o webhook para garantir o acesso.
