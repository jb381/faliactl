FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/faliactl ./main.go

FROM alpine:3.20

WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /out/faliactl /app/faliactl

EXPOSE 8080

ENTRYPOINT ["/app/faliactl"]
CMD ["serve"]
