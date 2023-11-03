<h1 align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://github.com/raito-io/raito-io.github.io/raw/master/assets/images/logo-vertical-dark%402x.png">
    <img height="250px" src="https://github.com/raito-io/raito-io.github.io/raw/master/assets/images/logo-vertical%402x.png">
  </picture>
</h1>

<h4 align="center">
  Raito CLI Docker Image
</h4>

<p align="center">
    <a href="/LICENSE.md" target="_blank"><img src="https://img.shields.io/badge/license-Apache%202-brightgreen.svg?label=License" alt="Software License" /></a>
</p>

<hr/>

# Introduction
This Docker image can be used to run the Raito CLI. It automatically keeps it up to date by regularly checking for update and restarting it when an update is available.

# How to run the RAITO CLI container?
The Docker image to use is `ghcr.io/raito-io/raito-cli-runner`. 
The image expects a Raito configuration file mounted to `/config/raito.yml`.

You can eaily start the container using the following command
```bash
docker run --mount type=bind,source="<Your local Raito configuration file>",target="/config/raito.yml",readonly ghcr.io/raito-io/raito-cli-runner:latest
```

Additional environment variables, that could be referred in your Raito configuration file, can be mounted by using the existing docker environment arguments `--env` and `--env-file`.

The following environment variables are used in the default entrypoint:

| Environment variable              | Description                                                                                       | Default Value |
|-----------------------------------|---------------------------------------------------------------------------------------------------|---------------|
| `TZ`                              | Timezone used by the container                                                                    | Etc/UTC       |
| `CLI_CRON`                        | Cron expression that defines when  to execute a sync                                              | `0 2 * * *`   |
| `RAITO_CLI_UPDATE_CRON`           | The cronjob definition for when the container needs to check if a newer CLI version is available. | `0 1 * * *`   |
| `RAITO_CLI_CONTAINER_STDOUT_FILE` | Output file stdout of the Raito CLI                                                               | `/dev/stdout` |
| `RAITO_CLI_CONTAINER_STDERR_FILE` | Output file stderr of the Raito CLI                                                               | `/dev/stderr` |

The default entrypoint of the container is defined as
```dockerfile
ENTRYPOINT /raito-cli-runner run -c "$CLI_CRON" --config-file /config/raito.yml --log-output
```

You can override the default entrypoint by using the `--entrypoint` option when execution `docker run`