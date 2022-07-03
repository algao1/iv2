# iv2

A management tool for type 1 diabetes.
**This project is highly experimental, and should only be used as so.**

## Setup

A configuration file is required, see [here](#Testing) on how to generate them. If docker-compose is available, then we can run:

```console
./scripts/restart.sh
```

## Features

**Note: Currently only supports the Dexcom G6 CGM.**

- Real-time glucose plots with support for customizable thresholds, and insulin, carbs display
- Customizable hyper/hypo-gylcemia alerts via Discord

## Testing

Includes integration tests that require the use of a running `MongoDB` instance and a Discord bot token. These are used by `shuppet` to generate the necessary configuration files.

```console
go run shuppet/shuppet.go \
  -dexcom-account=$DEXCOM_ACCOUNT \
  -dexcom-password=$DEXCOM_PASSWORD \
  -discord-token=$DISCORD_TOKEN \
  -discord-guild=$DISCORD_GUILD \
  -mongo-username=$MONGO_USERNAME \
  -mongo-password=$MONGO_PASSWORD
```

```console
mv docker-config.yaml config.yaml
./scripts/test.sh
```
