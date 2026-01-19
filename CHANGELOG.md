# Changelog

## [v1.0.7] - 2026-01-19
### Corrigido
- **Race condition no pareamento**: salvamento do JID agora é síncrono, garantindo que a sessão esteja persistida antes de liberar para uso.
- Mensagens ficando em "Aguardando" após conectar instância: adicionado controle `sessionReady` que indica quando a sessão está pronta para enviar mensagens (após presence enviado).

### Adicionado
- Método `IsSessionReady()` no Manager para verificar se a sessão está completamente pronta para operações de envio.

### Internals
- Limpeza automática do estado `sessionReady` e `pairingSuccess` ao deletar sessão.

## [v1.0.6] - 2026-01-18
### Adicionado
- Implementação de **Smart Debounce** para estabilização de conexão: desconexões acidentais são filtradas por 5 segundos antes de disparar webhook, enquanto desconexões intencionais (API/Logout) são notificadas imediatamente.

### Corrigido
- Evento `LoggedOut` agora dispara webhook corretamente com status `disconnected`, garantindo notificação imediata ao sair pelo celular.
- Endpoints de operação da instância (`/qr` e `/disconnect`) agora aceitam autenticação via Token de Instância (API Key) além do JWT, facilitando integrações externas.

### Atualizado
- Documentação do Dashboard (`docs.tmpl`) revisada para exibir exemplos de cURL com `$INSTANCE_TOKEN` nas operações de conexão.
- Guia do Usuário (`docs/users.md`) atualizado para esclarecer o escopo de uso dos tokens de instância.

## [v1.0.5] - 2026-01-18
### Corrigido
- Isolamento das sessões WhatsMeow em PostgreSQL: cada instância agora cria um `deviceStore` dedicado, tem o JID salvo no banco e as operações de restauração/remoção usam esse identificador exclusivo.

### Adicionado
- Campo `whatsapp_jid` na tabela `instances`, com migrações para PostgreSQL (`000010_whatsapp_jid`) e SQLite (`000003_whatsapp_jid`), além do suporte correspondente nos repositórios e no modelo `Instance`.


## [v1.0.4] - 2026-01-17
### Adicionado
- Suporte a `DASHBOARD_TIMEZONE`: a variável de ambiente agora é lida pelo backend, propagada aos templates e exposta via JS para garantir que todas as datas/hora do dashboard sigam o fuso configurado.

### Atualizado
- Helpers `formatTime` e `formatOptionalTime` usam a localização configurada, enquanto o layout fornece utilitário `formatDateTime` para o frontend.
- Todos os formulários críticos (instâncias, tokens, usuários, configurações) contam com bloqueio de duplo envio, estados de carregamento e feedback consistente.
- Botão de desconexão das instâncias ganhou spinner embutido, evitando flicker do texto “Desconectando...” e mantendo o visual harmônico.

## [v1.0.3] - 2026-01-17
### Internals
- Limpeza dos templates do dashboard: remoção de comentários redundantes nos arquivos de docs, diagnostics, QR, instances, layout, login e users para reduzir ruído visual mantendo apenas o código relevante.

## [v1.0.2] - 2026-01-17
### Adicionado
- Endpoint raiz agora responde com status, versão e nome da aplicação, enquanto o dashboard passou a receber `config.Version` para exibir a build atual.
- Nova variável `MEDIA_TTL_SECONDS` documentada no `.env.example` e nos READMEs para ajustar o TTL de mídias armazenadas via configuração.
- Captura de tela em modo claro (`docs/dashboard_light.png`) adicionada à documentação para destacar o novo tema visual do dashboard.

### Atualizado
- Fluxo de inicialização da API e do manager do WhatsMeow suporta DSN explícito para PostgreSQL, expõe diagnósticos de armazenamento e adiciona a dependência `github.com/lib/pq` para compatibilidade.
- Página "Docs" do dashboard recebeu um redesenho completo, com navegação lateral interativa, modos claro/escuro aprimorados e blocos de código expansíveis.
- O `docker-compose.scalable.yml` ganhou healthchecks para Postgres/Redis, rede dedicada e dependências condicionais para garantir subida ordenada dos serviços.

### Corrigido
- Logs do pool de webhooks agora incluem prefixos por worker e mensagens consistentes em todo o fluxo, facilitando o troubleshooting de entregas.
- Consultas de history sync e instances convertem `history_sync_cycle_id` para texto, evitando erros de tipo no Postgres.

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
