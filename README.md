<div align="center">

# iv2

**A personal management tool for type 1 diabetes.**

_This project is highly experimental, and should only be used as so._

<img src='docs/weekly-overlay.png' align='center' width=500>

</div>

---

## Setup

To get started, a configuration file is required, see [below](#Testing) on how to generate one.

```console
./scripts/restart.sh
```

## Features

**Note: Currently only supports the Dexcom G6 CGM.**

- Real-time glucose plots with customizable thresholds + insulin and carbs intake display
- Generate weekly and monthly reports on performance metrics such as time spent in range
- Customizable alerts for hyper/hypo-glycemia via Discord

## What's Next

- Query by short-form id (better support for mobile use)
- More detailed weekly and monthly reports, compile to PDF for endocrinologists
- Add documentation on bootstrapping a new iv2 instance from scratch
- Train new prediction model using LSTM

## Testing

Includes integration tests that require a running `MongoDB` instance and a Discord bot token.
The following parameters are used by `shuppet` to generate the necessary configuration files.

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
