package gourgeist

import "go.uber.org/zap"

type Config struct {
	Dexcom        DexcomConfig  `yaml:"dexcom"`
	Discord       DiscordConfig `yaml:"discord"`
	Mongo         MongoConfig   `yaml:"mongo"`
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

type MongoConfig struct {
	URI string `yaml:"uri"`
}
