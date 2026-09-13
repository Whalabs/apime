Endpoints avançados para operações de baixo nível do WhatsApp. Requerem **token de instância** (não JWT).

## Base Path
Todos os endpoints abaixo são prefixados por: `https://whapi.agnus.cloud/api/instances/{id}/whatsapp`

---

## Verificação e Presença

### Verificar Número WhatsApp
```
POST /api/instances/{id}/whatsapp/check
Body: { "phone": "5511999999999" }
```
**Nota:** O retorno inclui detalhes se o número existe e qual o JID correto.

### Definir Presença
```
POST /api/instances/{id}/whatsapp/presence
Body: { "state": "available|unavailable|composing|recording|paused", "to": "JID" }
```

---

## Mensagens

### Marcar Como Lida
```
POST /api/instances/{id}/whatsapp/messages/read
Body: { "chat": "JID", "message_id": "ID", "sender": "JID", "played": false }
```

### Deletar Para Todos
```
POST /api/instances/{id}/whatsapp/messages/delete
Body: { "chat": "JID", "message_id": "ID", "sender": "JID" }
```

---

## Contatos e Perfis

### Listar Contatos
```
GET /api/instances/{id}/whatsapp/contacts
```

### Obter Contato
```
GET /api/instances/{id}/whatsapp/contacts/{jid}
```

### Obter UserInfo
```
GET /api/instances/{id}/whatsapp/userinfo/{jid}
```

---

## Privacidade

### Obter Configurações de Privacidade
```
GET /api/instances/{id}/whatsapp/privacy
```

### Definir Configuração de Privacidade
```
POST /api/instances/{id}/whatsapp/privacy
Body: { "setting": "...", "value": "..." }
```

### Obter Status Privacy
```
GET /api/instances/{id}/whatsapp/status-privacy
```

---

## Chat Settings

### Obter Configurações do Chat
```
GET /api/instances/{id}/whatsapp/chat-settings/{chat}
```

### Definir Configurações do Chat
```
POST /api/instances/{id}/whatsapp/chat-settings/{chat}
Body: { ... }
```

### Definir Mensagem de Status
```
POST /api/instances/{id}/whatsapp/status
Body: { "text": "..." }
```

### Timer de Mensagens que Desaparecem
```
POST /api/instances/{id}/whatsapp/disappearing-timer
Body: { "duration": 86400 }
```

---

## QR Links

### Obter QR de Contato
```
GET /api/instances/{id}/whatsapp/qr/contact
```

### Resolver QR de Contato
```
POST /api/instances/{id}/whatsapp/qr/contact/resolve
Body: { "link": "..." }
```

### Resolver Link de Business Message
```
POST /api/instances/{id}/whatsapp/qr/business-message/resolve
Body: { "link": "..." }
```

---

## Grupos

### Criar Grupo
```
POST /api/instances/{id}/whatsapp/groups
Body: { "name": "Nome", "participants": ["JID1", "JID2"] }
```

### Obter Informações do Grupo
```
GET /api/instances/{id}/whatsapp/groups/{group}
```

### Sair do Grupo
```
POST /api/instances/{id}/whatsapp/groups/{group}/leave
```

### Obter Link de Convite
```
GET /api/instances/{id}/whatsapp/groups/{group}/invite-link
```

### Resolver Link de Convite
```
POST /api/instances/{id}/whatsapp/groups/resolve-invite
Body: { "link": "..." }
```

### Entrar com Link
```
POST /api/instances/{id}/whatsapp/groups/join
Body: { "link": "..." }
```

### Gerenciar Participantes
```
POST /api/instances/{id}/whatsapp/groups/{group}/participants
Body: { "action": "add|remove|promote|demote", "participants": ["JID"] }
```

### Listar Solicitações de Entrada
```
GET /api/instances/{id}/whatsapp/groups/{group}/requests
```

### Aprovar/Rejeitar Solicitações
```
POST /api/instances/{id}/whatsapp/groups/{group}/requests
Body: { "action": "approve|reject", "participants": ["JID"] }
```

---

## Newsletters (Canais)

### Inscrever em Atualizações
```
POST /api/instances/{id}/whatsapp/newsletters/{jid}/live-updates
```

### Marcar Como Visto
```
POST /api/instances/{id}/whatsapp/newsletters/{jid}/mark-viewed
Body: { "server_ids": ["1", "2"] }
```

### Enviar Reação
```
POST /api/instances/{id}/whatsapp/newsletters/{jid}/reaction
Body: { "server_id": "123", "reaction": "👍", "message_id": "ID" }
```

### Obter Atualizações de Mensagens
```
POST /api/instances/{id}/whatsapp/newsletters/{jid}/message-updates
```

---

## Listas de Transmissão

A lista é local do remetente: quem recebe vê uma conversa 1:1 comum e nunca sabe que a lista existe.

### Listar Listas de Transmissão
```
GET /api/instances/{id}/whatsapp/broadcast-lists
```
**Nota:** Cada item traz `jid`, `name`, `participants` (`pn` e `lid` de cada destinatário), `labelIds` e `updatedAt` em RFC3339. As listas vêm da sincronização de app state com o celular, então uma lista criada no aparelho só aparece aqui depois que o app state sincroniza. Não há como criar lista pela API.

### Obter Lista de Transmissão
```
GET /api/instances/{id}/whatsapp/broadcast-lists/{jid}
```
**Nota:** Aceita o JID completo (`123@broadcast`) ou só o id (`123`). Retorna 404 enquanto a lista não tiver chegado pela sincronização com o celular.

---

## Upload de Mídia

### Upload Direto
```
POST /api/instances/{id}/whatsapp/upload
Body: { "media_type": "image|video|audio|document", "data_base64": "..." }
```
Retorna URL para uso em mensagens.
