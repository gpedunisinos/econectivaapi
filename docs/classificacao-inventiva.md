# Classificação inventiva: contrato e habilitação

A API existente aceita arrays JSON em `metadata` sem alteração de tabela, coluna ou validação. Os campos reservados são `praticas_inventivas` e `metodologias_inventivas`. O cliente atualizado envia arrays de códigos estáveis e copia os demais metadados antes do PUT. Remoção explícita envia `[]`; null, valores antigos e códigos futuros continuam aceitos.

O PUT substitui o mapa recebido; clientes antigos que serializam listas como texto ou omitem chaves podem perder informações. Editar as classificações requer o app atualizado. Não foi introduzido bloqueio global de versões antigas.

## Habilitação por categoria

Executar no servidor da API:

```sh
python3 deploy/enable-inventive-classifications.py
python3 deploy/enable-inventive-classifications.py --apply
python3 deploy/enable-inventive-classifications.py
```

O primeiro comando é dry run. O segundo acrescenta somente as tags ausentes à categoria ativa chamada **Vivência de Cidadania Digital**; não depende de ID. O terceiro confirma que não há alteração pendente. Nenhum outro campo da categoria é atualizado. Ausência ou duplicidade da categoria interrompe a operação. A escrita compara o valor anterior para não sobrescrever edição concorrente.

Fazer backup e snapshot antes da aplicação. O resultado esperado é exatamente uma mudança em `categories.metadata_tags`, sem classificar qualquer pista existente. Guardar a saída com o valor anterior para eventual reversão pontual (nunca restaurar o banco inteiro por uma configuração).

Para outras categorias, a administração pode acrescentar uma ou ambas as chaves ao CSV `metadata_tags`. O app interpreta as chaves, e não os nomes/IDs das categorias. Remover a configuração não apaga classificações existentes.

## Validação

`TestInventiveClassificationsJSONPersistence` exercita POST/GET/PUT HTTP e persistência real em banco sintético: ambas as listas, seleção múltipla, limpeza explícita, valores desconhecidos e legados, campos JSON adicionais, descrição Quill com integração, coordenadas opcionais e permissões de Explorador/Inventor.

```sh
python3 scripts/run-isolated-tests.py --fresh-database --run TestInventiveClassificationsJSONPersistence
```

Nenhum conteúdo pedagógico é inferido de pistas reais. Esta entrega não exige novo binário da API: o comportamento de persistência já existe. A publicação operacional consiste em atualizar os utilitários/documentação no checkout e aplicar a configuração da categoria, preservando a imagem de runtime já publicada.
