<h1 align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://github.com/raito-io/raito-io.github.io/raw/master/assets/images/logo-vertical-dark%402x.png">
    <img height="250px" src="https://github.com/raito-io/raito-io.github.io/raw/master/assets/images/logo-vertical%402x.png">
  </picture>
</h1>

<h4 align="center">
  Extensible CLI to easily manage the access controls for your data sources.
</h4>

<p align="center">
    <a href="/LICENSE.md" target="_blank"><img src="https://img.shields.io/badge/license-Apache%202-brightgreen.svg?label=License" alt="Software License" /></a>
</p>

<hr/>

# Introduction
This is a container that will run the RAITO CLI and keep it up to date.

# How to run the RAITO CLI container?
A docker image can be used to run the RAITO CLI runner. The image to use is `ghcr.io/raito-io/raito-cli-runner`. 
The image expect RAITO configuration file mounted to `/config/raito.yml`.

You can eaily start the container using the following command
```bash
docker run --mount type=bind,source="<Your local Raito configuration file>",target="/config/raito.yml",readonly ghcr.io/raito-io/raito-cli-runner:latest
```

Additional environment variables, that could be defined in your Raito configuration file, can be mounted by using the existing docker environment arguments `--env` and `--env-file`.

The following environment variables are used in the default entrypoint:

| Environment variable    | Description                                                                              | Default Value   |
|-------------------------|------------------------------------------------------------------------------------------|-----------------|
| `TZ`                    | Timezone used by the container                                                           | Etc/UTC         |
| `CLI_FREQUENCY`         | The frequency used to do the sync (in minutes).                                          | 60              |
| `RAITO_CLI_UPDATE_CRON` | Cronjob definition when the container need to check if a newer CLI version is available. | `0 2 * * *`     |

The default entrypoint of the container is defined as
```dockerfile
ENTRYPOINT /raito-cli-runner run -f $CLI_FREQUENCY --config-file /config/raito.yml --log-output
```

You can override the default entrypoint by defining it `--entrypoint` option when execution `docker run`