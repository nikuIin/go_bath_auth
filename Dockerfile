FROM golang:tip-bookworm AS build

WORKDIR /app

COPY go.mod go.sum /app/

COPY migrations /app/migrations
COPY docs /app/docs
COPY src /app/src

RUN CGO_ENABLED=0 go build -o /app/main /app/src/main.go

# now add production version
FROM kukymbr/goose-docker:3.24.2 AS production

WORKDIR /app

# Install bash and netcat for the entrypoint script
RUN apk --no-cache add bash netcat-openbsd

COPY --from=build /app/main /app/main
COPY entrypoint.sh /app/entrypoint.sh
COPY ./migrations /app/migrations/

RUN chmod +x /app/main
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]

CMD ["/app/main"]
