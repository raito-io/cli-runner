## Build
FROM golang:1.21-alpine AS build

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
ENV CLI_CRON="0 2 * * *"
ENV RAITO_CLI_UPDATE_CRON="0 1 * * *"
ENV RAITO_CLI_CONTAINER_STDOUT_FILE="/dev/stdout"
ENV RAITO_CLI_CONTAINER_STDERR_FILE="/dev/stderr"
ENV RAITO_CLI_CONTAINER_LIVENESS_FILE="/tmp/liveness_file.lock"

COPY --from=build /raito-cli-runner /raito-cli-runner

ENTRYPOINT /raito-cli-runner run -c "$CLI_CRON" --config-file /config/raito.yml --log-output


## Deploy-amazon
FROM amazon/aws-cli:2.15.10 as amazonlinux

LABEL org.opencontainers.image.base.name="amazon/aws-cli:2.15.10"

RUN yum -y install tzdata jq

WORKDIR /

RUN mkdir -p /config

ENV TZ=Etc/UTC
ENV CLI_CRON="0 2 * * *"
ENV RAITO_CLI_UPDATE_CRON="0 1 * * *"
ENV RAITO_CLI_CONTAINER_STDOUT_FILE="/dev/stdout"
ENV RAITO_CLI_CONTAINER_STDERR_FILE="/dev/stderr"
ENV RAITO_CLI_CONTAINER_LIVENESS_FILE="/tmp/liveness_file.lock"

COPY --from=build /raito-cli-runner /raito-cli-runner

ENTRYPOINT []
CMD /raito-cli-runner run -c "$CLI_CRON" --config-file /config/raito.yml --log-output
