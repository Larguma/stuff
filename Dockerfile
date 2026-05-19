FROM golang:1.26-alpine AS build

RUN apk add --no-cache build-base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=1 GOOS=linux go build -tags "fts5" \
    -ldflags "-X 'github.com/Larguma/stuff/internal/config.Version=${VERSION}'" \
    -o /out/stuff ./main.go

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build /out/stuff /app/stuff
COPY --from=build /src/web /app/web

RUN mkdir -p /app/data/uploads && chown -R app:app /app/data

USER app

ENV APP_ADDR=:8080
ENV APP_DB_PATH=/app/data/app.db
ENV APP_UPLOAD_DIR=/app/data/uploads
ENV APP_SESSION_SECRET_FILE=/app/data/session_secret
ENV GIN_MODE=release

EXPOSE 8080

CMD ["/app/stuff"]