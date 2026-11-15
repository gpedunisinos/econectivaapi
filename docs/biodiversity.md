# Fauna do iNaturalist — UU-105

Endpoints GET autenticados pelo JWT normal do app:

- `/api/biodiversity/nearby?latitude=-29.794&longitude=-51.151&radius_m=1000`
- `/api/biodiversity/observations?north=-29.78&south=-29.81&east=-51.13&west=-51.17`

O proxy usa `ENVIRONMENTAL_CONTEXT_API_URL` e `ENVIRONMENTAL_CONTEXT_API_TOKEN` já existentes.
O segredo da Geo API nunca é enviado ao Flutter. Não há chave iNaturalist ou migração no MySQL.
A Geo API deve ser atualizada primeiro (branch UU-105 no repositório econectivageoapi).

Entorno: raios 500/1000/5000 m; até 12 espécies distintas com foto. Mapa: retângulo
positivo limitado a 3 graus por eixo, até 100 observações. Coordenadas finitas e válidas;
parâmetros adicionais são descartados, sem permitir proxy para destinos arbitrários.
A Geo API mantém cache de 6 h e quota compartilhada, servindo amostras de até 200 registros.
Preservam-se fonte, crédito, licença individual, data e aviso de coordenada aproximada.

Inválidos retornam 400; indisponibilidade/limite da Geo API retorna 503 opcional.
Timeout 20 s, corpo limitado a 2 MiB, cancelamento propagado pela requisição do app.
A falha não altera o contexto socioambiental existente nem impede a leitura de pistas.

Validação: testes unitários de parâmetros e proxy; rotas autenticadas com usuário comum
em MariaDB descartável; teste opt-in `TestFaunaLiveGeoContract` conectando à Geo API local
por `FAUNA_LIVE_GEO_URL` e a uma consulta real e limitada do iNaturalist. Não usa dados
nem usuários de produção. Swagger regenerado com os dois endpoints.
