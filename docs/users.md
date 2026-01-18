### Admin
- Acesso total ao sistema.
- Gerencia todos os usuários e instâncias.
- Pode criar, editar e remover qualquer recurso.

### User
- Acesso restrito às próprias instâncias.
- Não visualiza instâncias de outros usuários.
- Pode criar e gerenciar apenas suas instâncias.

---

## Isolamento de Instâncias

Cada instância possui um `owner_user_id` que define seu proprietário.

| Papel | Visualiza         | Edita/Remove      |
|-------|-------------------|-------------------|
| Admin | Todas as instâncias | Todas as instâncias |
| User  | Apenas as próprias | Apenas as próprias |

---

## Tokens

### Token de Usuário (JWT)
Obtido via `/api/auth/login` ou retornado imediatamente ao criar um usuário (`POST /users`). Permite gerenciar instâncias pelo dashboard ou API.

### Token da Instância
Gerado via dashboard ou `/api/instances/{id}/token/rotate`. Usado para conectar (QR Code), desconectar e operações de mensageria (enviar textos, mídias, etc). Cada instância tem seu próprio token.
