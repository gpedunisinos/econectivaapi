# Problemáticas vinculadas às pistas

O Delta aceita o provedor nativo `node`, versão 1, com referência canônica `/nodes/graph?focus=ID` (ID positivo, até 15 dígitos). A validação rejeita destinos externos, rotas arbitrárias e parâmetros adicionais. Nenhuma tabela ou permissão nova é necessária.

A busca utiliza o endpoint autenticado `GET /nodes?search=...&limit=20&offset=...`, por nome ou código, com tipo da problemática nos resultados. A navegação utiliza o mapa de nodos existente. O vínculo preserva um título e o ID; não duplica, exclui ou altera o cadastro da problemática. Referências removidas permanecem editáveis na pista e a sala informa sua indisponibilidade.

O vínculo é uma referência interna de navegação. Não é enviado ao leitor de fontes externas da Iareté como se fosse uma página da internet. Blocos antigos e provedores futuros continuam preservados.

Validação automatizada cobre contrato, persistência de Delta misto, edição e remoção de vínculos sem perda de outros blocos, permissões de Inventor/Explorador e exclusão dessas referências da consulta externa de fontes.
