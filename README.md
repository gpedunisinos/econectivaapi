# Educação OnLIFE — API

API da plataforma **Educação OnLIFE**, responsável pelos registros de pistas e suas relações, usuários, problemáticas, agentes de inteligência artificial, arquivos e interações entre participantes.

Este repositório contém o backend em **Go**, com **Gin** para as rotas HTTP e **GORM** para persistência em MySQL. O aplicativo Flutter é mantido em [educacaoonlifeapp](https://github.com/gpedunisinos/educacaoonlifeapp).

**[Acessar a plataforma](https://educacaoonlife.gpedu.com.br)** · **[Aplicativo](https://github.com/gpedunisinos/educacaoonlifeapp)** · **[Financiamento](#financiamento)**

## Funcionalidades

- Autenticação, usuários, perfis e permissões de participação.
- Pistas, categorias, comentários, vínculos e classificação de práticas e metodologias inventivas.
- Problemáticas, nodos e suas conexões.
- Agentes de IA e conversas, incluindo transcrição e resposta por áudio quando configuradas.
- Arquivos locais ou armazenamento privado no Amazon S3.
- Leituras de sensores vinculados às pistas.
- Presença e conversas entre participantes por WebSocket.
- Acesso integrado a reuniões, consultas geográficas e outros serviços configuráveis.

## Pré-requisitos

- Go **1.23 ou superior**, conforme `go.mod`, ou Docker com Compose.
- MySQL acessível pelo ambiente de execução.
- Configuração dos serviços externos utilizados: provedor de IA, e-mail, armazenamento, autenticação e API geográfica.

## Instalação

```bash
git clone git@github.com:gpedunisinos/econectivaapi.git
cd econectivaapi
cp .env.example .env
go mod download
```

Edite o arquivo `.env` antes de iniciar. Informe os dados de conexão do banco e gere segredos próprios para o ambiente. O arquivo `.env` é local e não deve ser versionado.

O nome interno do módulo Go foi preservado para manter os imports existentes compatíveis. O código pode ser compilado a partir deste repositório sem renomear os pacotes.

## Configuração

O arquivo [.env.example](.env.example) descreve as variáveis disponíveis. Os principais grupos são:

| Grupo | Variáveis principais |
| --- | --- |
| Aplicação | `APP_ENV`, `APP_URL`, `APP_PORT`, `PUBLIC_APP_URL` |
| Banco | `DB_DRIVER`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` |
| Autenticação | `API_SECRET`, `TOKEN_HOUR_LIFESPAN`, configurações de Google e Apple |
| Isolamento de ambientes | `PLATFORM_ID`, `EXPECTED_DATABASE_NAME`, `API_ALLOWED_ORIGINS`, `PRESENCE_ALLOWED_ORIGINS` |
| Arquivos | `STORAGE_DRIVER`, `AWS_REGION`, `S3_BUCKET`, `S3_ROLE_ARN` |
| Inteligência artificial | `GENAI_API_KEY`, `GEMINI_MODEL`, `OPENAI_API_KEY` e opções de voz |
| Integrações | `ENVIRONMENTAL_CONTEXT_API_URL`, configurações de e-mail e de Jitsi |
| Logs | `UNISINOS_USP_API_LOG_DIR`, `UNISINOS_USP_API_LOG_FILE_PREFIX`, `API_LOG_ACCESS_TOKEN` |

O acesso remoto aos logs fica desabilitado quando `API_LOG_ACCESS_TOKEN` não está definido. Configure um valor próprio somente nos ambientes em que esse acesso for necessário.

## Execução

### Com Go

```bash
go run .
```

### Com Docker Compose

Configure também `APP_PORT` no `.env` para a porta HTTP publicada no host. O MySQL deve estar acessível pelo `DB_HOST` informado; o Compose deste repositório sobe o serviço da API.

```bash
docker compose build unisinos_usp_api_go
docker compose up -d unisinos_usp_api_go
```

Para compilar um binário:

```bash
go build -o build/econectivaapi .
```

## Estrutura

```text
config/          Inicialização e configuração do banco
http/            Controllers, middlewares, requests e responses
models/          Modelos de dados
repository/      Acesso aos dados
routes/          Rotas HTTP e WebSocket
services/        Serviços da aplicação e integrações
storagebackend/  Armazenamento de arquivos
notifications/   Notificações
jobs/            Processamento de tarefas
docs/            Especificação da API e documentação técnica
deploy/          Configurações e ferramentas de implantação
```

## Documentação

- [Especificação OpenAPI/Swagger](docs/swagger.yaml)
- [Arquivos e armazenamento](docs/arquivos-s3.md)
- [Configuração de reuniões](deploy/jitsi/README.md)
- [Implantação e isolamento de plataformas](deploy/platform-split/README.md)
- [Publicação inicial deste repositório](docs/publicacao-github.md)

## Contribuições

Utilize as [issues](https://github.com/gpedunisinos/econectivaapi/issues) para sugestões e problemas. Envie alterações por pull request, descrevendo seu objetivo e as configurações ou contratos de API afetados.

Preserve a grafia **Educação OnLIFE**, **IAraeté**, **Pistas em fluxo** e **COnversAÇÕES** nos textos públicos.

## Financiamento

A plataforma Educação OnLIFE conta com financiamento da **Fundação de Amparo à Pesquisa do Estado do Rio Grande do Sul (FAPERGS)** e da **Fundação de Amparo à Pesquisa do Estado de São Paulo (FAPESP)**.

<p>
  <a href="https://fapergs.rs.gov.br/"><img src="docs/assets/financiamento/fapergs.jpg" alt="FAPERGS — Fundação de Amparo à Pesquisa do Estado do Rio Grande do Sul" width="280"></a>
  &nbsp;&nbsp;
  <a href="https://fapesp.br/"><img src="docs/assets/financiamento/fapesp.svg" alt="FAPESP — Fundação de Amparo à Pesquisa do Estado de São Paulo" width="260"></a>
</p>

Logos de origem institucional: [FAPERGS](https://fapergs.rs.gov.br/identidade-visual) e [FAPESP](https://fapesp.br/). O logo branco original da FAPESP é apresentado proporcionalmente sobre fundo azul para manter a legibilidade.

## Licença

Consulte a [licença Apache 2.0](LICENSE). Os nomes e logos institucionais pertencem às respectivas instituições e não são abrangidos pela licença do código.
