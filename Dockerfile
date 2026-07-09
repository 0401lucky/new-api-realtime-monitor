FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/monitor ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /out/monitor /app/monitor
COPY index.html API文档.md ./
COPY static ./static

ENV PORT=8080
EXPOSE 8080

CMD ["/app/monitor"]
