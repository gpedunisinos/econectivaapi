# Integração opcional CDIG → plataforma

## Ativação independente por base

A integração fica **desativada por padrão**, inclusive quando existe configuração
parcial. Só é habilitada com `CDIG_SYNC_ENABLED=true`, origem HTTPS, ambiente de
origem, usuário local responsável e credencial completa. Login, cadastro, pistas,
mapa e outras funções não consultam a CDIG e não dependem da sua disponibilidade.
A rotina não é ativada por nome de domínio ou por compartilhar a mesma imagem Go.

Exemplo para a instância que receberá conteúdos:

```dotenv
CDIG_SYNC_ENABLED=true
CDIG_BASE_URL=https://cidig-app-9gusl.ondigitalocean.app
CDIG_SOURCE_KEY=cdig-staging
CDIG_IMPORT_USER_ID=1
CDIG_API_TOKEN=
CDIG_EMAIL=
CDIG_PASSWORD=
CDIG_SYNC_INTERVAL_HOURS=0
```

Informe **um token `content:read`** em `CDIG_API_TOKEN` **ou** preencha e-mail e senha
de uma conta administradora ativa/aprovada da CDIG. Valores reais ficam somente no
`.env` protegido do backend. O token tem prioridade sobre e-mail/senha. No modo
credenciais, cada execução emite um token de um dia e o revoga após a leitura,
inclusive em falhas. Falha ao revogar impede a importação daquela execução. Um
token fornecido pelo operador não é revogado pelo sincronizador; renove-o antes
do vencimento. Não é login unificado e nenhuma conta da CDIG é importada.

`CDIG_IMPORT_USER_ID` é uma conta já existente na **base de destino**, usada na
história de criação/edição. Não é o ID de usuário da CDIG. Confirme o valor na base.

Para a instância sem integração, mantenha:

```dotenv
CDIG_SYNC_ENABLED=false
```

As demais variáveis podem estar ausentes. Não são criadas tabelas CDIG nem chamadas
externas quando a integração está desligada. Configuração inválida desativa apenas
o recurso; não impede a API de iniciar. Reinicie a instância após mudar o `.env`.

## Simular antes de importar

No mesmo ambiente/configuração do backend:

```sh
go run ./cmd/cdig-sync
```

O comando padrão é uma **simulação**: consulta o catálogo completo e o destino,
produz JSON de ações, mas não cria tabelas, categorias, pistas, conexões ou vínculos.
Pode emitir/revogar uma credencial temporária na origem quando usado o modo
usuário/senha. Não chama o inicializador geral do banco nem redefine contas.

Após conferir o relatório e ter backup do destino:

```sh
go run ./cmd/cdig-sync --apply
```

Ou compile uma vez `go build -o cdig-sync ./cmd/cdig-sync` e execute o binário
junto ao `.env` protegido. O MySQL do destino deve estar preparado com as tabelas
normais da plataforma. A primeira aplicação cria automaticamente apenas as duas
tabelas de controle CDIG. Nenhum acesso administrativo ao servidor CDIG é exigido.
O comando verifica `EXPECTED_DATABASE_NAME`, quando configurado.

O relatório contém tipos/IDs e ações `created`, `linked`, `updated`, `unchanged`,
`suppressed`, `conflict`, `restored` e `unavailable`. IDs provisórios de novas pistas na
simulação não são IDs reservados no banco. Se o comando falhar, as ações indicam
somente planejamento/tentativas: a transação de conteúdo é revertida. O campo
`committed` só é verdadeiro após uma aplicação concluída.

## Agendamento e desligamento

`CDIG_SYNC_INTERVAL_HOURS=0` (ou ausente) mantém execução **somente manual**.
Valores de 1 a 168 habilitam a rotina periódica dentro da API. A primeira execução
ocorre após o primeiro intervalo, sem carga automática na inicialização.
O mesmo sincronizador é utilizado pelo comando e pelo agendador. Um lock MySQL
por origem evita escritas concorrentes entre processos/containers. Não há rota
pública para disparar importações. Falha de sincronização não derruba a API.
Desative a flag e reinicie a instância para parar futuras execuções; pistas já
importadas e seus vínculos permanecem na base.

