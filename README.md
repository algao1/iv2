<div align="center">

# iv2

**A personal management tool for type 1 diabetes.**

_Inspired by the likes of [Nightscout](https://github.com/nightscout/cgm-remote-monitor) and [LoopKit](https://github.com/LoopKit/Loop)._

_This project is highly experimental, and should only be used as so._

<img src='docs/weekly-overlay.png' align='center' width=500>

</div>

---

## Setup

To get started, you first need a Discord developer account and create a Bot. A Dexcom account is also required with Dexcom follow enabled.

See the `example-config.yaml` included for further customizations such as:
- Glucose alert thresholds
- Time between alert triggers

Note you'll also need a `.env` file containing the `$MONGO_USERNAME` and `MONGO_PASSWORD`.

Having [Task](https://github.com/go-task/task) installed makes the setup easy, just:

```
task mongo-tools-setup
task proto
task start-all
```

## Features

**Note: iv2 currently only supports the Dexcom G6 CGM.**

- **Real-time** glucose plots with customizable thresholds + insulin and carbs intake display
- Generate weekly and monthly reports on performance metrics such as time spent within range
- Customizable alerts for hyper/hypo-glycemia via Discord

## Why Discord?

- Relatively easy-to-use and customizable cross-platform solution that didn't require writing frontend code
- Dashboard-esque experience (like Datadog) on top of discord; this means (near) realtime updates and **pretty** graphs.

## What's Next

Beyond what was suggested in the previous section, there's a few other things I want to accomplish.

- More detailed weekly and monthly reports, compiled to PDF format for endocrinologists
- Additional documentation on how to bootstrap a new iv2 instance from scratch
- Automatic backups
- Add support for `mg/dL` units
- Train new prediction model using LSTM
