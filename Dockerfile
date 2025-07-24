FROM --platform=linux/amd64 golang:tip-bookworm AS build

WORKDIR /app

COPY go.mod go.sum /app/

COPY migrations /app/migrations
COPY docs /app/docs
COPY src /app/src

RUN go build /app/src/main.go

# now add production version
FROM --platform=linux/amd64 kukymbr/goose-docker:3.24.2 AS production

WORKDIR /app

# Install bash and netcat for the entrypoint script
RUN apk --no-cache add bash netcat-openbsd

COPY --from=build /app/main /app/main
COPY entrypoint.sh /app/entrypoint.sh
COPY --from=build /app/migrations /app/migrations


RUN chmod +x /app/entrypoint.sh

EXPOSE 8000

ENTRYPOINT ["/app/entrypoint.sh"]
