# AGENTS.md — Task Flow (SPO)

Use este fluxo para executar tarefas com Jira neste projeto.

1. **Ler tarefa Jira**
   - Capturar: chave, escopo, tipo (erro/alteração) e critérios.

2. **Preparar projeto local**
   - Configurar `.env`.
   - Rodar setup do projeto (dependências, migrate/seed quando aplicável).

3. **Criar branch**
   - Formato: `<JIRA>-<slug>`
   - Ex.: `UU-373-ajuste-retorno-api`.

4. **Implementar somente o escopo da tarefa**
   - Não incluir refactors não solicitados.
   - Manter compatibilidade quando possível.
   - Nos controllers, adicionar os comentários com a documentação da API (OpenAPI/Swagger).
   - Quando finalizado, atualizar documentação da API com o comando `swag init --parseDependency --parseInternal`
   - Se falhar por falta de PATH, executar `export PATH=$(go env GOPATH)/bin:$PATH`

5. **Publicar entrega técnica**
   - Fazer commit no formato abaixo
    ### Fixed
    - Descrição do commit. Mencionar também o link da tarefa https://spo-tec.atlassian.net/browse/UU-999
    ### Changed
    - Descrição do commit. Mencionar também o link da tarefa https://spo-tec.atlassian.net/browse/UU-999
    ### Added
    - Descrição do commit. Mencionar também o link da tarefa https://spo-tec.atlassian.net/browse/UU-999

   - Fazer push da branch.
   - Comentar na tarefa Jira com resumo da implementação e branch.
   - Mover a tarefa para **Review**.

6. **Encerramento**
   - Não marcar Done sem validação e definição humana.

## Consulta operacional de logs

- A aplicação grava os logs em `storage/logs`, conforme a configuração do ambiente.
- O acesso remoto só fica habilitado quando `API_LOG_ACCESS_TOKEN` está definido.
- Não incluir tokens, URLs autenticadas ou credenciais reais no repositório.
- Use `GET /api/log/{access_hash}` com o token do ambiente, sem divulgá-lo em documentação pública.
