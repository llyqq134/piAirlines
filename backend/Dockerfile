FROM golang:1.26-alpine AS build

WORKDIR /app/backend

RUN apk add --no-cache ca-certificates git

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=build /bin/api /app/api
COPY db/migrations /migrations

EXPOSE 8080
ENV MIGRATIONS_PATH=/migrations

CMD ["/app/api"]

