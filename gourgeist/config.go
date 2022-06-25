package gourgeist

import "go.uber.org/zap"

type Config struct {
	Dexcom        DexcomConfig  `yaml:"dexcom"`
	Discord       DiscordConfig `yaml:"discord"`
	Mongo         MongoConfig   `yaml:"mongo"`
	Glucose       GlucoseConfig `yaml:"glucose"`
	TrevenantAddr string        `yaml:"trevenantAddress"`
	Timezone      string        `yaml:"timezone"`
	Logger        *zap.Logger
}

type DexcomConfig struct {
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
}

type DiscordConfig struct {
	Token string `yaml:"token"`
	Guild string `yaml:"guild"`
}

type GlucoseConfig struct {
	Low    float64 `yaml:"low"`
	High   float64 `yaml:"high"`
	Target float64 `yaml:"target"`
}

type MongoConfig struct {
	URI string `yaml:"uri"`
}
