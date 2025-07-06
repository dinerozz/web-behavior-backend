FROM golang:1.24-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o web-behavior ./cmd/main.go

# ====== RUN STAGE ======
FROM alpine:latest

WORKDIR /root/

ARG TELEGRAM_BOT_TOKEN
ENV TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN
ENV PORT=8080

COPY --from=builder /app/web-behavior .

COPY migrations ./migrations

EXPOSE 8080

CMD ["./web-behavior"]