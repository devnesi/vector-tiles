# Etapa de construção
FROM golang:1.22-alpine AS builder

# Definir o diretório de trabalho dentro do container
WORKDIR /app

# Copiar os arquivos go.mod e go.sum
COPY go.mod go.sum ./

# Baixar as dependências
RUN go mod tidy

# Copiar o restante dos arquivos da aplicação
COPY . .

# Compilar a aplicação
RUN go build -o main .

# Etapa de execução
FROM alpine:latest

# Definir o diretório de trabalho dentro do container
WORKDIR /root/

# Copiar o binário compilado da etapa de construção
COPY --from=builder /app/main .

# Expor a porta em que a aplicação será executada
EXPOSE 8080

# Comando para executar a aplicação
CMD ["./main"]
