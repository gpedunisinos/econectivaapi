# Publicação inicial no GitHub

Esta publicação apresenta o código da API da Educação OnLIFE no repositório `gpedunisinos/econectivaapi`.

A referência de origem é a revisão `f9c7bf4` da branch `main` mantida no Bitbucket. Foi utilizada uma cópia dos arquivos versionados dessa revisão, sem importar o histórico privado, arquivos de ambiente, bancos de dados, uploads ou alterações locais ainda não commitadas.

O repositório do GitHub preserva seu commit inicial e a licença Apache 2.0. A revisão inicial acrescenta os arquivos da API e prepara a configuração de acesso aos logs por `API_LOG_ACCESS_TOKEN`, sem credencial embutida no código. A ausência dessa variável mantém o acesso remoto aos logs desabilitado.

Essa publicação não altera os serviços implantados, o banco de dados nem as configurações de produção. Para futuras implantações desta versão, configure as variáveis do ambiente de destino, inclusive o token de logs se esse recurso for utilizado.

O nome do módulo Go e os imports existentes foram mantidos. Os documentos em `deploy/` são referências de operação e devem ser adaptados ao ambiente antes de executar qualquer implantação.
