# Imagem base
FROM golang:1.23.6-bookworm

# Adiciona informações da pessoa que mantem a imagem
ARG APP_REVISION=unknown
LABEL org.opencontainers.image.revision=$APP_REVISION
LABEL maintainer="Odair Pianta <odair@spotec.app>"

# Dependencia para extracao de texto de PDF (pdftotext)
RUN apt-get update && \
    apt-get install -y --no-install-recommends poppler-utils && \
    rm -rf /var/lib/apt/lists/*

# diretoria um diretorio de trabalho
WORKDIR /app/src/unisinos_usp

# aponta a variavel gopath do go para o diretorio app
ENV GOPATH=/app

# define a variavel de ambiente TZ para America/Sao_Paulo
ENV TZ="America/Sao_Paulo"

# copia os arquivos do projeto para o workdir do container
COPY . /app/src/unisinos_usp

RUN mkdir -p /app/src/unisinos_usp/storage/logs

# execulta o main.go e baixa as dependencias do projeto
RUN test ! -f .env && go build -trimpath -o main .

# Comando para rodar o executavel
ENTRYPOINT ["./main"]

# expose port 8080
EXPOSE 8080
