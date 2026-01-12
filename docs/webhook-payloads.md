## Estrutura Base
```json
{
  "id": "uuid-do-evento",
  "instanceId": "id-da-instancia",
  "type": "tipo-do-evento",
  "payload": { ... },
  "createdAt": "2024-01-01T12:00:00Z"
}
```

---

## Segurança (Assinatura)

Se um `webhook_secret` for definido na instância, o ApiMe envia um hash HMAC-SHA256 no header `X-ApiMe-Signature`.
Para validar, gere o HMAC-SHA256 do corpo da requisição usando seu secret e compare com o header.

---

## Tipos de Eventos

### `message`
Mensagem recebida (texto, imagem, áudio, vídeo, documento, sticker, contato ou localização).

| Campo       | Descrição                                      |
|-------------|------------------------------------------------|
| `from`      | JID do remetente                               |
| `to`        | JID do destinatário (chat)                     |
| `isFromMe`  | `true` se enviado pela própria instância       |
| `isGroup`   | `true` se for mensagem de grupo                |
| `messageId` | ID único da mensagem no WhatsApp               |
| `timestamp` | Timestamp Unix                                 |
| `pushName`  | Nome do remetente                              |
| `text`      | Conteúdo (para texto)                          |
| `mediaType` | `image`, `video`, `audio`, `document`, `sticker`, `location`, `contact` |
| `mediaUrl`  | URL local para download da mídia (pré-baixada) |
| `mimetype`  | Tipo MIME do arquivo                           |
| `caption`   | Legenda (imagem/vídeo)                         |

---

### `receipt`
Confirmação de entrega ou leitura.

| Campo          | Descrição                                        |
|----------------|--------------------------------------------------|
| `messageIds`   | Array de IDs confirmados                         |
| `timestamp`    | Timestamp da confirmação                         |
| `chat`         | JID do chat                                      |
| `status`       | `read`, `delivered` ou `played`                  |

---

### `presence`
Mudança de status online/offline.

| Campo        | Descrição                      |
|--------------|--------------------------------|
| `from`       | JID do contato                 |
| `unavailable`| `true` = offline               |
| `lastSeen`   | Última vez online (se offline) |

---

### `connected`
A instância conectou ao WhatsApp.

---

### `disconnected`
A instância desconectou do WhatsApp.
