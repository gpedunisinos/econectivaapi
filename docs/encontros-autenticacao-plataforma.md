# Encontros com autenticação da plataforma

Qualquer conta autenticada na Educação OnLIFE pode iniciar encontros, em modo Explorador ou Inventor. O endpoint protegido `POST /api/me/meetings/access` recebe somente o nome da sala. A identidade e o nome vêm da sessão; não é necessário criar outra conta de organizador.

A API assina JWT HS256 com segredo exclusivo do Jitsi, diferente de `API_SECRET`. O token expira em dez minutos, identifica uma sala exata e não autoriza outras salas. As respostas usam `Cache-Control: no-store`; não há registro de reunião, conversa ou participantes no banco. O convite público não contém token. Caso a entrada expire antes de usar ou numa reconexão tardia, a pessoa deve abrir novamente pela plataforma para receber outra autorização.

O servidor usa autenticação JWT, convidados anônimos e verificação de emissor, audiência, domínio, assinatura e sala. O módulo `token_affiliation` concede moderação a identidades autenticadas cujo token assinado contém `context.user.moderator=true`; convidados não são promovidos automaticamente. Anfitriões são pessoas que iniciam pela plataforma; o link público recebido por terceiros continua sendo um acesso de convidado.

Variáveis protegidas da API: `JITSI_PUBLIC_URL`, `JITSI_APP_ID`, `JITSI_APP_SECRET`, `JITSI_XMPP_DOMAIN`. Segredos nunca vão ao Flutter, ao Git ou ao convite compartilhado. A imagem da API não contém `.env`. O JWT de entrada permanece temporariamente em memória no app e é usado somente na abertura do Jitsi.

`deploy/jitsi/enable-platform-auth.py` prepara a configuração no host existente, verifica que não há conferências nem endpoints ativos e preserva ambientes, configuração Jitsi e proxy em backup privado. Não reinicia serviços; a recriação do Jitsi e da API deve ser feita após compilar e validar a imagem. O proxy de encontros passa a omitir parâmetros e Referer dos logs e envia `Referrer-Policy: no-referrer`. O Nginx da distribuição já mascara parâmetros de token em seus logs. Não ativar logs de depuração de autenticação contendo credenciais.

Reversão: recuperar os ambientes e o virtual host a partir do backup privado, validar Apache e recriar somente os serviços alterados; voltar o frontend à release anterior preservada. Não restaurar banco ou apagar volumes. Chamadas audiovisuais em dispositivos e redes diferentes continuam exigindo validação operacional específica.

Na distribuição instalada, a verificação de sala exige uma exceção explícita para `guest.meet.jitsi`, configurada em `token_verification_allowlist` no componente MUC. Isso não aceita tokens vazios no domínio autenticado: `JWT_ALLOW_EMPTY=0` permanece ativo, a criação de conferência é protegida pelo Jicofo e convidados não recebem moderação.

## Validação e publicação em 21 de outubro de 2026

API de execução `40380fa`, imagem `onlife-api:release-40380fa`; configuração de convidados ajustada no checkpoint `2bd6ab7`. Aplicação web publicada com código `b7d2a57`, release `/var/www/onlife-releases/integrations-20261021T143206Z-b7d2a57`. SHA-256 do arquivo de publicação `41bde2bf2dea35b92cc843c9c3d1fbe00131f871efcee7d75fbfdb9b2ae009f2`; manifesto e três arquivos principais conferidos nos dois domínios públicos.

Todos os pacotes Go passaram em banco sintético. O teste do endpoint inclui uma conta comum em Explorador e Inventor, identidade imposta pela sessão, entrada inválida e indisponibilidade de configuração. A API pública recusou pedido sem sessão com 401. Banco e configurações foram preservados em backup privado; comparação nas 28 tabelas sem registros anteriores removidos ou alterados.

`scripts/validate-meeting-access.py` usa somente a conta sintética da prévia local, pede um JWT à API e verifica pelo protocolo XMPP real: convidado não cria conferência nem entra antes do anfitrião; anfitrião autenticado recebe papel moderator e afiliação owner; convidado entra como participant sem afiliação; token de outra sala existente é recusado; assinatura inválida e token corretamente assinado mas expirado são recusados. Todas as sessões técnicas são encerradas. Não há câmera, microfone nem publicação de conteúdo nesses testes.

Pela interface da prévia, o botão Encontros abriu o diálogo, a API autorizou a sala e a aba Jitsi foi aberta mantendo o cockpit. O navegador interno ficou na configuração de dispositivos; por isso a confirmação de papéis e limites veio do teste de protocolo. Não afirmar que essa rodada homologa mídia em redes e aparelhos distintos. A suíte Flutter completa e a verificação do convite público cobrem o fluxo do app.

Backups privados: `/home/ubuntu/backups/onlife-integrations-20261021T142302Z` preserva o estado anterior ao deploy; `/home/ubuntu/backups/onlife-meeting-jwt-20261021T142414Z` preserva a configuração anterior à ativação JWT; o ajuste posterior de convidados foi preservado também em `onlife-meeting-jwt-20261021T143052Z`. A release web anterior e a imagem `onlife-api:release-c203aca` continuam disponíveis. Não restaurar dados automaticamente em uma reversão de código/configuração.
