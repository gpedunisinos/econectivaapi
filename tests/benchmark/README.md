# Benchmark educacional da Iareté

Implementa a matriz de 19 cenários do INT-09, revisão 2592 do [plano aprovado](https://wiki.spotec.app/index.php/Plataforma_Educa%C3%A7%C3%A3o_Onlife_-_Planejamento_de_integra%C3%A7%C3%B5es_e_intelig%C3%AAncia_artificial). Todos os nomes, usuários, leituras, pistas, relações e documentos são fictícios. Não usa casos, prompts, dados de atendimento ou históricos SUSI.

## Execução

Listar cenários sem banco ou chamada ao provedor:

```sh
go run ./tests/benchmark/cmd --list
```

Testar o executor e as expectativas, sem banco e sem LLM:

```sh
go test ./tests/benchmark/...
```

Executar todos os cenários determinísticos, três vezes, contra o banco local sintético:

```sh
python3 scripts/run-ai-benchmark.py --mode deterministic --runs 3 --output /tmp/onlife-benchmark-deterministic.json
```

Executar explicitamente o benchmark real (**consome quota**) com Gemini 3.8 Flash:

```sh
python3 scripts/run-ai-benchmark.py --mode gemini --runs 3 --output /tmp/onlife-benchmark-gemini.json
```

Selecionar IDs ou tags:

```sh
python3 scripts/run-ai-benchmark.py --mode deterministic --case iarete-troca-de-pista,pista-chat-fonte-externa --runs 3
python3 scripts/run-ai-benchmark.py --mode gemini --tag seguranca --runs 3
```

O script exige o contêiner local `rieonlife-validation-db`, publicado em `127.0.0.1:33419`; cria somente `onlife_validation_benchmark` quando ausente, nunca o remove. A credencial MariaDB é lida em memória do contêiner. No modo real, somente a chave autorizada `GEMINI_API_KEY` é lida de `/home/odair/projects/susi_ai/.env` e passada como `GENAI_API_KEY` ao processo filho. Nenhum valor é impresso, versionado ou persistido no relatório. A chave fica no cabeçalho enviado pelo adaptador da API, sem query string. Não executar o CLI contra outro banco. O modo real exige `ONLIFE_BENCHMARK_ALLOW_PROVIDER=1`, aplicado apenas quando o argumento explícito `--mode gemini` é utilizado.

## O que é real e o que é simulado

Nos dois modos, o executor usa os prompts de produção `BuildAIMessageSystemInstruction`/`BuildLivingClueSystemInstruction`, a montagem de contexto atual/rascunho, o adaptador `GeminiChatService`, declarações de ferramentas, orçamento, assinaturas de partes e o ciclo de consultas. As consultas passam pelo cliente e servidor do SDK oficial MCP e pelo `DatabaseBackend` com as políticas/repositórios reais, sobre registros sintéticos. Os históricos são mantidos apenas em memória por execução, usuário e identidade de chat; nenhuma mensagem é escrita nas contas existentes.

Fontes usam URLs fictícias públicas `onlife-fixtures.example` e transporte injetado exclusivamente nas fixtures. A mesma extração HTML, cache, classificação de conteúdo e validação de URLs de produção é exercitada. Há documento íntegro, conflito (18 árvores na pista / 27 no relatório), falha 503, incorporação sem transcrição e instrução maliciosa. Nenhum documento de fixture é buscado na internet. A segurança DNS/rede de produção é testada separadamente em `services/clue_sources_test.go`.

No modo **determinístico**, somente o transporte Gemini é substituído por respostas/funções declaradas em `cases.json`. Ele valida integração, rastros e o próprio avaliador; não mede a capacidade do modelo. O modelo retornado no relatório tem sufixo `-simulated`, e os tokens são zero. No modo **gemini**, o transporte Gemini é real e somente essa chamada pode sair para o provedor autorizado. Não há fallback para outro modelo.

O cenário de acesso representa revogação entre a autenticação e a consulta: a identidade recebida pelo adaptador deixou de existir no banco, e o backend MCP precisa recusá-la em cada consulta. O teste HTTP `TestIareteMCPAuthenticatedQueriesAndPinnedSource` cobre também autenticação e política da rota, sem depender do modelo.

## Aprovação e relatório

Todas as expectativas configuradas precisam passar: fatos/ausência de contradições, origens e referências, ferramenta, argumentos/IDs, dados realmente presentes nos resultados, proibições e limites. Uma palavra correta não compensa ID incorreto, resultado ausente ou fato proibido. Erros de quota/modelo/provedor, resposta vazia/inválida ou expectativa não atendida reprovam e produzem saída diferente de zero. Somente o erro explícito de orçamento do orquestrador pode ser esperado no cenário de limites; um erro do provedor jamais conta como aprovação.

O JSON contém modo, commit, data UTC, versão das fixtures/prompts/ferramentas, parâmetros, modelo solicitado/retornado, cada execução e etapa, entradas/contexto sintéticos, respostas, chamadas e resultados, duração e tokens. As contagens por tag não ocultam reprovações individuais. Arquivos são gravados com permissão `0600`; mantenha relatórios em diretório temporário ou evidências ignoradas, sem versionar caches nem credenciais.

Assertivas verificáveis não equivalem a uma avaliação semântica universal: clareza, acolhimento e postura não diretiva exigem revisão humana das respostas, além dos testes automáticos. Não usar outro LLM como único juiz. A primeira referência real só existe depois de executar e auditar o relatório Gemini; testes determinísticos e provas de conectividade não a substituem.

## Ampliação USP App — plano de melhorias

A matriz agora possui 22 cenários (19 anteriores e 3 do piloto USP), executados três vezes em cada modo. A versão `onlife-usp-synthetic-2` acrescenta planta, sensor, ambiente e pesquisador com categorias próprias, relações dirigidas e duas leituras inteiramente fictícias. Os IDs reservados 910107–910110 não são IDs de produção.

| Critério USP | Cenário |
| --- | --- |
| Oito indicadores, zero válido, distinção entre contexto territorial e planta, ausência de procedência no contexto entregue ao modelo | `usp-contexto-oito-indicadores` |
| Instante versus recebimento, leitura antiga, simulação, ausência de unidade no legado e limite de diagnóstico | `usp-leitura-antiga-sem-diagnostico` |
| Direção pesquisador → planta → ambiente e distinção do alvo medido | `usp-relacoes-direcao` |
| Ausência de geografia, troca de pista, fontes conflitantes/indisponíveis e isolamento | Cenários anteriores preservados |

A referência armazenada no snapshot não é repassada pelo contrato MCP atual como procedência histórica. O novo cenário exige reconhecer essa limitação; não atribui automaticamente MapBiomas ou IBGE aos dados recebidos pelo modelo. A interface apresenta as fontes do snapshot separadamente.

Responsável proposto para consolidar a revisão conceitual: Odair, com os pesquisadores do projeto. As respostas reais e os casos devem ser apresentados à revisão humana; execução automatizada não equivale à aprovação científica.
