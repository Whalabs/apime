# Changelog

## [v1.0.1] - 2026-01-17
### Adicionado
- Changelog inicial seguindo o ponto de partida marcado pelo tag `v1.0.0`.

### Corrigido
- Ativação do modo `ManualHistorySyncDownload` para liberar o dispositivo imediatamente após o pareamento e evitar travamentos na tela de sincronização do QR code.
- Worker de history sync simplificado para concluir ciclos sem bloquear o login enquanto o modo manual está ativo.
- Logs mais claros sobre o estado da instância (conexão, sincronização crítica e presença) para facilitar o diagnóstico do fast login.

### Internals
- Limpeza do pipeline de eventos de History Sync para impedir tentativas de desserialização inválidas e reduzir ruído de erros.

## [v1.0.0] - 2026-01-17
- Versão inicial publicada.
