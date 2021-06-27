# discord-rsvp

This project implements a simple Discord Bot that creates webhook messages in a user-selected channel and provides an update mechanism for these messages.
A message can be updated by buttons that appear under the created messages and allow user input by everyone on the server/guild.

The data model contains events, which have a start timestamp and can be repeated.
Each event then gets one message attached for a game, which are configured globally and not on a per-event basis.

## Installation and Requirements

A Discord [application](https://discord.com/developers/applications) with a configured interactions endpoint URL and an OAuth2 redirect URL.

The URL must follow the following schema:

| Interactions Endpoint URL | `https://example.org` |
| OAuth2 Redirect URL | `https://example.org/webhook-token` |

The recommended way to install this software is using a container.
Either build the application yourself via the `Dockerfile` in this repository or download from the GitHub Container Registry `(TODO)`.

Run the container with `docker run -v /path/to/config/file:/config/config.json -v /path/to/storage:/data/ -p 8080:80 TODO`.
The state should be persistent and is stored in the container under `/data`.
The configuration is accessible in the container via `/config/config.json`.

## Configuration

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

## First Run

Note that on the first run, an invitation link is printed to the logs, which can be used to select the webhook channel this software then proceeds to use.
Only one webhook integration can be active at a time.

Logs can be retrieved via [`docker logs`](https://docs.docker.com/engine/reference/commandline/logs/).
