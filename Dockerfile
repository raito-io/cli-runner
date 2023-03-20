## Build
FROM golang:1.20-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
ADD constants /app/constants
ADD github /app/github

RUN go build -o /raito-cli-runner

## Deploy
FROM alpine:3 as deploy

LABEL org.opencontainers.image.base.name="alpine:3"

RUN apk add --no-cache tzdata

WORKDIR /

RUN mkdir -p /config

ENV TZ=Etc/UTC
ENV CLI_FREQUENCY=60
ENV RAITO_CLI_UPDATE_CRON="0 2 * * *"

COPY --from=build /raito-cli-runner /raito-cli-runner

ENTRYPOINT /raito-cli-runner run -f $CLI_FREQUENCY --config-file /config/raito.yml
