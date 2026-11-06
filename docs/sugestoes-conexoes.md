# Sugestões de conexões — primeira entrega

A API reconhece referências explícitas aos nomes e nomes alternativos aprovados das pistas ativas. O serviço interno usa spaCy 3.8.7, o modelo português pt_core_news_sm 3.8.0 e PhraseMatcher. Não há busca semântica, chamadas a modelos externos nem criação automática de vínculos.

## Fluxo

No cadastro e na edição, **Também conhecido como** aceita até 20 nomes próprios da entidade, separados por ponto e vírgula (2–120 caracteres). A ausência de `alias_names` no POST/PUT é compatível com clientes antigos; omissão no PUT preserva os nomes, enquanto `[]` limpa os nomes aprovados. Nunca usar entidades meramente mencionadas como nomes da própria pista.

Ao criar/editar, ou ao consultar uma pista cujo conteúdo/catálogo/rede mudou, a API agenda análise na fila existente. A versão considera o nome, descrição, nomes aprovados e conexões da rede. Resultados atrasados são descartados e reagendados. A gravação da pista não depende da disponibilidade do extrator. Uma falha vira estado de análise indisponível, sem tentativas automáticas a cada cinco segundos; há atualização explícita e nova tentativa após intervalo.

A tela da pista exibe até cinco sugestões com motivo e trecho de origem. Referências diretas vêm primeiro. Vizinhos da rede só entram quando também há pelo menos dois termos textuais relevantes em comum além dos nomes da instituição/referência. Expressões genéricas de currículos/publicações e termos presentes em uma parcela ampla do catálogo são excluídos; não representam semelhança semântica nem comprovação de participação. Pistas inativas, excluídas, a própria pista e pares já conectados são omitidos. Nomes ambíguos podem produzir mais de um destino, sempre sujeito a revisão.

**Revisar conexão** abre o formulário existente com origem, destino e evidência. O tipo e o sentido continuam sujeitos à confirmação humana e às permissões existentes. Dispensar guarda a decisão na versão atual do conteúdo/catálogo. Alterações posteriores podem produzir novas sugestões. Nenhuma porcentagem é apresentada como probabilidade.

Autodefinições explícitas, como `Nome completo (Sigla)` ou `Nome completo, também conhecido como Nome curto`, podem gerar nomes pendentes para a própria entidade. Apenas aprovação os inclui no dicionário. O NER também apresenta até dez nomes ainda não resolvidos para conferência; sua classificação pode errar e não cadastra entidades automaticamente. O extrator usa apenas nome e descrição armazenados, sem ler anexos, vídeos ou páginas externas.

## Infraestrutura

- `discovery/Dockerfile` instala o serviço Python; `discovery/compose.yml` documenta recursos e isolamento.
- Integrar o serviço à rede Docker da API; configurar `CLUE_DISCOVERY_URL=http://onlife-clue-discovery:8091` na API OnLIFE. A rede pode ser definida externamente quando o serviço for gerenciado separadamente.
- Um processo, análise serializada, no máximo uma CPU e 1 GiB de memória/swap total, usuário sem privilégios, sistema de arquivos somente leitura e nenhum acesso ao banco ou segredos da API. A porta 8091 não é publicada no host.
- `/health` interno informa modelo e componentes. Não há log de corpo das requisições. O texto permanece na infraestrutura da plataforma.
- O limite do piloto é 10 mil pistas ativas e 200 mil caracteres por análise. Não há truncamento silencioso: o excesso é reportado como análise indisponível; o cadastro continua funcional. Textos são particionados para o NER; o reconhecimento de referências cobre o texto todo. O NER interrompe após coletar dez entidades não resolvidas.
- Tabelas aditivas: `clue_aliases`, `clue_discovery_states`, `clue_suggestions`. O histórico das pistas e as conexões anteriores são preservados.

## Endpoints autenticados

- `GET /living_clues/{id}/connection_suggestions`: estado, até cinco sugestões, nomes aprovados/pendentes e referências não resolvidas.
- `POST /living_clues/{id}/connection_suggestions/refresh`: reagenda sem criar vínculos.
- `POST /living_clues/{id}/connection_suggestions/{suggestion}/dismiss`: dispensa sugestão da pista.
- `PUT /living_clues/{id}/aliases/{alias}` com `status=approved|rejected`: revisão de nome pendente; registra edição no histórico.

Escritas seguem a restrição do modo Inventor. As consultas respeitam o acesso da pessoa autenticada. As sugestões são compartilhadas na pista, incluindo a dispensa, nesta primeira versão colaborativa.

## Validação

Testes Python cobrem acentos, limites de palavras, ambiguidade, autodefinição e a distinção entre menção e nome alternativo. Testes Go com MariaDB sintético verificam persistência, preservação para clientes antigos, estados, dispensa, filtros e permissões. A integração real Go → spaCy usa `CLUE_DISCOVERY_TEST_URL` somente em ambiente de teste. Testes Flutter cobrem sugestões, revisão de nomes, Explorador e limites do campo.
