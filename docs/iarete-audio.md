# Conversa por áudio com a Iareté

## Fluxo

Na conversa da Iareté, **Falar com Iareté** pede acesso ao microfone. **Parar e revisar** mantém a gravação local e permite ouvir, regravar, descartar ou enviar. O limite é de 2 minutos. Nada é enviado antes de **Enviar áudio**.

O envio apresenta imediatamente “Recebi seu áudio. Estou ouvindo…” e um sinal breve, respeitando o som da nave. Após a transcrição, a conversa utiliza o endpoint, o contexto, as permissões e o histórico existentes. A resposta é exibida em texto e pode ser ouvida. **Interromper voz**, silenciar, minimizar o app ou fechar a conversa interrompem o áudio. A escolha de responder por voz fica salva neste dispositivo. O botão **Ouvir resposta** também permite recuperar uma reprodução bloqueada pelo navegador.

Se a transcrição falhar, a gravação fica disponível para tentar novamente. Se o envio do texto falhar, a transcrição também fica disponível; confira o histórico antes de reenviar caso a rede tenha interrompido a confirmação. Se somente a voz falhar, a resposta em texto é preservada e não é reenviada ao modelo.

## Configuração da API

Variáveis em `.env` ou no gerenciador de segredos do servidor, nunca em dart-define, assets ou frontend:

```
OPENAI_API_KEY=<credencial válida>
OPENAI_TRANSCRIBE_MODEL=gpt-transcribe
OPENAI_VOICE_MODEL=gpt-realtime-2.1-mini
OPENAI_VOICE=marin
```

A transcrição usa `POST /v1/audio/transcriptions`. A voz usa a API Realtime por WebSocket exclusivamente para narrar a resposta já gerada pela conversa existente; não substitui o raciocínio, as ferramentas ou o histórico da Iareté. Modelos e voz são configuráveis no servidor. Uma conta sem acesso aos modelos configurados precisa ter a configuração e o acesso ajustados.

Referências verificadas: [transcrição](https://developers.openai.com/api/docs/guides/speech-to-text), [Realtime](https://developers.openai.com/api/docs/guides/realtime-conversations). A documentação atual desaconselha novas integrações com os modelos Speech API legados; por isso a narração utiliza Realtime.

## Endpoints autenticados

- `POST /api/ai/voice/transcriptions`: multipart `file`, WAV PCM16 mono 16kHz, 0,5–120 segundos, até 3.840.044 bytes. Retorna `{ "text": "..." }`. Valida cabeçalho, tamanho real, formato e silêncio antes de chamar o provedor. Não cria mensagens.
- `POST /api/ai/voice/speech`: JSON `{ "message_id": 123 }`. Só aceita mensagem `assistant` da conta autenticada. Retorna `audio/wav`, `Cache-Control: no-store`.

Disponíveis nos modos Explorador e Inventor. Limite global de quatro operações de voz simultâneas, uma por usuário e vinte por minuto por usuário. Erros do provedor são traduzidos sem expor credenciais ou conteúdo de requisições. Sem chave, retorna 503 e a conversa por texto continua disponível.

## Privacidade e plataformas

A API processa gravações em memória, sem criar objetos no S3 ou anexos. O histórico mantém a transcrição e a resposta em texto. A política de processamento do provedor externo também se aplica. No app, os bytes são descartados ao enviar com sucesso, cancelar ou fechar. Apple/Linux exigem arquivo temporário privado para reprodução; ele é removido ao parar, concluir ou fechar. São mantidas no máximo duas respostas de voz em memória enquanto a conversa está aberta.

No navegador, microfone exige HTTPS ou localhost. Permissão negada e bloqueio de autoplay têm recuperação pela interface. Android declara RECORD_AUDIO e solicita permissão no primeiro uso. iOS possui descrição de uso do microfone; a validação em aparelho Apple requer macOS/Xcode.

## Homologação manual

1. Abrir Iareté e conceder acesso ao microfone.
2. Gravar uma pergunta, parar, ouvir e enviar; conferir transcrição, resposta e voz.
3. Gravar e cancelar: nenhum conteúdo deve chegar ao histórico.
4. Desativar respostas por voz, enviar áudio e confirmar resposta somente em texto.
5. Ouvir uma resposta anterior e interromper.
6. Bloquear microfone, simular falha de rede, minimizar e fechar durante gravação/reprodução.
7. Conferir celular vertical/horizontal e navegador com teclado aberto.

Os testes automatizados de protocolo usam provedor simulado, nunca gravações de usuários reais. A homologação com OpenAI real requer credencial válida configurada.

Para uma prévia isolada com credenciais configuradas: `python3 scripts/run-integrations-preview.py --gemini --voice`. Sem `--voice`, a prévia bloqueia chamadas OpenAI mesmo se existir chave no checkout. A suíte isolada também desabilita o provedor externo.
