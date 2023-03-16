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

# Build
```bash
docker build -t raito-cli-runner:latest . 
```

# Run
```bash
docker run --mount type=bind,source="/Users/git/raito/cli-examples/raito-snowflake.yml",target="/config/raito.yml",readonly --env-file .env raito-cli-runner
```
