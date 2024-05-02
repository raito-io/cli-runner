## Build
FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
ADD constants /app/constants
ADD github /app/github

RUN go build -o raito-cli-runner

## Deploy
FROM alpine:3 as deploy

LABEL org.opencontainers.image.base.name="alpine:3"

RUN apk add --no-cache tzdata

RUN mkdir -p /config

ENV TZ=Etc/UTC
ENV CLI_CRON="0 2 * * *"
ENV RAITO_CLI_UPDATE_CRON="0 1 * * *"
ENV RAITO_CLI_CONTAINER_STDOUT_FILE="/dev/stdout"
ENV RAITO_CLI_CONTAINER_STDERR_FILE="/dev/stderr"
ENV RAITO_CLI_WORKING_DIR="/home/raito/"

RUN addgroup -S raito && adduser -D -S -G raito raito && chmod +w /tmp

RUN chown raito:raito /config

COPY --from=build /app/raito-cli-runner /raito-cli-runner
RUN chown raito:raito /raito-cli-runner


USER raito

ENTRYPOINT /raito-cli-runner run -c "$CLI_CRON" --config-file /config/raito.yml --log-output


## Deploy-amazon
FROM amazon/aws-cli:2.15.10 as amazonlinux

LABEL org.opencontainers.image.base.name="amazon/aws-cli:2.15.10"

RUN yum -y install tzdata jq shadow-utils

RUN mkdir -p /config

ENV TZ=Etc/UTC
ENV CLI_CRON="0 2 * * *"
ENV RAITO_CLI_UPDATE_CRON="0 1 * * *"
ENV RAITO_CLI_CONTAINER_STDOUT_FILE="/dev/stdout"
ENV RAITO_CLI_CONTAINER_STDERR_FILE="/dev/stderr"
ENV RAITO_CLI_WORKING_DIR="/home/raito/"

RUN groupadd -r raito && useradd -d /home/raito -g raito raito

RUN chown raito:raito /config

COPY --from=build /app/raito-cli-runner /raito-cli-runner
RUN chown raito:raito /raito-cli-runner

USER raito

ENTRYPOINT []
CMD /raito-cli-runner run -c "$CLI_CRON" --config-file /config/raito.yml --log-output
