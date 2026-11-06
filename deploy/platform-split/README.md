# Isolamento Educação OnLIFE / Atopos

| Plataforma | Front | API | MySQL | Container | S3 |
|---|---|---|---|---|---|
| OnLIFE | educacaoonlife.gpedu.com.br | api.uniusp.gpedu.com.br | unisinos_usp_api | UnisinosUspAPI:8081 | educacao-onlife-arquivos-247433253659-sa-east-1 |
| Atopos | plataforma.atopos.spotec.app | api.atopos.spotec.app | atopos_api | AtoposAPI:8084 (loopback) | atopos-arquivos-247433253659-sa-east-1 |

O código da API é único, com instâncias, `.env`, usuários SQL e armazenamento independentes. `EXPECTED_DATABASE_NAME` bloqueia inicialização contra a base errada. Não copiar `.env` entre instalações nem usar credenciais reais em scripts/relatórios públicos.

## Preparação e corte

1. `provision-aws.py` usa a identidade AWS local para DNS, bucket privado com versionamento e role assumida pela Atopos. Nenhuma chave permanente é criada. O role da Atopos só acessa seu bucket.
2. No host, `manage.py prepare` salva configuração e binário anteriores em `/home/ubuntu/backups/platform-split`, cria banco/conta SQL exclusivos e a configuração protegida da Atopos. O helper `tools/platform-env` deve ser compilado como `env-reader` ao lado de `manage.py`, para ler `.env` com o mesmo parser da API (incluindo comentários e escapes). Precisa do diretório `/var/www/atopos_api` previamente criado para ubuntu. A conta administrativa MySQL usa `/etc/mysql/debian.cnf` via sudo.
3. `manage.py clone` gera dump consistente, restaura, compara contagens/checksums e copia os arquivos locais necessários. A primeira cópia é de validação. O candidato pode receber testes, pois será substituído antes de abrir o front.
4. No corte, bloquear escritas no Apache original, parar os dois workers, fazer `clone --refresh`, sincronizar S3, revalidar cópia, iniciar as APIs isoladas e publicar somente o front Atopos apontando para sua API. Remover o bloqueio de escrita. `--refresh` é recusado após criar `activated.json`.
5. Usuários e conteúdo histórico são preservados; tokens de sessão, recuperação e notificação são limpos somente na Atopos. Logins são independentes. Conexões dos sensores existentes continuam na OnLIFE; novos dispositivos Atopos devem usar seu endpoint.

## Integrações e limites

A Geo API/PostGIS e suas camadas públicas continuam únicos. Cada API guarda seus próprios contextos por pista. `ENVIRONMENTAL_CONTEXT_API_*` é configuração do servidor, nunca do front.

A busca Nominatim usa `/var/lib/onlife-place-search`, bind comum às duas APIs: lock interprocesso, intervalo mínimo global de 1,1 s e cache compartilhado limitado a 512 resultados por 24 h. Não expor esse diretório em HTTP.

`PLATFORM_ID` prefixa as salas Jitsi, inclusive convites já prefixados de forma idempotente. O serviço Jitsi e seu emissor JWT podem ser compartilhados; os nomes de sala das plataformas não colidem. CORS e a presença aceitam somente as origens configuradas de cada plataforma. JWTs da Atopos usam segredo diferente. Os tokens antigos da OnLIFE continuam válidos somente na OnLIFE.

Arquivos S3 mantêm suas chaves, mas o registro da Atopos aponta para o bucket copiado. Referências a arquivos locais da API antiga são reescritas na cópia. Conteúdo externo continua externo; não é transformado em uma cópia privada. O bucket mantém objetos privados e downloads temporários.

## Publicações posteriores

Na pasta `/var/www/unisinos_usp_api`, após commit e testes:

```
python3 deploy/platform-split/deploy-api.py --platform onlife --build
python3 deploy/platform-split/deploy-api.py --platform atopos --image onlife-api:release-<revisao-validada>
```

O pipeline padrão publica apenas OnLIFE. A publicação Atopos é explícita e não reinicia a outra API. O front Atopos pertence ao repositório `usp_app`; o cockpit OnLIFE ao `unisinos_usp_app`. Scripts de deploy do front não devem mudar o document root da outra plataforma.

## Backup e reversão

Os backups protegidos contêm dump, arquivos, configuração e binário originais, além das contagens e checksums. Antes de surgirem escritas independentes, é possível repor o front anterior e a API original. Depois do corte, não apontar novamente a Atopos para o banco OnLIFE: primeiro preservar e reconciliar as novas alterações. Os backups desta migração não são releases antigos de front descartáveis.

## Validação em produção

`smoke.py` usa contas e registros descartáveis com nomes únicos para validar login, rejeição de JWT cruzado, CORS, catálogos, pistas, sensores, nodos, agentes, arquivos S3, Geo API e reuniões. O helper `tools/platform-presence-check` verifica WebSocket real, participantes opacos, texto/emojis e ausência de conversas cruzadas. Remove somente os próprios registros e guarda o relatório protegido em `/home/ubuntu/backups/platform-split/smoke.json`.

A suíte completa de API deve ser executada antes do corte contra a cópia descartável, com `S3_ROLE_ARN` vazio e `STORAGE_DRIVER=local`; seus testes de S3 usam um provedor simulado. O armazenamento de produção é validado separadamente pelo helper `tools/storage-check`, com exclusão de seu objeto sintético ao terminar.

## Corte concluído — 04/11/2026

- API publicada nas duas instâncias: `8b1f338`; front Atopos: `9177254`. O front OnLIFE foi preservado em `515b034`.
- Cópia final: 32 tabelas e 1.570 registros por banco. Contagens e checksums foram conferidos antes dos ajustes de sessão/armazenamento; após os testes, as contagens originais foram reconferidas.
- 157 testes de integração da API aprovados na cópia descartável e 107 testes Flutter aprovados no export limpo do front Atopos. Alterações locais não commitadas no editor Atopos não foram publicadas.
- 30 verificações reais de produção aprovadas, mais WebSocket/presença/chat com texto e emojis. JWT, origem e conversas cruzadas foram rejeitados. As contas e registros descartáveis foram removidos.
- IA real validada nas duas APIs: resposta de texto, voz e transcrição de áudio reamostrado para o formato do gravador (PCM16 mono 16 kHz). Não foram usados dados pessoais nesse teste.
- S3 privado validado com upload, download assinado, preservação do anexo histórico e acesso público negado. A cópia final dos objetos foi verificada por tamanho e ETag.
- Usuário MySQL Atopos impedido de ler o banco OnLIFE. O segredo JWT original da OnLIFE foi preservado; a Atopos usa outro segredo e requer novo login. A Geo API e a quota de geocodificação permanecem compartilhadas.
- Relatórios protegidos: `/home/ubuntu/backups/platform-split/{latest-clone,smoke,ai-smoke,final-audit,activated}.json`. O log da suíte completa fica no mesmo diretório. Não publicar os dumps/configurações junto de evidências.

O login Google/Apple foi coberto pelos testes da API com provedores simulados; não foi iniciada uma nova sessão OAuth pessoal no navegador durante o corte. As origens dos fronts e os IDs dos clientes OAuth existentes foram mantidos. Os testes reais de login usaram contas descartáveis com senha.
