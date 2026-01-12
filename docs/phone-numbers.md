### Contatos
Números de telefone podem ser enviados em qualquer um dos formatos:

| Entrada               | Normalizado para       |
|-----------------------|------------------------|
| `5511999999999`       | `551199999999@s.whatsapp.net` |
| `551199999999`        | `551199999999@s.whatsapp.net` |
| `5511999999999@s.whatsapp.net` | (mantido) |

**Nota:** A API adiciona `@s.whatsapp.net` automaticamente se não estiver presente.

---

### Grupos
Grupos devem ser enviados com o JID completo:

```
120363123456789012@g.us
```

A API **não** adiciona sufixo automaticamente para grupos.

---

## Normalização de Números Brasileiros

Para números brasileiros (prefixo `55`), a API remove automaticamente o 9º dígito de celulares quando necessário.

| Entrada (com 9)     | Saída (normalizado)   |
|---------------------|-----------------------|
| `5511999999999`     | `551199999999`        |
| `5521987654321`     | `552187654321`        |

A normalização ocorre apenas quando:
- O número começa com `55`
- O número local tem 9 dígitos e começa com `9`

Números de 8 dígitos ou de outros países não são alterados.
