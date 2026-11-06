# Ambiente local com dados do servidor — 20/10/2026

O deploy em `bitbucket-pipelines.yml` aponta para `ubuntu@54.94.142.29`, diretório `/var/www/unisinos_usp_api`. Foi feito um dump consistente do MySQL 8 com `--single-transaction`, incluindo rotinas, eventos e triggers. Todas as tabelas da origem são InnoDB. Nenhum deploy ou alteração no banco remoto foi executado.

## Backup e restauração

Backup local privado: `/home/odair/backups/rieonlife/2026-10-20/mysql.sql.gz`.

SHA-256, idêntico no servidor e localmente:

```
c56ab09b21def6c3363afbe02a8e44433e130f55054163945884f41d8ed325c8
```

Restaurado no container MySQL 8 `rieonlife-server-copy-db`, volume persistente `rieonlife-server-copy-mysql`, porta `127.0.0.1:33420`, base `rieonlife_server_copy`. O banco de demonstração anterior, em MariaDB/33419, foi preservado. Contagens brutas da cópia: 9 categorias, 43 pistas, 28 nós, 29 usuários; incluem registros excluídos logicamente. A API retorna 27 pistas e 22 nós ativos.

O `.env` deste checkout agora é um arquivo local independente (antes era um link para outro checkout). Configura esse MySQL, um segredo JWT local e o Geo API. O arquivo anterior e o destino do link foram preservados no diretório privado do backup. Credenciais ficam fora do Git. Os arquivos remotos de `storage/chat_templates/` também foram copiados para manter os anexos disponíveis.

## Iniciar

No checkout `/home/odair/projects/unisinos_usp_api`:

```bash
./scripts/start-local-server-copy.sh
```

O comando inicia o MySQL, estabelece o túnel SSH `127.0.0.1:18000 → servidor:8000` e serve a API em `127.0.0.1:8080`. Precisa da chave SSH já configurada neste computador. Mantenha o terminal aberto; Ctrl+C encerra a API e o túnel. O MySQL preserva os dados no volume.

A inicialização aplica as migrações locais e prepara a conta administrativa local como no bootstrap existente. O executável de prévia não inicia o worker, para não executar a fila de jobs copiada do servidor. Mapas, catálogo, tiles e consultas diretas ao Geo API funcionam; jobs novos de contexto não são processados automaticamente nesse modo. Integrações de envio/IA não receberam as credenciais operacionais do servidor.

A versão web compilada aponta para esta API. Para servir o build, no checkout Flutter:

```bash
python3 -m http.server 8767 --bind 127.0.0.1 --directory build/web
```

Abra `http://localhost:8767/#/login`. Ao trocar a base, saia da sessão anterior e entre novamente, pois o segredo JWT é local. Contas sem modo passam por `/choose_mode`; os dois modos seguem para `/home`.

## Validação

- Importação integral do SQL e migrações locais concluídas.
- Login local e consultas autenticadas de categorias, pistas, nós e camadas: HTTP 200.
- Geo API remoto autenticado: 8 camadas disponíveis.
- `/context` para São Leopoldo: HTTP 200, Mata Atlântica, seis fontes.
- Build web atualizado e testes Flutter de rotas/experiência executados.

Não executar a suíte Go de integração sobre esta cópia: ela deve usar uma base descartável própria. A cópia serve para testes manuais com dados reais.
