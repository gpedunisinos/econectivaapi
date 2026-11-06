# Validação das integrações — 21/10/2026

## Ambiente e resultado

App `08ab102`, API `0fc9d41`. Testes com banco e conta sintéticos, separados da cópia do servidor e das contas reais. Prévia: http://127.0.0.1:8773/#/living_clues/1, API local 8082. Não foram publicados conteúdos em murais ou mundos de terceiros nem concedidas permissões de câmera/microfone.

- **Flutter:** suíte completa, 94 testes aprovados. Inclui blocos Quill, URLs, preservação de texto/ordem, referências desconhecidas, prevenção de prévia duplicada, navegação, contexto da IA e ciclo de vida do ambiente externo.
- **API:** todos os pacotes com testes aprovados usando `scripts/run-isolated-tests.py --fresh-database --package ./... --run .`. Banco exclusivo `onlife_validation_integrations_review_20261021_120925_464504`; nenhum banco anterior removido. Inclui CRUD autenticado, permissões, contrato versionado, proteção contra clientes antigos, MCP e consulta segura de fontes.
- **Benchmark determinístico:** 19 cenários × 3 execuções = 57 aprovados. Esse modo valida o encadeamento, sem medir o modelo real.
- **Gemini real:** 19 cenários × 3 execuções = **57 aprovados, zero reprovados**; 72 etapas, 87 chamadas MCP, 297566 tokens, 412596 ms. Modelo solicitado e retornado: `gemini-3.8-flash`, sem fallback; prompts `onlife-iarete-4`, ferramentas `onlife-readonly-1`. Revisada uma repetição dos 19 cenários, incluindo as continuações. As fontes do benchmark são fixtures controladas, não consultas a páginas reais dos provedores.

Manifesto SHA-256: `eab762bc0569702d5093796ba2892159690092be7d0d3c0773d12187601f5a33`. Relatório Gemini SHA-256: `3e6e39a69da63df6efc3bb220f12dfe94afc6c20948ddcfb28f033ed1a007daf`. Relatórios completos em arquivos locais protegidos `/tmp/onlife-integracoes-benchmark-{deterministic,gemini}-current.json`, sem credenciais; não versionados.

## Verificações reais pela interface

| Integração | Resultado e limite da evidência |
| --- | --- |
| Genially | Apresentação real carregada; avançou da página 1 para a 2; ampliou, permitiu navegar e voltou à incorporação original preservando a página 2. Editado o título para “Genially — demonstração validada”, salvo e reaberto pela interface. GET autenticado confirmou os três blocos na ordem Padlet/FrameVR/Genially, versão 1, texto autoral com as 18 árvores intacto e ausência de HTML de iframe em `description`. |
| Padlet | Mural oficial de exemplo carregado, lido, ampliado e fechado com retorno à pista. Não houve postagem, cópia do mural ou alteração de privacidade. Contribuição/resposta e conteúdo restrito exigem mural apropriado e autorizado; não homologados por esse teste. |
| FrameVR | Sem iframe antes da entrada; clique criou uma instância e carregou a cena 3D. Abrir origem removeu o iframe interno e exibiu “Ambiente encerrado”; “Entrar novamente” criou uma instância; voltar à pista deixou zero instâncias FrameVR. No navegador interno, a conexão LiveKit falhou em `RTCPeerConnection.setConfiguration`, inclusive fora do iframe; a cena carregada não comprova sessão conectada ou mídia. A continuação no Chrome foi bloqueada por uma janela de outra extensão, com solicitação de liberação enviada ao usuário. |
| Jitsi | HTTPS respondeu 200. Convidado sem organizador ficou em “Esperando o anfitrião”, sem criar conferência. Confirmada desativação persistente da conta técnica temporária; zero conferências/endpoints no Videobridge; o Jicofo foi reiniciado somente nesse estado vazio para remover o cache da autenticação técnica. A prova anterior com três sessões mutadas no mesmo navegador não homologa papéis independentes, áudio/vídeo nem redes distintas. Organizador real, chamada em três redes/dispositivos, compartilhamento de tela e reconexão continuam pendentes. Controle da plataforma segue desabilitado por padrão, ativo apenas na prévia isolada. |
| Iareté / Gemini / MCP | Pergunta real feita pelo chat global na pista reaberta: informou nome correto, 18 árvores e URLs Padlet/FrameVR/Genially; declarou que não lê a apresentação interativa nem vê pessoas no mundo 3D. Benchmark real e testes exercitaram contexto dinâmico, rascunho, autorização, troca de pista, histórico separado, fontes contraditórias/indisponíveis, injeção e orçamento de ferramentas. |
| YouTube | Contrato e preservação cobertos pela suíte; reprodução audiovisual real não foi repetida nesta rodada. |
| Overte / Vircadia | URLs, blocos e preservação cobertos por contrato/testes. Não há cliente/mundo fornecido para homologar entrada e interação reais. Vircadia World não foi tratado como equivalente ao cliente clássico. |
| Metaverse | Continua oculto enquanto o produto não for identificado; não é uma integração operacional homologada. |

## Evidências locais

Capturas não versionadas: `/tmp/onlife-padlet-validado.png`, `/tmp/onlife-genially-validado.png`, `/tmp/onlife-framevr-encerramento-validado.png`, `/tmp/onlife-jitsi-convidado-validado.png`, `/tmp/onlife-iarete-integracoes-validada.png`. Logs locais: `/tmp/onlife-integracoes-flutter-current.log`, `/tmp/onlife-integracoes-go-current.log`, `/tmp/onlife-integracoes-benchmark-gemini-current.log`.

**Conclusão:** os testes automatizados e as verificações funcionais descritas passaram. Isso não constitui homologação integral de todos os provedores, aparelhos, conteúdos privados ou mídia em redes diferentes; as limitações acima permanecem explícitas.
