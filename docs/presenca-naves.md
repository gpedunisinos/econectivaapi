# Presença temporária e naves

Implementa o plano https://wiki.spotec.app/index.php?curid=928. Uma instância Go mantém sessões em memória, uma identidade opaca por usuário e eventos versionados. Não grava histórico de seções nem altera tabelas. A implantação atual tem uma única instância. Múltiplas réplicas exigem coordenação compartilhada antes de habilitar o recurso.

## Protocolo

`GET /api/presence` faz upgrade WebSocket. Rejeita query strings e origens não autorizadas. A conexão precisa enviar `{"type":"auth","token":"..."}` em até 5 segundos; nenhum evento de presença é entregue antes da validação HS256, expiração e usuário existente. O token permanece no app, nunca é enviado ao iframe ou incluído em URL.

O primeiro retrato é `snapshot {version,self,participants}`. Alterações posteriores são `update {version,participants,removed}`. Participantes têm somente ID opaco, primeiro nome reduzido, destino e atividade genérica. Nomes iguais usam inicial do último sobrenome e, se ainda necessário, ordinal de apresentação sem ID interno. O cliente pode ignorar eventos antigos e reconectar se houver lacuna. Leitores atrasados recebem um novo snapshot.

Frames `state` e `heartbeat` aceitam destino permitido, visibilidade e indicação de interação. Identidade/nome arbitrários são ignorados. Destinos: cockpit, earth, research, observatory, agents e platform. A última interação visível prevalece entre abas; heartbeat não muda sua prioridade. Sem aba visível, conserva o destino e usa “Na plataforma”. A nave local não aparece entre naves remotas, mas entra no contador e na lista como “Você”.

Heartbeat: 20s no app. Expiração: 90s sem mensagem válida; varredura a cada 5s, portanto uma sessão suspensa pode levar até 95s para sumir. `bye` explícito remove a sessão prontamente; fechar uma entre várias abas não remove o usuário. Queda de rede conserva presença temporária com atividade genérica; reconexão não duplica a pessoa. Expiração do JWT encerra a sessão. Reinício da API recompõe presença por reconexão.

Limites: 4KiB por frame, 30 frames por janela de 10s, até 12 sessões por usuário e 512 conexões simultâneas (incluindo autenticação pendente). São contenções iniciais, não capacidade homologada. O limite global usa memória do processo; proteções de borda devem acompanhar um crescimento de uso.

## Ambientes

`PRESENCE_ALLOWED_ORIGINS` é uma lista de origens completas separadas por vírgulas. Quando definida, substitui os padrões. Sem configuração, permite `https://educacaoonlife.gpedu.com.br`, o endereço anterior `https://web.uniusp.gpedu.com.br` e `https://plataforma.atopos.spotec.app`. Os scripts locais usam explicitamente as portas 8772 (cópia local) e 8773 (massa sintética), sem conectar a presença ao servidor oficial.

Usar WSS em produção. Validar suporte a upgrade do Apache; versões modernas de mod_proxy_http podem encaminhar o upgrade na regra HTTP existente. Não reusar o canal Jitsi. Não há dependência de Redis ou da Geo API nesta implantação.

## Validação

Testes do hub cobrem múltiplas abas, arbitragem, nomes iguais, eventos, leitores lentos, limite de sessões, saída e expiração, incluindo execução com detector de corrida. Testes HTTP/WebSocket usam duas identidades distintas em banco sintético, autenticação, origens, rejeição de token em URL e mudança de seção. Testes Flutter cobrem reconexão, ordem de eventos, logout, visibilidade, seção genérica e indisponibilidade. Complementar com validação visual web, teclado, móvel e carga inicial de 30 participantes antes de afirmar capacidade.

Na implantação de 21/10/2026, a regra HTTP genérica retornou 426 ao tentar WSS. Foi necessário adicionar a regra específica `/api/presence` usando `mod_proxy_wstunnel`, já habilitado no servidor. O procedimento reproduzível é `sudo python3 deploy/configure-presence-proxy.py --apply`, com backup do vhost, `apache2ctl configtest`, reload e restauração automática em falha. O teste externo deve confirmar upgrade 101 e rejeição 1008 para token inválido, sem criar presença de teste em produção.
