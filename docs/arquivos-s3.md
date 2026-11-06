# Arquivos e evidências no S3

Os novos anexos de pistas são enviados pelo app à API em `multipart/form-data`, campo `file`. A API valida o formato, o limite de 25 MiB e a pista de destino, gera uma chave independente do caminho recebido e grava o objeto privado. O nome original fica nos metadados e no download. JSON `base_64` permanece compatível com clientes antigos. Links HTTP/HTTPS continuam disponíveis sem serem copiados para o bucket.

## Infraestrutura

- Conta AWS: `247433253659`; região São Paulo `sa-east-1`.
- Bucket: `educacao-onlife-arquivos-247433253659-sa-east-1`.
- Bloqueio público completo, propriedade `BucketOwnerEnforced`, criptografia SSE-S3 AES256, HTTPS obrigatório, versionamento habilitado.
- Role da instância EC2: `AmazonSSMRoleForInstancesQuickSetup`; política inline `OnLIFEPrivateAttachments`, limitada a Put/Get/Delete em `files/*` deste bucket. Não há credenciais AWS no app ou no repositório.
- Provisionamento idempotente: `python3 deploy/s3/provision.py`, pela identidade autorizada da AWS CLI.
- Não há CORS de upload: o app envia à API, e os downloads são navegações para URLs assinadas.

## Configuração e publicação

Após o backup do servidor, executar `python3 deploy/s3/configure-runtime.py`. O script preserva um backup privado do `.env` e define `STORAGE_DRIVER=s3`, `AWS_REGION` e `S3_BUCKET`. Recriar somente o container da API com a imagem validada; não recriar bancos ou volumes. IMDSv2 deve permanecer obrigatório, hop limit 2 para o container (já configurado nesta instância).

Sem `STORAGE_DRIVER=s3`, instalações locais continuam usando disco. Em modo S3, uma falha de configuração ou do provedor retorna erro; não faz fallback silencioso para disco.

O schema de `files` recebe apenas campos adicionais: provedor, bucket/chave internos, tamanho e tipo MIME. Os registros existentes não são migrados ou removidos. Restaurar a imagem e o `.env` anteriores mantém os dados, mas a versão anterior não sabe acessar novos anexos S3; para esses anexos, preferir corrigir a integração em vez de servir a interface antiga.

## Acesso e remoção

`GET /api/files?living_clue_id=ID` filtra os anexos da pista. Listagens e registros não contêm URLs assinadas nem as chaves do bucket. `GET /api/files/ID/access`, autenticado, entrega uma URL GET por cinco minutos, com `Cache-Control: no-store`. O app solicita uma URL nova em cada acesso. Qualquer pessoa com a URL temporária poderá usá-la até expirar; não deve ser tratada como link público permanente.

Leitura segue a permissão atual das pistas (contas autenticadas). Criação e remoção obedecem ao bloqueio do modo Explorador. O download usa Content-Disposition attachment. Formatos ativos como HTML, SVG e executáveis não são aceitos como anexos.

A remoção retira o vínculo do banco e cria um delete marker no S3; versões anteriores permanecem recuperáveis por operação administrativa. Não há política de expiração automática de versões. Restaurar um anexo removido requer recuperar também seu registro a partir do backup do banco. Links externos não removem objetos S3 ou arquivos locais.

## Validação

- `python3 scripts/run-isolated-tests.py --fresh-database --run . --package ./...` usa banco sintético local. Testes S3 usam servidor HTTP de teste e credenciais fictícias.
- `go run ./tools/storage-check` usa apenas bytes sintéticos, confere hash, download assinado 200 e acesso anônimo 403, e remove o objeto de validação. Não abre banco. URLs assinadas e credenciais não são impressas.
- Prévia integrada: `python3 scripts/run-integrations-preview.py --gemini --s3`. `--s3` usa credenciais temporárias da AWS CLI apenas na memória do processo; a role EC2 é usada no servidor. Reiniciar a prévia se a sessão temporária expirar.
- App: selecionar arquivo, enviar, reabrir a pista, acessar o anexo e conferir a remoção; testar rejeição de arquivo acima de 25 MiB, modo Explorador e tela estreita.
