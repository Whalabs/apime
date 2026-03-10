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

## Resolução Dinâmica de Números Brasileiros

Para números brasileiros (prefixo `55`), a API utiliza uma verificação dinâmica para determinar o formato correto (com ou sem o 9º dígito), consultando diretamente os servidores do WhatsApp.

### O Processo de Validação
O código distingue fixos de celulares pela regra ANATEL: números que iniciam com 2-5 após o DDD são fixos e **nunca** recebem o 9º dígito.

1. **Geração de Candidatos**:
   - Se o número tem **13 dígitos** (ex: `5511999998888`), o sistema gera uma opção **sem** o 9º dígito (`551199998888`).
   - Se o número tem **12 dígitos** e **não é fixo** (prefixo após DDD não é 2-5), o sistema gera uma opção **com** o 9º dígito.
   - Se o número tem **12 dígitos** e **é fixo** (prefixo 2-5, ex: `554141010021`), **não** gera variante com 9.
   
2. **Consulta (IsOnWhatsApp)**:
   - O sistema envia ambos os formatos para a API do WhatsApp para verificar qual deles (ou ambos) possui uma conta ativa.
   
3. **Decisão**:
   - Se o WhatsApp confirmar que um dos formatos existe, esse JID é utilizado.
   - Se ambos existirem, o primeiro retornado é usado.
   - Se o WhatsApp retornar vazia (número não registrado) para todas as tentativas, o sistema **NÃO enviará a mensagem** e retornará um erro `JID inválido: número não registrado no WhatsApp`. 
