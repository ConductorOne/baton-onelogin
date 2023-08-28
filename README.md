# baton-onelogin
`baton-onelogin` is a connector for OneLogin built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the OneLogin API to sync data about users, apps, groups and roles.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## Prerequisites

1. OneLogin account
2. API Credentials with `Manage all` scope. Credentials can be created in Administration panel under `Developers -> API Credentials`


## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-onelogin
baton-onelogin
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_ONELOGIN_CLIENT_ID=clientId BATON_ONELOGIN_CLIENT_SECRET=clientSecret BATON_SUBDOMAIN=subdomain ghcr.io/conductorone/baton-onelogin:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-onelogin/cmd/baton-onelogin@main

BATON_ONELOGIN_CLIENT_ID=clientId BATON_ONELOGIN_CLIENT_SECRET=clientSecret BATON_SUBDOMAIN=subdomain
baton resources
```

# Data Model

`baton-onelogin` pulls down information about the following OneLogin resources:
- Users
- Groups
- Apps
- Roles

# Contributing, Support, and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone. If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-onelogin` Command Line Usage

```
baton-onelogin

Usage:
  baton-onelogin [flags]
  baton-onelogin [command]

Available Commands:
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --client-id string                The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string            The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string                     The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                            help for baton-onelogin
      --log-format string               The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string                The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
      --onelogin-client-id string       OneLogin client ID used to generate the access token. ($BATON_ONELOGIN_CLIENT_ID)
      --onelogin-client-secret string   OneLogin client secret used to generate the access token. ($BATON_ONELOGIN_CLIENT_SECRET)
      --subdomain string                OneLogin subdomain to connect to. ($BATON_SUBDOMAIN)
  -v, --version                         version for baton-onelogin

Use "baton-onelogin [command] --help" for more information about a command.

```
