# Plataforma eCOnectiva — implementação local de 20/10/2026

Referência: [planejamento consolidado, revisão 2589](https://wiki.spotec.app/index.php/Plataforma_Rieonlife_-_Planejamento_de_altera%C3%A7%C3%B5es_e_requisitos), considerando a reunião de 19/10/2026. Base: `origin/main` em `b7dedb2`. Branch local: `UU-95-primeira-versao-plataforma-econectiva`.

## Contratos e persistência

| Requisito | Implementação |
|---|---|
| REQ-01 | `users.usage_mode` anulável (`explorer`, `inventor`). `GET/PUT /api/me/preferences` consulta/grava a própria conta. O atributo também integra as respostas existentes do usuário. Perfil continua independente. Middleware bloqueia autoria estrutural de contas explicitamente em Explorador, permitindo preferências, Locker, comentários e conversas de IA. Contas antigas sem escolha permanecem compatíveis com clientes antigos; o aplicativo novo exige escolha. |
| REQ-02 | `locker_items`: chave única conta/tipo/id; PUT idempotente, DELETE pessoal, GET ordenado por salvamento decrescente. Tipos: `living_clue`, `node`. O original é resolvido com autorização na consulta. Referências indisponíveis continuam removíveis sem revelar seu conteúdo. |
| REQ-03/04 | `description_delta` aceita texto e `{"video":"URL YouTube"}`. Validação limita os hosts e exige ID de vídeo. `description` é derivada do texto autoral, sem transcrição de vídeo. Eventos em `clue_edits` são gravados na mesma transação da edição; identidade e horário vêm do servidor. Cache em `link_previews`, associado ao documento por `clue_preview_references`; a associação é recalculada em cada salvamento, sem duplicar URLs ou vídeos incorporados. |
| REQ-05 | `clue_comments`: texto até 5000 caracteres Unicode, autor autenticado, comentário raiz e destinatário da resposta. Consulta paginada dos comentários principais em ordem decrescente; `root_id` recupera respostas em ordem crescente. PUT/DELETE exigem autoria da mensagem. Exclusão mantém uma lápide e todas as respostas. |
| REQ-06 | Latitude/longitude são ponteiros anuláveis. Ausência integral é válida; pares incompletos e coordenadas fora do intervalo são rejeitados. Filtros SQL geográficos excluem nulos naturalmente. Sem coordenadas não se agenda nem consulta a Geo API. |

Todos os endpoints novos exigem autenticação. Identificação em comentários e histórico contém apenas ID e nome, sem e-mail. As operações de participação são independentes do documento e dos históricos privados de IA.

### Atualização da localização

- Par válido: substitui as coordenadas, incluindo valores zero.
- Ambos os campos omitidos: mantém a localização existente para compatibilidade.
- `remove_location: true`: limpa o par integralmente, mesmo se o cliente enviar coordenadas.
- `null/null` sem o sinalizador tem a mesma semântica de omissão. O aplicativo utiliza o sinalizador para remoção explícita.
- Remover/alterar invalida o contexto anterior na transação da edição. A gravação de resultados assíncronos bloqueia a linha da pista, confere as coordenadas e a geração do contexto. Reprocessamento cria uma geração com novo ID, impedindo que um trabalho antigo restaure resultados mesmo após mover A → B → A.

### Endpoints adicionais

```
GET, PUT          /api/me/preferences
GET               /api/me/locker
PUT, DELETE       /api/me/locker/{living_clue|node}/{entity_id}
GET, POST         /api/living_clues/{id}/comments
PUT, DELETE       /api/living_clues/{id}/comments/{comment_id}
GET               /api/living_clues/{id}/edits
GET               /api/living_clues/{id}/link_previews
GET               /api/link_previews?url=...
```

Comentários e histórico: `limit` padrão 20, máximo 100; `offset` não negativo. Respostas: `root_id` na consulta de comentários. Respostas a respostas conservam o mesmo `root_id`, com `reply_to_id` apontando ao destinatário real.

Prévias: HTTP(S) público, timeout de 7 segundos, limite de leitura 1 MiB, até 4 redirecionamentos, resolução DNS com conexão ao IP verificado, sem proxy e com bloqueio de IP privado/local/link-local/CGNAT. O cache tem validade de 24 horas. Falhas de metadados não alteram nem impedem salvar o documento. URLs maiores que o índice do cache são consultáveis, mas não armazenadas nesse cache; referências por pista utilizam hash e preservam o endereço integral.

## Migração e operação

A inicialização mantém a estratégia AutoMigrate existente: adiciona o atributo, as tabelas novas e torna as coordenadas anuláveis. Não converte zeros em ausência e não atribui modos nem autores a registros antigos. Nenhuma migração foi executada contra o banco operacional; os testes utilizaram MariaDB 11 isolado, publicado apenas em `127.0.0.1:33419`.

Os contratos Swagger foram regenerados com `swag init --parseDependency --parseInternal`.

## Verificação

Suíte completa: `go test ./...`, com credenciais de um banco local vazio passadas por variáveis `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` e `GIN_MODE=test`. Os testes antigos com e-mails fixos exigem um schema novo a cada execução completa.

Os testes novos cobrem duas contas, modo sem concessão de privilégios, idempotência e isolamento do Locker, mensagem em branco, Unicode, respostas a respostas, autoria, exclusão com respostas, história transacional, pista sem coordenadas, zero válido, manutenção por omissão, remoção explícita, resultado geográfico obsoleto, processamento geográfico normal, associação/removal de URLs e recusa de prévias privadas. Corrigiu-se um teste antigo que pressupunha exatamente dois agentes globais mesmo quando outras fixtures já existiam; agora verifica a presença dos dois agentes criados e continua verificando isolamento das mensagens.

## Limites deliberados do planejamento

Não foram acrescentados Genially/RA/geração de imagens ou vídeo, editor de cenas/ramificações, notificações, colaboração em tempo real, trechos no Locker ou leitura automática de conteúdo externo pela IA. Essas propostas ficaram para detalhamento posterior no planejamento consolidado. A Geo API e o agente Python não precisaram de alterações de contrato.

## Resultado registrado

Em 20/10/2026: 155 testes/casos da API aprovados, 22 testes Flutter aprovados e 1 fluxo de integração aprovado no emulador Android API 35. O teste Android compilou e instalou o APK e verificou primeiro acesso, escolha de modo, Locker e abertura de pista sem localização com documento rico e player. A inicialização passou a aguardar seu preparo e a publicação de mudanças de rota é feita após o quadro, evitando navegação prematura e reconstrução durante a montagem do navegador.

## Contexto ambiental exclusivamente pela Geo API — 20/10/2026

Removidos modelos, requisições, políticas, repositórios, controllers e endpoints dos cadastros manuais de biomas/territórios. As pistas não possuem mais `territory_id`, associação `Territory`, filtro territorial manual nem referências desses cadastros no contexto do chat. O campo textual `environmental_context.biome`, os cálculos, fontes, camadas e jobs da Geo API permanecem.

A migração idempotente remove `living_clues.biome_id` (bases antigas), `living_clues.territory_id` e respectivas FKs, depois `territories` e `biomes`. Foi aplicada apenas à cópia local, após backup privado `/home/odair/backups/rieonlife/2026-10-20/antes-remover-biomas-territorios.sql.gz` (SHA-256 `d5ef7a021f4a0be4798dfb6fbed787fb1ea2ff21d975d20c99b5d6eb6fd08c0b`). Preservadas as 46 pistas, 27 contextos e 9 categorias existentes antes da migração. Nenhuma alteração ou deploy no servidor.

Validação: teste de migração no MySQL 8 com tabelas/FKs legadas e contexto Geo API, execução repetida e preservação de dados. Suíte Go final: 154 testes/casos aprovados (integração em MariaDB descartável e migração em MySQL 8 descartável). A execução inicial de toda a suíte no MySQL 8 encontrou três falhas preexistentes de autenticação por datas zeradas em `email_verified_at` sob modo estrito; não foi alterado o modo SQL nem o cadastro de usuários neste escopo. Endpoints antigos retornam 404, pistas e catálogo Geo API retornam 200; Swagger regenerado.


## Registro da entrega

Tarefa: [UU-95](https://spo-tec.atlassian.net/browse/UU-95). Branch: `UU-95-primeira-versao-plataforma-econectiva`, compartilhada com o repositório Flutter. Entrega para revisão humana; sem merge ou deploy de produção.

Verificação final da UU-95: `go test ./... -count=1` aprovado com 154 testes/casos, em bases descartáveis. A branch contém exclusivamente a entrega local preparada para revisão.
