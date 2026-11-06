# Educação Onlife — execução das integrações e IA

Fonte de requisitos: [plano aprovado na Wiki](https://wiki.spotec.app/index.php/Plataforma_Educa%C3%A7%C3%A3o_Onlife_-_Planejamento_de_integra%C3%A7%C3%B5es_e_intelig%C3%AAncia_artificial), revisão 2592, consultada em 21/10/2026. Este documento registra evidências e pendências; não substitui nem reduz os critérios do plano. Trabalho na `main` dos repositórios existentes. Não remover dados nem históricos do servidor.

| Frente | Entrega e evidência necessárias | Estado verificado atual |
|---|---|---|
| Contrato das pistas | Blocos versionados, validação dos dois lados, texto/YouTube compatíveis, desconhecidos preservados, proteção contra editores antigos | Contrato implementado; testes de serviço e API isolada aprovados |
| Autoria | Cartões com nome/ícone/explicação; inserir no cursor guardado, editar/remover e reabrir sem perda | Inserção/gravação/reabertura de Genially, FrameVR e Padlet reais; edição Genially validada; remoção UI e falhas em conferência |
| Visualização | Todos os blocos na ordem correta, componentes por provedor, ampliação/acesso externo e falhas isoladas | Inserção/gravação/reabertura de Genially, FrameVR e Padlet reais; edição Genially validada; remoção UI e falhas em conferência |
| Genially/Padlet | Persistência e uso real de exemplos, responsividade e alternativa para conteúdo restrito | Exemplos reais lidos e ampliados; contribuição, restrições e navegador estreito ainda pendentes |
| FrameVR | Cartão sob demanda, mídia apenas por ação, descarte ao sair, alternativa externa e sala real | Visitante no Chrome e cena incorporada confirmados; descarte conferido; mídia/iframe Chrome pendentes, WebRTC incompatível no navegador interno |
| Overte/Vircadia | Contratos e cliente externo; investigar referências oficiais; não prometer iframe ou homologação não testada | Referências oficiais investigadas; contrato/cartões com fixtures, sem cliente local ou ambiente do grupo homologado |
| Metaverse | Investigar produto; manter opção operacional oculta até identificação | Produto não identificado no plano |
| Android/iOS | Ocultar recursos não homologados, preservar blocos no documento e não quebrar builds | Recursos sem suporte ocultos, preservação em testes Flutter VM e APK debug arm64 compilado; iOS não compilado neste host Linux |
| Jitsi | `meet.spotec.app`, stack oficial fixado e isolado, HTTPS/UDP, organizadores autenticados, convidados, cockpit e rollback | Stack/HTTPS/UDP publicados; renovação dry-run passou; nova aba validada na prévia; definição de organizadores e confirmação de apresentação pendentes |
| Coexistência/Jitsi | Registrar serviços antes/depois; chamada com três participantes, redes distintas, áudio/vídeo/tela/reconexão; não declarar capacidade não medida | Endpoints existentes normais; três participantes, redes distintas/móvel, áudio/vídeo/tela/reconexão e métricas sob chamada pendentes |
| Contexto Flutter | IDs estruturados e dados atuais em cockpit/mapa/listagem/grafo/pista/autoria/agentes; restaurar contexto ao retornar | Implementado e testado; cockpit, pista e rascunho validados com Gemini no navegador |
| Gemini | Modelo explícito `gemini-3.8-flash`, credencial protegida no backend, cabeçalho, timeout/erros sem segredos | Configurado no backend remoto protegido; metadados do modelo HTTP 200 no servidor e geração real/benchmark no ambiente sintético |
| MCP | SDK oficial compatível; endpoint autenticado, cliente e sete consultas autorizadas somente leitura; limites/paginação | Implementado; testes HTTP/SDK/banco e benchmark real aprovados |
| Dois chats | Contexto de pista completo no backend, consultas e fontes nos dois fluxos, históricos separados, referências acionáveis | Benchmark e interface real aprovados, identidade da pista e ausência de leituras corretas; estado das análises preservado em teste de navegação |
| Fontes | Somente links vinculados, extração real limitada, cache, proteção de DNS/redirecionamentos e conteúdo não confiável | Implementado; fixtures, benchmark real e leitura pública com transporte de produção aprovados |
| Benchmark | 19 cenários do plano, duas modalidades, fixtures isoladas, três repetições reais, seleção/JSON/métricas, erros reprovam e saída não zero | Referência Gemini v4: 57/57 execuções e 72 etapas aprovadas; ver auditoria abaixo |
| Entrega | Testes, navegador amplo/estreito, OpenAPI, configuração/deploy/rollback e auditoria por requisito | API e frontend publicados com retorno preservado; todos os pacotes Go e 88 testes Flutter passaram; auditoria final e validações externas/móveis em andamento |

## Evidências e decisões técnicas

- Repositórios limpos e atualizados com `origin/main` no início. A tarefa histórica UU-95 está Done; não foi reaberta automaticamente.
- O endpoint de metadados Gemini foi consultado com a credencial autorizada em memória e cabeçalho `x-goog-api-key`. Nenhum valor foi impresso ou versionado. Acesso ao modelo não comprova ainda geração, ferramentas ou benchmark.
- Inventário remoto somente leitura: Apache ocupa 80/443; API e Geo API ativas; aproximadamente 39 GiB livres, 7,6 GiB RAM total. Nenhum serviço reiniciado nem configuração remota alterada nesta inspeção.
- Novos blocos usam chave `onlife` e payload JSON `{v:1, provider, url, title?, instructions?}`. A API também aceita objeto para interoperabilidade; o Flutter grava o payload em string para seguir o contrato `BlockEmbed` do Quill. Provedores/versões desconhecidos são dados opacos, nunca HTML executável.
- Edições de documentos com blocos estendidos exigem `description_format_version: 1`, impedindo que um editor antigo transforme os novos blocos em texto sem perceber. Atualizar API antes do frontend; manter esse campo nos clientes novos.

## Auditoria de encerramento

Cada linha deverá apontar para testes, resultados e evidências reais. Pendências de produto, ambiente, quota ou homologação devem permanecer explícitas. Uma execução não realizada não conta como aprovação. Não marcar esta frente concluída enquanto os critérios aplicáveis da revisão 2592 não estiverem demonstrados.

### Checkpoint — contrato dos blocos

`go test ./services` aprovado. `TestClueIntegrationsPersistReopenEditRemoveAndProtectLegacyEditors` aprovado com banco novo `onlife_validation_integrations` no contêiner MariaDB local, porta 33419; nenhum banco remoto foi usado. Esse teste verifica criação/GET/PUT/GET/remoção, ordem, texto derivado, autoria negada ao Explorador, autenticação e conflito para cliente antigo. Flutter: 18 testes de integrações/conteúdo/Quill aprovados e build web concluído. A validação de acesso real aos provedores e o restante do plano continuam pendentes.


### Checkpoint — MCP, fontes e ciclo Gemini

- SDK oficial MCP `v1.3.1`; linguagem Go 1.23, toolchain local 1.23.4 e imagem existente Go 1.23.6 compatíveis. Endpoint autenticado `/api/mcp/iarete`, transporte Streamable HTTP e cliente MCP real também usado dentro do orquestrador. Sete ferramentas somente leitura; limites de paginação, chamada e tamanho; IDs reautorizados em cada consulta. Testes `go test ./services ./mcpserver` aprovados.
- `TestIareteMCP*`, `TestAI*`, `TestLivingClueChat*` e `TestClueIntegrations*` aprovados com `scripts/run-isolated-tests.py`, exclusivamente no banco sintético `onlife_validation_integrations`, sem chave Gemini e sem acesso aos bancos de usuário/servidor. A autenticação, o bloqueio de mutações/argumentos fora do contrato, pesquisas com/sem geografia, campos privados e o isolamento da pista no chat específico são exercitados.
- Fontes são obtidas pelos IDs associados à pista autorizada. HTML/texto reais, cache limitado de 15 minutos, limites de corpo/texto/tempo, validação e fixação DNS, bloqueios de IP privado/reservado, credenciais e redirecionamentos. Provedores interativos são informados como referências, sem afirmar leitura de vídeos ou mundos. Testes incluem conteúdo controlado, scripts ignorados, cache e rejeições de URLs.
- Gemini usa `gemini-3.8-flash` explicitamente, chave em `x-goog-api-key`, prazo global de 45 segundos, seis rodadas/oito consultas e orçamento de resultados/histórico. Preserva partes e assinaturas de raciocínio no retorno de funções. Erros de quota, configuração, modelo e resposta inválida não expõem a credencial nem acionam outro modelo.
- Chamada real sintética com o modelo 3.8 e cliente MCP oficial: uma consulta executada, nome/link da fixture na resposta, 4.014 tokens informados pelo provedor e aproximadamente 10,4 segundos. Evidência local protegida `/tmp/onlife-gemini-tool-smoke.json`; não contém dados SUSI. Isso comprova um ciclo real, ainda não o benchmark completo nem o fluxo Flutter autenticado.
- Históricos global e da própria pista continuam separados; a pista é carregada pelo backend por referência estruturada e autorização. O novo `screen_context` é dado de interface, limitado/sanitizado, com modo obtido da conta. O restante da cobertura Flutter e as referências acionáveis estão em implementação.
- Configuração persistente e deploy remoto ainda não realizados neste checkpoint. Nada foi reiniciado ou excluído no servidor. Benchmark de 19 cenários e Jitsi permanecem pendentes.


### Proteção da inicialização para o deploy

O bootstrap passa a preservar integralmente contas administrativas existentes (inclusive desativadas), sem redefinir senha, nome ou perfil. Migrações históricas que removem catálogos geográficos e conexões antigas deixam de executar automaticamente: exigem `ALLOW_DESTRUCTIVE_LEGACY_MIGRATIONS=true`, que **não será habilitado nesta entrega**. Não se trata de autorização para eliminar dados no servidor. Erros de conexão não imprimem o DSN. Antes do deploy ainda serão feitos backup e contagens comparativas do banco.


### Checkpoint — executor do benchmark

Matriz dos 19 cenários em `tests/benchmark/cases.json`; executor Go com prompts/orquestração/MCP/repositórios reais e banco dedicado `onlife_validation_benchmark`. A fonte externa usa o extrator real com transporte controlado; apenas o transporte do provedor é simulado no modo determinístico. `go test ./tests/benchmark/...` testa seleção, expectativas, resposta inválida, contradição, IDs/resultados faltantes e erro do provedor. Execução determinística: **57/57 aprovadas**, três repetições por cenário; relatório protegido local `/tmp/onlife-benchmark-deterministic.json`. A execução Gemini e sua auditoria ainda são pendentes; este resultado não comprova raciocínio do modelo.

Validação Flutter autenticada em `http://127.0.0.1:8773`, API isolada `127.0.0.1:8082`: login com conta sintética, pergunta no cockpit (modo Inventor/Terra), mudança para registro da pista e nova pergunta contextual. A resposta consultou 18 árvores, metadado e ausência de coordenadas; referência acionável retornou ao registro correto. Histórico real de usuários não foi alterado. Evidência visual local ignorada `docs/evidencias/iarete-gemini-contexto-pista.png`, no app. Contraste dos trechos Markdown, leitura por acessibilidade e horário local ajustados após essa conferência.


Primeira execução Gemini, commit `b05636a`: 48/57 aprovadas. As nove reprovações foram auditadas individualmente: três exigiam citar o modo sem que a pergunta o solicitasse; três proibiam uma consulta legítima a uma problemática ao perguntar o que analisar; três rejeitavam paráfrases corretas de ausência de histórico. O relatório original é preservado em `/tmp/onlife-benchmark-gemini-baseline.json`. Ajustadas as perguntas para pedir o modo explicitamente e limitar a etapa de interface aos controles, sem consulta a registros; ampliadas somente as paráfrases negativas aceitas, mantendo a proibição de vazamento e um teste que reprova resposta com a palavra privada. Os critérios de IDs, fatos, fontes, acesso e erros continuam obrigatórios. Relatórios novos incluem SHA-256 do manifesto de casos. A nova execução ainda precisa ser realizada e auditada.


Reexecução seletiva dos três cenários ajustados: 7/9 aprovadas. As duas reprovações restantes identificaram uma inconsistência real de linguagem: o modelo mostrou o identificador interno `explorer` em inglês, embora o cockpit use “Explorador”. Acrescentado `usage_mode_label`, calculado pelo backend a partir da conta e nunca aceito como autorização do cliente, e orientação para usar os nomes da interface. Prompt versionado como `onlife-iarete-4`; o critério continua exigindo o rótulo em português. A referência real completa deve ser refeita após essa correção.


### Referência real Gemini v4 — auditoria

Commit do backend `3b3fd8a4ae1c034b370d68ecf83ca6d0787d5549`, manifesto SHA-256 `eab762bc0569702d5093796ba2892159690092be7d0d3c0773d12187601f5a33`. Execução explícita com `--mode gemini --runs 3`: **57/57 casos-execuções aprovados, 72 etapas de conversa, 87 consultas MCP, 296.680 tokens informados e 417.553 ms no total**. Todas as etapas informaram modelo solicitado/retornado `gemini-3.8-flash`, prompt `onlife-iarete-4`. Relatório protegido local `/tmp/onlife-benchmark-gemini-reference-v4.json`, SHA-256 `4593403ec45e5743406923de19e2cf29f5d6991f167885185804afc2c3bc6e03`; ausência da credencial real no relatório conferida em memória, sem imprimir seu valor. Relatórios anteriores com reprovações foram preservados.

Auditadas respostas das 19 categorias, incluindo as continuações: distinção de telas e pistas, rascunho não salvo, leituras da própria pista, fontes realmente consultadas e divergentes, ausência de geografia, falha/sem resultados, conteúdo interativo não assistido, instrução maliciosa ignorada, acesso revogado e históricos separados. Não há compensação das falhas de segurança por outras categorias. Qualidades abertas continuam sujeitas à validação do usuário; o resultado do benchmark não homologa provedores externos nem capacidade do Jitsi.

Leitura pública adicional, fora da massa fixa: `NewSourceFetcher` de produção recuperou `https://learn.framevr.io/blog/iframes`, status `ok`, 3.331 caracteres, sem truncamento. Nenhum script foi executado. A prova usa DNS/TLS/transporte reais, sem substituição por fixture.

### Imagem da API e preservação do servidor

A imagem ativa passa a usar a base explícita `golang:1.23.6-bookworm`, nome próprio `onlife-api` e `.env` montado somente leitura em execução, excluído do contexto Docker. A chave Gemini não é passada ao build. Validado o build e um contêiner isolado: Go 1.23.6, `pdftotext` presente, `/api/status` e política de privacidade HTTP 200, MCP sem autenticação HTTP 401, nenhum `.env` embutido e zero chamadas ao provedor. Somente o banco sintético `onlife_validation_container` foi utilizado. O teste remove apenas seu próprio contêiner/arquivo temporário, mantendo o banco para auditoria.

Backup remoto concluído antes das alterações desta implantação: `/home/ubuntu/backups/onlife-integrations-20261021T053248Z`, 23 tabelas, dump SQL compactado, cópias de storage/frontend/Apache/TLS, ambientes anteriores e metadados do contêiner, todos em diretório privado. O manifesto contém hashes e tamanhos, sem imprimir conteúdos. O primeiro acesso por localhost foi recusado; o backup usou o gateway da rede Docker autorizado à conta da API. Nenhuma permissão do banco foi alterada. O código de preparação do Jitsi e o backup estão em `deploy/`; publicar os serviços ainda é uma etapa separada.


### Checkpoint — infraestrutura Jitsi e privacidade dos encontros

Publicado o stack separado em `/opt/onlife-jitsi`, distribuição oficial `stable-11248`: web, Prosody, Jicofo e JVB. Registro A `meet.spotec.app`, certificado válido (até 19/01/2027), proxy Apache separado e regra pública 10000/UDP; demais portas publicadas somente em loopback. Quatro contêineres em execução, sem reinícios observados na primeira inspeção. HTTPS dos endpoints existentes e do encontro retornou 200. Os snapshots anteriores e o backup privado foram preservados. Isso comprova publicação técnica, **não** áudio/vídeo, capacidade ou homologação de redes móveis. Organizadores reais, entrega das contas e chamada com três participantes em redes distintas continuam pendentes de definição/validação.

Links do servidor Jitsi são referências interativas: a Iareté não tenta recuperar presença, mensagens, áudio ou vídeo de reuniões. Testes de classificação e consulta MCP impedem qualquer transporte de rede nesse caso. A suíte completa da API passou em banco novo exclusivo, conforme `/tmp/onlife-api-full-suite-fresh.log`. O executor oferece `--fresh-database`, sem excluir bancos anteriores; SMTP e telemetria externos ficam desativados no ambiente de testes. Um teste antigo foi alinhado ao contrato já implementado de preservar embeds desconhecidos como dados opacos; vídeos ativos de origem não suportada continuam rejeitados.


### Implantação da API — preservação conferida

API implantada em uma imagem de release identificada no registro operacional privado, sem reinícios observados após a subida. Imagem construída antes da troca; somente o serviço da API foi recriado. Configuração protegida em execução, chave Gemini fora da imagem, `gemini-3.8-flash`, migrações destrutivas explicitamente desabilitadas. Nenhum dump foi restaurado, nenhum banco/volume removido.

`deploy/verify-preserved-data.py` registrou hashes por ID de todas as linhas e colunas existentes nas 23 tabelas antes da troca. Após inicialização/migrações, as mesmas linhas/valores foram reencontradas sem diferença, inclusive contas e históricos. Snapshot privado junto ao backup remoto; não contém conteúdos originais. A primeira tentativa de troca falhou ao procurar a tag padrão ainda não configurada e manteve o contêiner antigo ativo; a troca posterior usou a imagem construída e configuração persistente explícita.

Verificados após a troca: status da API, frontend, Geo API e Jitsi HTTP 200; MCP não autenticado HTTP 401. Teste Certbot `renew --cert-name meet.spotec.app --dry-run` concluído com código 0, sem modificar os certificados dos demais serviços. Health e métricas do JVB retornaram 200 na interface privada; Jicofo e JVB autenticados no Prosody conforme logs. O endpoint REST de saúde testado do Jicofo retornou 404 nesta versão e não foi usado como prova de saúde. Corrigido o identificador do idioma para `pt-BR`, presente nos arquivos oficiais desta distribuição. Chamada real com mídia e organizadores continua pendente; não inferir capacidade desses resultados.


### Consolidação dos testes e pesquisa de clientes

Todos os pacotes Go passaram em banco novo `onlife_validation_all_packages_20261021_064314_054532`, com chave Gemini vazia e SMTP/telemetria externos desativados (`/tmp/onlife-all-go-packages.log`). Metadados do modelo consultados também a partir do servidor com a configuração persistente: HTTP 200, `models/gemini-3.8-flash`, credencial somente no cabeçalho e sem impressão. Os snapshots de DNS/SG e IDs das alterações foram copiados para o backup privado remoto para permitir retorno apenas das mudanças Jitsi.

Overte: [arquitetura](https://docs.overte.org/en/latest/explore/get-started/architecture.html), [viagens entre mundos](https://docs.overte.org/en/latest/explore/travel.html) e [downloads oficiais](https://overte.org/downloads.html) confirmam o cliente Interface, descoberta de lugares dentro do aplicativo e dependência de permissão do domínio. Nenhum cliente/protocolo Overte/Vircadia registrado foi encontrado nos diretórios locais de aplicativos examinados. Não foi instalado um cliente nem escolhido um mundo por inferência; os contratos hifi/cartões não constituem abertura homologada.

Vircadia: [vircadia-web](https://github.com/vircadia/vircadia-web) declara descontinuação em favor de [Vircadia World](https://github.com/vircadia/vircadia-world). A [documentação do cliente Babylon.js](https://vircadia.com/vircadia-world/client/web_babylon_js/) exige configurar o endpoint do mundo; não comprova compatibilidade universal com ambientes desktop hifi. A tentativa de consultar `https://app.vircadia.com` expirou; isso não comprova encerramento do serviço. Sem versão/ambiente do grupo, não foi implantado um mundo nem anunciado suporte a iframe; referências desktop permanecem dependentes de validação específica. Metaverse continua oculto conforme o recorte explícito do plano.

Padlet: selecionado um exemplo público da [galeria oficial](https://padlet.com/site/templates?name=research-notes-general-v1), URL `https://padlet.com/gallerytemplates/social-media-research-notes-u527etuc56i78kiv`. Inserido, salvo e reaberto pela interface na pista sintética junto de Genially/FrameVR, com leitura incorporada e ampliação reais. Não foi publicada contribuição em mural de terceiros nem alterada sua privacidade. Contribuição exige mural apropriado/autorizado, cuja referência já foi solicitada. Evidência no app `docs/evidencias/padlet-real-ampliado.png` (ignorada). A tentativa de ajustar o viewport do navegador interno não alterou as dimensões efetivas (1280×720); não foi contada como prova de navegador móvel.


### Verificações adicionais — falha de provedor, Android e Jitsi

Teste real de Genially indisponível: um bloco temporário com ID inexistente retornou a página 404 do provedor dentro do iframe, sem bloquear os outros blocos nem remover o acesso externo. O bloco foi removido pela interface, salvo e a pista reaberta. GET autenticado posterior confirmou exatamente Padlet/FrameVR/Genially nessa ordem, título Genially editado e texto autoral das 18 árvores preservados. Evidência ignorada no app `docs/evidencias/genially-indisponivel-sem-bloquear-pista.png`.

APK `flutter build apk --debug --no-pub --target-platform android-arm64` concluído, 130 MiB, sem instalar em dispositivo. O toolchain alertou sobre NDK de plugins; a compilação existente completou. Não foram aceitos novos contratos de SDK. iOS não foi compilado neste host Linux.

Jitsi: visitante aguardou anfitrião; credenciais fictícias não cadastradas rejeitadas. Sem contas reais criadas, sem câmera/microfone ativos e sem conferências no JVB. Reinício controlado somente dos quatro contêineres comprovou retomada de saúde/configuração e endpoints existentes HTTP 200; não foi reiniciado o host. Digests, amostra ociosa de memória e observabilidade documentados em `deploy/jitsi/README.md`. Aceite com organizador real e três participantes/redes distintas permanece pendente.


## Revalidação de 21/10/2026

Resultado atual e limites de homologação consolidados em [Validação das integrações](validacao-integracoes-2026-10-21.md): Flutter 94/94, testes Go em banco novo, benchmark determinístico 57/57 e nova rodada Gemini real 57/57. Inclui reedição/persistência, uso dos provedores na interface e pendências externas explicitadas.