## Dados, identidades e preservação

- Varredura completa paginada de recursos publicados, vídeos ativos, séries ativas
  e projetos aprovados com coordenadas. Verifica erros GraphQL, totais, IDs repetidos,
  páginas vazias/incompletas e relações de episódios antes de gravar.
- Identidade única por URL de origem + `CDIG_SOURCE_KEY` + tipo + ID. Não reutilize
  chaves de ambiente e não aponte o staging para a identidade da produção. Trocar
  a origem cria outra identidade e requer uma migração/reconciliação revisada.
- Repetir a execução não duplica itens. Um vídeo compartilhado por várias séries
  continua sendo uma pista. Séries usam `Coleção temática`, `Coleção contém item`
  e `colecao_ordem_itens_v1`, as convenções já utilizadas no app.
- Guarda título/descrição/ordem, autoria, direitos, metadados e origem. Remove HTML
  de descrições antes de gerar texto/Delta. Usa capas de séries/YouTube quando
  disponíveis e preserva as capas já existentes; não inventa capas para recursos. Títulos acima de 255 caracteres são abreviados
  para o limite do destino, com o original integral preservado nos metadados.
- Importações legadas são associadas somente quando o identificador **e a URL do
  recurso** correspondem. IDs iguais com URLs divergentes ou associações ambíguas
  vão para revisão. A primeira associação preserva apresentação e capa existentes.
  Não há correspondência por título ou suposição de igualdade entre ambientes.
- Pistas excluídas localmente geram supressão persistente ao serem reencontradas;
  não são recriadas. Isso inclui a pista retirada anteriormente, desde que seu
  registro excluído e a identidade/URL confirmada ainda existam na base. Se ela
  tiver sido apagada fisicamente, restaure a evidência de identidade/supressão
  antes de aplicar a carga, para não perder essa decisão editorial.
- Atualizações verificam um fingerprint dos campos geridos. Edições locais nesses
  campos viram conflitos, sem sobrescrita; comentários, arquivos, metadados locais,
  conexões e outros registros não são substituídos. Conflitos precisam de revisão,
  não são automaticamente resolvidos em execuções futuras.
- Mudança em episódios altera a ordem da coleção e apenas as conexões criadas pelo
  importador. Membros adicionados localmente permanecem. Conteúdos individuais não
  são excluídos ao sair da série. Falta de um episódio resolvido aborta a transação.
- Ausência em uma varredura **concluída** marca a origem como indisponível, sem
  apagar pistas, interações ou vínculos. Falha de rede/página não significa remoção.
  A política inicial preserva a referência; não a oculta nem a exclui automaticamente.

## App e configuração pública

`GET /api/integrations/cdig` retorna apenas `enabled` e, quando habilitado, a URL
pública da CDIG. Não expõe credencial, conta, motivo técnico ou dados do servidor.
O app mostra a orientação sobre cadastro somente nessa condição; falha/404 ou
integração desligada não bloqueiam a criação local. Pistas importadas mostram a
fonte, autoria/direitos e eventual indisponibilidade sem apresentar JSON interno.
O app fala somente com a API da própria plataforma; os segredos ficam no backend.

## Validação

```sh
go test ./integrations/cdig -count=1
python3 scripts/run-isolated-tests.py --fresh-database \
  --database onlife_validation_cdig --package ./integrations/cdig \
  --run TestSyncIsolatedMySQL --log /tmp/cdig-sync-isolated.log
```

O segundo comando usa somente um banco MySQL/MariaDB local sintético com prefixo
`onlife_validation_`. Testes cobrem simulação sem escritas, repetição, exclusão
manual, edição local, conexões compartilhadas, ausência de origem, erro remoto e
lock concorrente. Testes unitários de HTTP/configuração não precisam de banco.
