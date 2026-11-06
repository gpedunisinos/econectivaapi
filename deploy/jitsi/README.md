# Jitsi — Educação Onlife

Preparação isolada para `https://meet.spotec.app`, com Apache mantendo 80/443. A distribuição oficial é `stable-11248` (14/09/2026), arquivo de origem e SHA-256 fixados em `prepare.py`. Consulte o [guia oficial Docker](https://jitsi.github.io/handbook/docs/devops-guide/devops-guide-docker/) para os caminhos desta versão, que usa contêineres sem root e sistema de arquivos somente leitura.

## Preparação

`prepare.py` apenas prepara arquivos e segredos; não publica serviços nem altera DNS/Apache. Executar como usuário UID 1000, em diretório próprio, após preservar o estado existente:

```sh
sudo install -d -o ubuntu -g ubuntu -m 750 /opt/onlife-jitsi
python3 prepare.py --directory /opt/onlife-jitsi --public-ip 54.94.142.29
```

O script recusa substituir `.env` existente, verifica o checksum do pacote oficial, gera senhas distintas e prepara diretórios persistentes graváveis pelo UID dos contêineres. O `.env` recebe permissão `0600`; não colocar seu conteúdo em Git, wiki, logs ou mensagens.

O Compose oficial fica em `release/docker-compose.yml`. O override limita memória/CPU e rotação de logs; não adiciona serviços. Conferir os nomes/portas sem imprimir `docker compose config` completo, que contém senhas resolvidas.

```sh
cd /opt/onlife-jitsi
docker compose --env-file .env -f release/docker-compose.yml -f compose.override.yml -p onlife-jitsi pull
docker compose --env-file .env -f release/docker-compose.yml -f compose.override.yml -p onlife-jitsi up -d
```

Somente `web`, `prosody`, `jicofo` e `jvb`. A interface HTTP fica em `127.0.0.1:8180`; a porta HTTPS interna não é usada pelo proxy e permanece publicada apenas em loopback, 8181. Os endpoints internos de JVB/Jicofo usam 8182/8183 em loopback. A única publicação direta de mídia é `10000/UDP`. XMPP não é exposto no host. Não há Jibri, gravação, transcrição, telefonia nem JWT/SSO. Os limites de memória somam 2.125 GiB; isso limita o processo, não comprova capacidade de participantes.

## Domínio e proxy

Antes de alterar a infraestrutura, registrar o A atual (se existir), a regra de rede e o estado dos endpoints existentes. Criar apenas o A de `meet.spotec.app` e a regra `10000/UDP` necessária; não abrir bancos nem portas de administração.

Usar os virtual hosts separados fornecidos. O HTTP reserva a rota de desafio ACME; o restante redireciona para HTTPS. Emitir o certificado por webroot antes de habilitar o virtual host SSL. Conferir `proxy`, `proxy_http`, `proxy_wstunnel`, `headers`, `rewrite` e `ssl`, habilitando somente os módulos ausentes. Executar `apache2ctl configtest` antes de cada reload. Não substituir os virtual hosts da plataforma.

## Organizadores e convidados

A autenticação prevista é local do Jitsi, com convidados aguardando o organizador. A seleção nominal dos organizadores e a forma de entrega das contas continuam dependendo da definição solicitada ao usuário. O login da plataforma não autentica automaticamente no encontro; não criar esse vínculo por inferência.

Nesta distribuição, o `prosodyctl` usa `/run/prosody/config/prosody.cfg.lua`; a conta de organizador pertence ao domínio XMPP `meet.jitsi`, conforme o guia oficial. Criar contas individualmente após a definição, com senha fornecida por entrada protegida; não copiar senhas de usuários da plataforma nem usar contas do sistema operacional. Não registrar senhas em comandos versionados.

## Validação e retorno

O preparo foi testado localmente com o Compose real: quatro imagens fixadas, sistema de arquivos somente leitura, portas TCP em loopback, autenticação interna e gravação desativada. A segunda execução recusou substituir os segredos existentes. Isso ainda não comprova funcionamento de uma chamada.

Para aceite operacional, testar organizador/convidados, três participantes e redes distintas (inclusive móvel), áudio, vídeo, tela, reconexão e saída. Registrar métricas do host, conectividade UDP, certificado/renovação e os endpoints API/Geo/frontend antes e depois. Acessar a página HTTPS não substitui esses testes.

Se houver regressão, parar somente o projeto `onlife-jitsi` com `docker compose ... down`, **sem `-v`**. Desabilitar apenas os virtual hosts novos, validar e recarregar Apache. Reverter somente o registro DNS e a regra de rede criados para este serviço, usando os snapshots e IDs registrados. Preservar configurações/volumes e os demais stacks. Não restaurar dumps nem configurações globais automaticamente.


## Estado verificado em 21/10/2026

Stack publicado no diretório indicado, com quatro imagens `ghcr.io/jitsi/*:stable-11248`, sistema de arquivos somente leitura e persistência em `config/`. Digests observados:

| Serviço | SHA-256 da imagem | Limite de memória |
|---|---|---|
| web | `615d9882b7bc2bc0b648fb48c153ce2b9c503f37a15e6aaf4f1c8e40f5cf8ca4` | 384 MiB |
| prosody | `a7a58090d93412f7ecd6692330631519833b2e8c8b06ff433089fb54b7149bb2` | 256 MiB |
| jicofo | `c4c1ad331d4d942bc433e5452d75c25758dab7058e3fe07b970ced54cb7f5e47` | 512 MiB |
| jvb | `55cf534aa12931c524835269475c1ea63e5e884a43540f9b318378fd080a335c` | 1.024 MiB |

DNS e HTTPS conferidos. Certificado até 19/01/2027, renovação Certbot em dry-run aprovada. Locale válido da distribuição: `pt-BR`. Novos virtual hosts e módulos mínimos preservados no backup privado `/home/ubuntu/backups/onlife-integrations-20261021T053248Z`, junto dos snapshots/IDs de DNS e Security Group; não publicar esse diretório. A API e os dois nomes do frontend responderam 200 antes/depois.

Visitante sem conta aguardou o anfitrião; uma única tentativa de credenciais fictícias não cadastradas foi rejeitada. Nenhuma conta real de organizador criada. Câmera/microfone desligados; sem conferência no JVB. Posteriormente, reiniciados somente os quatro contêineres Jitsi, sem conferências ativas: saúde do JVB voltou a 200, configuração protegida manteve o hash e os endpoints existentes continuaram em 200. Isso valida retomada do stack; não equivale a testar uma reinicialização completa do host.

Para observabilidade privada, `curl -fsS http://127.0.0.1:8182/about/health` e `curl -fsS -H 'Accept: text/plain' http://127.0.0.1:8182/metrics`. Nesta distribuição a negociação de `/metrics` sem cabeçalho Accept retornou 500; com `text/plain` ou o padrão do curl retornou 200. O endpoint Jicofo de saúde anteriormente tentado respondeu 404; não usar esse retorno como prova de funcionamento. Conexão Jicofo/Prosody/JVB conferida pelos logs sem expor segredos.

Em repouso, uma amostra registrou aproximadamente 15/36/140/163 MiB para web/Prosody/Jicofo/JVB; são valores ociosos, sem validade como limite de capacidade. Os logs Docker têm rotação 10 MiB × 3 por serviço. Não há gravação/transcrição. Responsável operacional nominal, organizadores reais, quantidade de salas esperadas e teste audiovisual com três participantes/redes distintas continuam pendentes de definição/aceite. O acesso do cockpit permanece desabilitado no build publicado até esses critérios serem atendidos.
