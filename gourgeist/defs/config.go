package defs

import (
	"time"

	"go.uber.org/zap"
)

const DefaultDB = "ichor"

// Intervals.
const (
	LookbackInterval   = -12 * time.Hour
	DownloaderInterval = 1 * time.Minute
	UpdaterInterval    = 1 * time.Minute
	TimeoutInterval    = 2 * time.Second
)

// Channels.
const (
	AlertsChannel  = "alerts"
	ReportsChannel = "reports"
)

type Config struct {
	Dexcom        DexcomConfig  `yaml:"dexcom"`
	Discord       DiscordConfig `yaml:"discord"`
	Mongo         MongoConfig   `yaml:"mongo"`
	Glucose       GlucoseConfig `yaml:"glucose"`
	Alarm         AlarmConfig   `yaml:"alarm"`
	TrevenantAddr string        `yaml:"trevenantAddress"`
	Timezone      string        `yaml:"timezone"`
	Logger        *zap.Logger   `yaml:"_,omitempty"`
}

type DexcomConfig struct {
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
}

type DiscordConfig struct {
	Token string `yaml:"token"`
	Guild int    `yaml:"guild"`
}

type GlucoseConfig struct {
	Low    float64 `yaml:"low"`
	High   float64 `yaml:"high"`
	Target float64 `yaml:"target"`
}

type AlarmConfig struct {
	GlucoseTimeout   int `yaml:"glucoseTimeout"`
	NoInsulinTimeout int `yaml:"noInsulinTimeout"`
}

type MongoConfig struct {
	URI      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
