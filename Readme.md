# discord-rsvp

This project implements a simple Discord Bot that creates webhook messages in a user-selected channel and provides an update mechanism for these messages.
A message can be updated by buttons that appear under the created messages and allow user input by everyone on the server/guild.

The data model contains events, which have a start timestamp and can be repeated.
Each event then gets one message attached for a game, which are configured globally and not on a per-event basis.

## Installation and Requirements

### Requirements

A host for running containers (e.g. docker or podman) that has valid DNS configuration and a HTTPS proxy with valid certificates for the host.

A registered Discord [application](https://discord.com/developers/applications) with a configured interactions endpoint URL and an OAuth2 redirect URL.
The interactions endpoint URL can be set under *General Information* and the OAuth2 Redirect URL can be set under *OAuth2* and then *Redirects*.

The URLs must follow the following schema:

| Setting | Value |
| ------- | ----- |
| Interactions Endpoint URL | `https://example.org` |
| OAuth2 Redirect URL | `https://example.org/webhook-token` |

Note that to set the Interactions Endpoint URL, these steps have to be taken in the specific order:

1. Create a Discord [application](https://discord.com/developers/applications)
2. Copy the values to the config.json (see [configuration](#configuration) for more information)
3. Start the service; Must be accessible via HTTPS with valid certificates
4. Then enter the endpoint URL (see table above); If Discord could not verify the URL, check that the service is reachable via HTTPS
5. Activate the service for a channel (see [First Run](#first-run) for more information)

### Installation

The recommended way to install this software is using a container.
Either build the application yourself via the `Dockerfile` in this repository or download from the [GitHub Container Registry](https://github.com/localthomas/discord-rsvp/pkgs/container/discord-rsvp) (`ghcr.io/localthomas/discord-rsvp`).

Run the container with `docker run -v /path/to/config/file:/config/config.json -v /path/to/storage:/data/ -p 8080:80 ghcr.io/localthomas/discord-rsvp:latest`.
The state should be persistent and is stored in the container under `/data`.
The configuration is accessible in the container via `/config/config.json`.

## Configuration

The apps public key (`HexEncodedDiscordPublicKey`) can be found under [applications](https://discord.com/developers/applications), *General Information* and then *Public Key*.
The value can be copied as is.

The `ClientID` and `clientSecret` can be found under [applications](https://discord.com/developers/applications), *OAuth2* and then *Client Information*.
The values can be copied as is.

An exemplary configuration for the `config.json` file can be found below:

```json
{
    "HexEncodedDiscordPublicKey": "1234567890abcdef",
    "ThisInstanceURL": "https://example.org",
    "ClientID": "1234567890",
    "ClientSecret": "fkgASaFa",
    "Games": {
        "Game1": "Description for [Game1](https://example.org)",
        "Game2": "Description for Game2"
    },
    "Events": {
        "Test-Event": {
            "FirstTime": "2021-06-20T14:31:00+02:00",
            "Repeat": "weekly"
        }
    }
}
```

Values for repeating events can be `weekly`, `daily` and `never`.

## First Run

Note that on the first run, an invitation link is printed to the logs, which can be used to select the webhook channel this software then proceeds to use.
Only one webhook integration can be active at a time.

Logs can be retrieved via [`docker logs`](https://docs.docker.com/engine/reference/commandline/logs/).
