# Usar a imagem base do Go
FROM golang:1.22-alpine AS builder

# Definir o diretório de trabalho dentro do container
WORKDIR /app

# Copiar o código do projeto para dentro do container
COPY . .

# Inicializar o módulo Go (gera o go.mod e go.sum)
RUN go mod init myapp && go mod tidy

# Compilar a aplicação
RUN go build -o /app/myapp .

# Segunda fase: imagem final com o binário
FROM alpine:latest

# Definir o diretório de trabalho no novo container
WORKDIR /root/

# Copiar o binário da fase de build
COPY --from=builder /app/myapp .

# Expor a porta que a aplicação utiliza (se necessário)
EXPOSE 8080

# Executar a aplicação
CMD ["./myapp"]
