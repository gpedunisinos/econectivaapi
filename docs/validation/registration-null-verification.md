# Correção do cadastro — 01/11/2026

Duas requisições de cadastro às 11h05 (Brasília) retornaram 500. O MySQL recusou `0000-00-00 00:00:00` no campo `email_verified_at`, com erro 1292. O cadastro por e-mail ainda não possui verificação concluída e deve gravar NULL.

A data no modelo User agora é opcional. Cadastros comuns e salvamentos posteriores preservam NULL; cadastros Google, criação administrativa e fixtures que possuem verificação mantêm a data efetiva. Não foi marcado nenhum usuário como verificado para contornar a falha.

Validação:

- `TestRegisterUnverifiedEmailUnderStrictDates`: cadastro HTTP 200 com token, data SQL NULL, salvamento posterior e preservação de data em conta Google, sob `STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE`; transação de teste revertida.
- Testes de cadastro e bloqueio do login por senha para conta Google passaram. Fixtures receberam e-mails únicos para permitir novas execuções no banco local sem duplicidade.
- `go test ./... -run '^$'`: todos os pacotes compilados.
- Swagger regenerado, sem alteração do esquema produzido.

Não foi criada conta nem definida senha em nome do participante em produção. A nova tentativa deve ser feita pelo próprio usuário.
