# Agentes da plataforma vinculados a pistas

O bloco Quill `onlife` aceita `{"v":1,"provider":"agent","url":"/space/agents/123","title":"Nome do agente"}`. A referência é interna, com ID positivo e rota canônica, sem domínio, query string, credenciais ou HTML. Não é um iframe ou uma fonte externa consultável pela Iareté.

A integração não copia instruções, arquivos de conhecimento ou histórico para a pista. A validação de Delta verifica a referência sem consultar o agente: uma remoção posterior não impede salvar o restante da pista. O acesso real é verificado ao abrir a conversa; um agente removido retorna 404.

`GET /api/chat_templates/{id}` consulta o agente diretamente, com a mesma política de leitura já usada pela listagem e pelas conversas. Evita o limite anterior do cliente, que tentava localizar IDs apenas nos primeiros 200 registros. Não altera permissões de edição nem o isolamento do histórico por usuário.

O app permite buscar por nome ou descrição, paginar e selecionar um agente no novo card. O vínculo é exibido abaixo do editor, pode ser trocado ou removido e é salvo junto ao Delta. Na leitura da pista, o card abre a Câmara de Inteligências no agente escolhido; voltar preserva a pista. Não envia mensagens automaticamente.

Validação: contratos de rota, persistência/reabertura junto aos demais blocos, rejeição de rotas externas e consulta de agente fora da primeira página, inclusive após remoção. A suíte isolada usa exclusivamente bancos locais sintéticos. A publicação não exige migração de banco ou configuração adicional.
