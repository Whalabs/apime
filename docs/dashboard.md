# Dashboard

Interface web do ApiMe, em `http://<host>:8080/dashboard`. É o único caminho com poder de
**administrador**: a API `/api/*` não carrega role, então listar ou remover instância de outro
usuário só acontece por aqui.

## Autenticação

Cookie `dashboard_token`, um JWT assinado com `JWT_SECRET`, diferente do header `Authorization`
usado pela API. As claims trazem `sub`, `email` e `role`, e o middleware confere a cada requisição
se o usuário ainda existe no banco. Sessão inválida cai em `/dashboard/login`.

Ligar ou desligar a interface: `DASHBOARD_ENABLED`.

## Páginas

| Rota | O que faz |
|---|---|
| `GET /dashboard` | visão geral das instâncias |
| `GET /dashboard/instances` | lista de instâncias |
| `POST /dashboard/instances` | cria instância |
| `POST /dashboard/instances/:id/update` | edita nome, webhook e secret |
| `POST /dashboard/instances/:id/token` | rotaciona o token da instância |
| `GET /dashboard/instances/:id/qr` | tela de conexão por QR |
| `GET /dashboard/instances/:id/qr/status` | estado da conexão, consultado em polling |
| `GET /dashboard/instances/:id/qr/image` | imagem do QR |
| `POST /dashboard/instances/:id/disconnect` | desconecta do WhatsApp |
| `GET /dashboard/instances/:id/diagnostics` | diagnóstico da instância |
| `POST /dashboard/instances/:id/delete` | remove a instância |
| `GET /dashboard/docs` | documentação da API |
| `GET /dashboard/docs/openapi` | baixa o `openapi.yaml` |

## Somente administrador

Exigem `role = admin` na claim do cookie:

| Rota | O que faz |
|---|---|
| `GET /dashboard/users` | lista usuários |
| `POST /dashboard/users` | cria usuário |
| `POST /dashboard/users/:id/password` | troca a senha |
| `POST /dashboard/users/:id/delete` | remove usuário |
| `GET /dashboard/users/:id/tokens` | lista os tokens de API do usuário |
| `POST /dashboard/users/:id/tokens` | gera token de API |
| `POST /dashboard/users/:id/tokens/:tokenID/delete` | revoga token de API |

## Por que isso importa para quem integra

O token de API que as integrações usam nasce aqui, em `/dashboard/users/:id/tokens`. O token de
instância nasce na criação da instância e se troca em `/dashboard/instances/:id/token`. São
credenciais diferentes, com alcances diferentes: o de instância só enxerga a própria instância.
