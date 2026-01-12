Endpoints avançados para operações de baixo nível do WhatsApp. Requerem **token de instância** (não JWT).

## Base Path
`/api/instances/{id}/whatsapp/`

---

## Verificação e Presença

### Verificar Número WhatsApp
```
POST /check
Body: { "phone": "5511999999999" }
```

### Definir Presença
```
POST /presence
Body: { "state": "available|unavailable|composing|recording|paused", "to": "JID" }
```

---

## Mensagens

### Marcar Como Lida
```
POST /messages/read
Body: { "chat": "JID", "message_id": "ID", "sender": "JID", "played": false }
```

### Deletar Para Todos
```
POST /messages/delete
Body: { "chat": "JID", "message_id": "ID", "sender": "JID" }
```

---

## Contatos e Perfis

### Listar Contatos
```
GET /contacts
```

### Obter Contato
```
GET /contacts/{jid}
```

### Obter UserInfo
```
GET /userinfo/{jid}
```

---

## Privacidade

### Obter Configurações de Privacidade
```
GET /privacy
```

### Definir Configuração de Privacidade
```
POST /privacy
Body: { "setting": "...", "value": "..." }
```

### Obter Status Privacy
```
GET /status-privacy
```

---

## Chat Settings

### Obter Configurações do Chat
```
GET /chat-settings/{chat}
```

### Definir Configurações do Chat
```
POST /chat-settings/{chat}
Body: { ... }
```

### Definir Mensagem de Status
```
POST /status
Body: { "text": "..." }
```

### Timer de Mensagens que Desaparecem
```
POST /disappearing-timer
Body: { "duration": 86400 }
```

---

## QR Links

### Obter QR de Contato
```
GET /qr/contact
```

### Resolver QR de Contato
```
POST /qr/contact/resolve
Body: { "link": "..." }
```

### Resolver Link de Business Message
```
POST /qr/business-message/resolve
Body: { "link": "..." }
```

---

## Grupos

### Criar Grupo
```
POST /groups
Body: { "name": "Nome", "participants": ["JID1", "JID2"] }
```

### Sair do Grupo
```
POST /groups/{group}/leave
```

### Obter Link de Convite
```
GET /groups/{group}/invite-link
```

### Resolver Link de Convite
```
POST /groups/resolve-invite
Body: { "link": "..." }
```

### Entrar com Link
```
POST /groups/join
Body: { "link": "..." }
```

### Gerenciar Participantes
```
POST /groups/{group}/participants
Body: { "action": "add|remove|promote|demote", "participants": ["JID"] }
```

### Listar Solicitações de Entrada
```
GET /groups/{group}/requests
```

### Aprovar/Rejeitar Solicitações
```
POST /groups/{group}/requests
Body: { "action": "approve|reject", "participants": ["JID"] }
```

---

## Newsletters (Canais)

### Inscrever em Atualizações
```
POST /newsletters/{jid}/live-updates
```

### Marcar Como Visto
```
POST /newsletters/{jid}/mark-viewed
Body: { "server_ids": ["1", "2"] }
```

### Enviar Reação
```
POST /newsletters/{jid}/reaction
Body: { "server_id": "123", "reaction": "👍", "message_id": "ID" }
```

### Obter Atualizações de Mensagens
```
POST /newsletters/{jid}/message-updates
```

---

## Upload de Mídia

### Upload Direto
```
POST /upload
Body: { "media_type": "image|video|audio|document", "data_base64": "..." }
```
Retorna URL para uso em mensagens.
